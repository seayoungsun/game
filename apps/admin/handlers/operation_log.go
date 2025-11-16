package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kaifa/game-platform/internal/database"
	esClient "github.com/kaifa/game-platform/internal/elasticsearch"
	"github.com/kaifa/game-platform/internal/logger"
	"github.com/kaifa/game-platform/pkg/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// GetOperationLogs 获取操作日志列表
// 优先从 Elasticsearch 查询，如果 ES 不可用则从 MySQL 查询
func GetOperationLogs(c *gin.Context) {
	// 检查是否强制使用 MySQL（用于兼容性）
	useMySQL := c.Query("use_mysql") == "true"

	if !useMySQL && esClient.GetClient() != nil {
		// 尝试从 Elasticsearch 查询
		logs, total, err := getLogsFromES(c)
		if err == nil {
			logger.Logger.Debug("从 Elasticsearch 查询成功", zap.Int64("total", total))
			c.JSON(http.StatusOK, gin.H{
				"code": 200,
				"data": gin.H{
					"list":  logs,
					"total": total,
				},
			})
			return
		}
		// ES 查询失败，记录错误并降级到 MySQL
		logger.Logger.Warn("Elasticsearch 查询失败，降级到 MySQL", zap.Error(err))
	}

	// 从 MySQL 查询（降级方案或强制使用）
	getLogsFromMySQL(c)
}

// getLogsFromES 从 Elasticsearch 查询日志
func getLogsFromES(c *gin.Context) ([]map[string]interface{}, int64, error) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	from := (page - 1) * pageSize

	// 构建查询条件
	filters := make(map[string]interface{})

	if adminID := c.Query("admin_id"); adminID != "" {
		if id, err := strconv.ParseUint(adminID, 10, 64); err == nil {
			filters["admin_id"] = id
		}
	}

	if module := c.Query("module"); module != "" {
		filters["module"] = module
	}

	if status := c.Query("status"); status != "" {
		if s, err := strconv.Atoi(status); err == nil {
			filters["status"] = s
		}
	}

	if ip := c.Query("ip"); ip != "" {
		filters["ip"] = ip
	}

	if path := c.Query("path"); path != "" {
		filters["path"] = path
	}

	if search := c.Query("search"); search != "" {
		filters["search"] = search
	}

	// 时间范围
	if dateStart := c.Query("date_start"); dateStart != "" {
		if ts, err := strconv.ParseInt(dateStart, 10, 64); err == nil {
			filters["start_time"] = time.Unix(ts, 0).Format(time.RFC3339)
		}
	}
	if dateEnd := c.Query("date_end"); dateEnd != "" {
		if ts, err := strconv.ParseInt(dateEnd, 10, 64); err == nil {
			filters["end_time"] = time.Unix(ts, 0).Format(time.RFC3339)
		}
	}

	// 构建查询
	query := esClient.BuildQuery(filters)

	// 执行搜索
	logs, total, err := esClient.SearchOperationLogs(query, from, pageSize)
	if err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// getLogsFromMySQL 从 MySQL 查询日志（降级方案）
func getLogsFromMySQL(c *gin.Context) {
	var logs []models.AdminOperationLog
	var total int64

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	query := database.DB.Model(&models.AdminOperationLog{})

	// 搜索条件
	if adminID := c.Query("admin_id"); adminID != "" {
		query = query.Where("admin_id = ?", adminID)
	}
	if module := c.Query("module"); module != "" {
		query = query.Where("module = ?", module)
	}
	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if dateStart := c.Query("date_start"); dateStart != "" {
		if ts, err := strconv.ParseInt(dateStart, 10, 64); err == nil {
			query = query.Where("created_at >= ?", ts)
		}
	}
	if dateEnd := c.Query("date_end"); dateEnd != "" {
		if ts, err := strconv.ParseInt(dateEnd, 10, 64); err == nil {
			query = query.Where("created_at <= ?", ts)
		}
	}

	// 获取总数
	query.Count(&total)

	// 获取列表
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取操作日志失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":  logs,
			"total": total,
		},
	})
}

// GetOperationLog 获取操作日志详情
func GetOperationLog(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的日志ID",
		})
		return
	}

	var log models.AdminOperationLog
	if err := database.DB.First(&log, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "日志不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "获取日志详情失败: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": log,
	})
}

// DeleteOperationLog 删除操作日志
func DeleteOperationLog(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "无效的日志ID",
		})
		return
	}

	if err := database.DB.Delete(&models.AdminOperationLog{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "删除日志失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "删除成功",
	})
}

// BatchDeleteOperationLogs 批量删除操作日志
func BatchDeleteOperationLogs(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	if err := database.DB.Where("id IN ?", req.IDs).Delete(&models.AdminOperationLog{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "批量删除失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "删除成功",
	})
}

// CleanOldLogs 清理旧日志
func CleanOldLogs(c *gin.Context) {
	var req struct {
		Days int `json:"days" binding:"required"` // 保留最近N天的日志
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	if req.Days < 1 {
		req.Days = 30 // 默认保留30天
	}

	// 计算时间戳
	cutoffTime := time.Now().AddDate(0, 0, -req.Days).Unix()

	var count int64
	database.DB.Model(&models.AdminOperationLog{}).Where("created_at < ?", cutoffTime).Count(&count)

	if err := database.DB.Where("created_at < ?", cutoffTime).Delete(&models.AdminOperationLog{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "清理日志失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "清理成功",
		"data": gin.H{
			"deleted_count": count,
		},
	})
}
