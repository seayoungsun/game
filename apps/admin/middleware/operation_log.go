package middleware

import (
	"bytes"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kaifa/game-platform/internal/database"
	esClient "github.com/kaifa/game-platform/internal/elasticsearch"
	"github.com/kaifa/game-platform/internal/logger"
	"github.com/kaifa/game-platform/pkg/models"
	"go.uber.org/zap"
)

// OperationLogMiddleware 操作日志中间件
func OperationLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 跳过健康检查和静态文件
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/favicon.ico" {
			c.Next()
			return
		}

		startTime := time.Now()

		// 获取管理员信息
		adminID, exists := c.Get("admin_id")
		if !exists {
			c.Next()
			return
		}

		// 尝试获取管理员用户名（支持多种键名）
		adminNameStr := ""
		if adminName, ok := c.Get("admin_username"); ok && adminName != nil {
			adminNameStr = adminName.(string)
		} else if adminName, ok := c.Get("username"); ok && adminName != nil {
			adminNameStr = adminName.(string)
		}

		// 如果仍然没有用户名，尝试从数据库查询
		if adminNameStr == "" {
			var admin models.Admin
			if err := database.DB.First(&admin, adminID.(uint)).Error; err == nil {
				adminNameStr = admin.Username
			}
		}

		// 读取请求体（用于记录请求参数）
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// 创建响应写入器包装器
		responseWriter := &responseWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = responseWriter

		// 执行下一步
		c.Next()

		// 计算耗时
		duration := time.Since(startTime).Milliseconds()

		// 解析模块和动作
		module, action := parseModuleAndAction(c.Request.URL.Path, c.Request.Method)

		// 记录操作日志
		log := models.AdminOperationLog{
			AdminID:   adminID.(uint),
			AdminName: adminNameStr,
			Module:    module,
			Action:    action,
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			IP:        c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			Request:   string(requestBody),
			Response:  responseWriter.body.String(),
			Status:    1,
			Duration:  duration,
			CreatedAt: time.Now().Unix(),
		}

		// 如果状态码不是2xx，记录为失败
		if c.Writer.Status() >= 400 {
			log.Status = 2
			log.ErrorMsg = responseWriter.body.String()
			// 限制错误信息长度
			if len(log.ErrorMsg) > 500 {
				log.ErrorMsg = log.ErrorMsg[:500]
			}
		}

		// 限制请求和响应长度
		if len(log.Request) > 2000 {
			log.Request = log.Request[:2000] + "..."
		}
		if len(log.Response) > 2000 {
			log.Response = log.Response[:2000] + "..."
		}

		// 异步保存日志到 MySQL 和 Elasticsearch
		go func() {
			// 先保存到 MySQL，获取 ID
			if err := database.DB.Create(&log).Error; err != nil {
				// MySQL 保存失败，记录错误但不影响请求
				return
			}

			// 同时保存到 Elasticsearch（静默失败，不影响主流程）
			logData := map[string]interface{}{
				"id":         log.ID,
				"admin_id":   log.AdminID,
				"admin_name": log.AdminName,
				"module":     log.Module,
				"action":     log.Action,
				"method":     log.Method,
				"path":       log.Path,
				"ip":         log.IP,
				"user_agent": log.UserAgent,
				"request":    log.Request,
				"response":   log.Response,
				"status":     log.Status,
				"error_msg":  log.ErrorMsg,
				"duration":   log.Duration,
				"created_at": log.CreatedAt,
				"@timestamp": time.Unix(log.CreatedAt, 0).Format(time.RFC3339),
			}
			if err := esClient.IndexOperationLog(logData); err != nil {
				// 记录 ES 写入失败，但不影响主流程
				logger.Logger.Debug("Elasticsearch 写入失败", zap.Error(err), zap.Uint("log_id", log.ID))
			}
		}()
	}
}

// responseWriter 响应写入器包装器
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// parseModuleAndAction 解析模块和动作
func parseModuleAndAction(path, method string) (module, action string) {
	// 根据路径解析模块
	prefix := "/api/v1/admin/"
	if len(path) < len(prefix) {
		return "unknown", "unknown"
	}

	// 提取 /api/v1/admin/ 之后的部分
	rest := path[len(prefix):]
	if rest == "" {
		return "unknown", "unknown"
	}

	// 移除开头的斜杠
	if len(rest) > 0 && rest[0] == '/' {
		rest = rest[1:]
	}

	// 如果路径为空，返回unknown
	if rest == "" {
		return "unknown", "unknown"
	}

	// 使用 strings.Split 分割路径
	parts := strings.Split(rest, "/")
	moduleName := parts[0]

	// 模块名称映射（用于显示中文名称）
	moduleMap := map[string]string{
		"users":             "用户管理",
		"roles":             "角色管理",
		"admins":            "管理员管理",
		"recharge-orders":   "充值订单",
		"withdraw-orders":   "提现订单",
		"deposit-addresses": "充值地址",
		"payments":          "支付管理",
		"dashboard":         "仪表盘",
		"permissions":       "权限管理",
		"operation-logs":    "操作日志",
		"system-configs":    "系统设置",
		"profile":           "个人中心",
	}

	// 获取模块名称（如果不在映射中，返回原始名称）
	if mappedName, ok := moduleMap[moduleName]; ok {
		module = mappedName
	} else {
		module = moduleName
	}

	// 根据HTTP方法和路径结构解析动作
	switch method {
	case "GET":
		// 如果路径正好是模块路径，或者后面还有路径，判断是列表还是详情
		if len(parts) == 1 {
			// 例如: /api/v1/admin/users -> list
			action = "list"
		} else {
			// 例如: /api/v1/admin/users/123 -> detail
			// 例如: /api/v1/admin/dashboard/stats -> detail
			action = "detail"
		}
	case "POST":
		// 特殊处理：某些POST请求可能是其他操作
		if len(parts) > 1 {
			lastPart := parts[len(parts)-1]
			if lastPart == "audit" || lastPart == "batch-delete" || lastPart == "clean" || lastPart == "collect" || lastPart == "batch-collect" {
				action = lastPart
			} else {
				action = "create"
			}
		} else {
			action = "create"
		}
	case "PUT", "PATCH":
		action = "update"
	case "DELETE":
		action = "delete"
	default:
		action = method
	}

	return module, action
}
