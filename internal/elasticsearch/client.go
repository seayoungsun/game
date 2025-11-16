package elasticsearch

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/kaifa/game-platform/internal/config"
	"github.com/kaifa/game-platform/internal/logger"
	"go.uber.org/zap"
)

var client *elasticsearch.Client

// Init 初始化 Elasticsearch 客户端（必须成功）
func Init(cfg *config.Config) error {
	addresses := cfg.ES.Addresses
	if len(addresses) == 0 {
		return fmt.Errorf("Elasticsearch 地址未配置")
	}

	logger.Logger.Info("正在连接 Elasticsearch...", zap.Strings("addresses", addresses))

	// 创建自定义 Transport，增加超时和重试
	transport := &http.Transport{
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       30 * time.Second,
		DisableKeepAlives:     false,
		ResponseHeaderTimeout: 30 * time.Second, // 增加响应超时
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second, // 连接超时
			KeepAlive: 30 * time.Second,
		}).DialContext,
		// 跳过 TLS 证书验证（用于开发环境，生产环境应该使用正确的证书）
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	cfgClient := elasticsearch.Config{
		Addresses:  addresses,
		MaxRetries: 5, // 增加重试次数
		Transport:  transport,
		// 禁用 SSL 验证（仅用于开发环境，生产环境应该使用正确的证书）
		// 如果需要验证证书，可以配置 CACert
		// CACert: []byte("..."),
	}

	// 如果配置了用户名和密码
	if cfg.ES.Username != "" && cfg.ES.Password != "" {
		cfgClient.Username = cfg.ES.Username
		cfgClient.Password = cfg.ES.Password
		logger.Logger.Debug("使用 Elasticsearch 认证", zap.String("username", cfg.ES.Username))
	}

	// 创建客户端（带重试）
	var err error
	var lastErr error
	maxAttempts := 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		client, err = elasticsearch.NewClient(cfgClient)
		if err == nil {
			break
		}
		lastErr = err
		logger.Logger.Warn("创建 Elasticsearch 客户端失败，正在重试",
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", maxAttempts),
			zap.Error(err))
		if attempt < maxAttempts {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	if err != nil {
		return fmt.Errorf("创建 Elasticsearch 客户端失败（重试 %d 次后）: %w", maxAttempts, lastErr)
	}

	// 测试连接（使用带超时的上下文，带重试）
	var ctx context.Context
	var cancel context.CancelFunc

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// 每次重试都创建新的上下文
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)

		res, err := client.Info(client.Info.WithContext(ctx))
		if err == nil && !res.IsError() {
			// 验证响应
			var info map[string]interface{}
			if err := json.NewDecoder(res.Body).Decode(&info); err != nil {
				res.Body.Close()
				cancel()
				return fmt.Errorf("无法解析 Elasticsearch 信息响应: %w", err)
			}
			res.Body.Close()

			// 提取版本信息
			var version string
			if v, ok := info["version"].(map[string]interface{}); ok {
				if num, ok := v["number"].(string); ok {
					version = num
				}
			}

			logger.Logger.Info("Elasticsearch 连接成功",
				zap.Strings("addresses", addresses),
				zap.String("version", version))
			cancel()
			return nil
		}

		// 处理错误
		if res != nil {
			if res.IsError() {
				bodyBytes, _ := io.ReadAll(res.Body)
				res.Body.Close()
				lastErr = fmt.Errorf("Elasticsearch 连接错误: %s, 响应: %s", res.Status(), string(bodyBytes))
			} else {
				res.Body.Close()
			}
		} else {
			lastErr = err
		}

		cancel() // 释放当前上下文

		logger.Logger.Warn("Elasticsearch 连接失败，正在重试",
			zap.Int("attempt", attempt),
			zap.Int("max_attempts", maxAttempts),
			zap.Error(lastErr))

		if attempt < maxAttempts {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	return fmt.Errorf("连接 Elasticsearch 失败（重试 %d 次后）: %w", maxAttempts, lastErr)
}

// GetClient 获取 Elasticsearch 客户端
func GetClient() *elasticsearch.Client {
	return client
}

// IndexOperationLog 索引操作日志到 Elasticsearch
func IndexOperationLog(logData map[string]interface{}) error {
	if client == nil {
		return nil // 如果未初始化，静默失败
	}

	// 生成索引名称（按日期）
	indexName := fmt.Sprintf("admin-operation-logs-%s", time.Now().Format("2006.01.02"))

	// 确保有 @timestamp 字段
	if logData["@timestamp"] == nil {
		logData["@timestamp"] = time.Now().Format(time.RFC3339)
	}

	// 转换为 JSON
	body, err := json.Marshal(logData)
	if err != nil {
		return fmt.Errorf("序列化日志数据失败: %w", err)
	}

	// 创建索引请求（使用 true 而不是 wait_for，因为 wait_for 在某些版本可能不支持）
	req := esapi.IndexRequest{
		Index:      indexName,
		DocumentID: "", // 让 ES 自动生成 ID
		Body:       strings.NewReader(string(body)),
		Refresh:    "true", // 同步刷新，确保数据立即可查询（相比 wait_for 更兼容）
	}

	// 执行请求（使用带超时的上下文）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := req.Do(ctx, client)
	if err != nil {
		return fmt.Errorf("索引日志失败: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		// 读取错误响应
		var errorResp map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&errorResp); err == nil {
			logger.Logger.Error("Elasticsearch 索引错误",
				zap.String("status", res.Status()),
				zap.Any("error", errorResp),
				zap.String("index", indexName))
			return fmt.Errorf("Elasticsearch 错误: %s, 详情: %v", res.Status(), errorResp)
		}
		// 如果无法解析错误响应，读取响应体
		bodyBytes, _ := io.ReadAll(res.Body)
		logger.Logger.Error("Elasticsearch 索引错误",
			zap.String("status", res.Status()),
			zap.String("body", string(bodyBytes)),
			zap.String("index", indexName))
		return fmt.Errorf("Elasticsearch 错误: %s, 响应: %s", res.Status(), string(bodyBytes))
	}

	// 验证响应（可选，记录成功信息）
	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err == nil {
		logger.Logger.Debug("Elasticsearch 写入成功",
			zap.String("index", indexName),
			zap.Any("result", result))
	}

	return nil
}

// SearchOperationLogs 搜索操作日志
func SearchOperationLogs(query map[string]interface{}, from, size int) ([]map[string]interface{}, int64, error) {
	if client == nil {
		return nil, 0, fmt.Errorf("Elasticsearch 未初始化")
	}

	// 生成索引名称（最近30天的索引，扩大范围）
	indices := []string{}
	now := time.Now()
	for i := 0; i < 30; i++ {
		date := now.AddDate(0, 0, -i).Format("2006.01.02")
		indices = append(indices, fmt.Sprintf("admin-operation-logs-%s", date))
	}

	// 构建查询
	searchQuery := map[string]interface{}{
		"from": from,
		"size": size,
		"sort": []map[string]interface{}{
			{
				"@timestamp": map[string]interface{}{
					"order": "desc",
				},
			},
		},
		"query": query,
	}

	queryBody, err := json.Marshal(searchQuery)
	if err != nil {
		return nil, 0, fmt.Errorf("序列化查询失败: %w", err)
	}

	// 执行搜索
	res, err := client.Search(
		client.Search.WithIndex(strings.Join(indices, ",")),
		client.Search.WithBody(strings.NewReader(string(queryBody))),
		client.Search.WithContext(context.Background()),
		client.Search.WithIgnoreUnavailable(true), // 忽略不存在的索引
	)
	if err != nil {
		logger.Logger.Error("Elasticsearch 搜索请求失败", zap.Error(err), zap.Strings("indices", indices))
		return nil, 0, fmt.Errorf("搜索失败: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		// 读取错误响应
		var errorResp map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&errorResp); err == nil {
			logger.Logger.Error("Elasticsearch 搜索错误", zap.String("status", res.Status()), zap.Any("error", errorResp))
			return nil, 0, fmt.Errorf("Elasticsearch 搜索错误: %s, 详情: %v", res.Status(), errorResp)
		}
		logger.Logger.Error("Elasticsearch 搜索错误", zap.String("status", res.String()))
		return nil, 0, fmt.Errorf("Elasticsearch 搜索错误: %s", res.String())
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		logger.Logger.Error("解析 Elasticsearch 响应失败", zap.Error(err))
		return nil, 0, fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查是否有 hits 字段
	hits, ok := result["hits"].(map[string]interface{})
	if !ok {
		logger.Logger.Warn("Elasticsearch 响应中没有 hits 字段", zap.Any("result", result))
		return []map[string]interface{}{}, 0, nil
	}

	// 提取总数
	var total int64
	if totalObj, ok := hits["total"]; ok {
		switch v := totalObj.(type) {
		case float64:
			total = int64(v)
		case map[string]interface{}:
			if value, ok := v["value"].(float64); ok {
				total = int64(value)
			}
		}
	}

	// 提取结果列表
	hitsArray, ok := hits["hits"].([]interface{})
	if !ok {
		logger.Logger.Debug("Elasticsearch 查询无结果", zap.Strings("indices", indices), zap.Int64("total", total))
		return []map[string]interface{}{}, total, nil
	}

	logs := make([]map[string]interface{}, 0, len(hitsArray))
	for _, hit := range hitsArray {
		hitMap, ok := hit.(map[string]interface{})
		if !ok {
			continue
		}
		source, ok := hitMap["_source"].(map[string]interface{})
		if !ok {
			continue
		}
		logs = append(logs, source)
	}

	logger.Logger.Debug("Elasticsearch 查询成功", zap.Int64("total", total), zap.Int("returned", len(logs)))
	return logs, total, nil
}

// BuildQuery 构建查询条件
func BuildQuery(filters map[string]interface{}) map[string]interface{} {
	mustQueries := []map[string]interface{}{}

	// 管理员ID过滤
	if adminID, ok := filters["admin_id"]; ok && adminID != nil {
		mustQueries = append(mustQueries, map[string]interface{}{
			"term": map[string]interface{}{
				"admin_id": adminID,
			},
		})
	}

	// 模块过滤
	if module, ok := filters["module"]; ok && module != nil && module != "" {
		mustQueries = append(mustQueries, map[string]interface{}{
			"term": map[string]interface{}{
				"module": module,
			},
		})
	}

	// 状态过滤
	if status, ok := filters["status"]; ok && status != nil {
		mustQueries = append(mustQueries, map[string]interface{}{
			"term": map[string]interface{}{
				"status": status,
			},
		})
	}

	// IP 搜索
	if ip, ok := filters["ip"]; ok && ip != nil && ip != "" {
		mustQueries = append(mustQueries, map[string]interface{}{
			"wildcard": map[string]interface{}{
				"ip": "*" + ip.(string) + "*",
			},
		})
	}

	// 路径搜索
	if path, ok := filters["path"]; ok && path != nil && path != "" {
		mustQueries = append(mustQueries, map[string]interface{}{
			"wildcard": map[string]interface{}{
				"path": "*" + path.(string) + "*",
			},
		})
	}

	// 时间范围查询
	if startTime, ok := filters["start_time"]; ok && startTime != nil {
		timestampRange := make(map[string]interface{})
		timestampRange["gte"] = startTime
		if endTime, ok := filters["end_time"]; ok && endTime != nil {
			timestampRange["lte"] = endTime
		}
		if len(timestampRange) > 0 {
			mustQueries = append(mustQueries, map[string]interface{}{
				"range": map[string]interface{}{
					"@timestamp": timestampRange,
				},
			})
		}
	}

	// 全文搜索
	if search, ok := filters["search"]; ok && search != nil && search != "" {
		mustQueries = append(mustQueries, map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":  search,
				"fields": []string{"admin_name", "module", "action", "path", "request", "response", "error_msg"},
			},
		})
	}

	// 如果没有查询条件，返回 match_all
	if len(mustQueries) == 0 {
		return map[string]interface{}{
			"match_all": map[string]interface{}{},
		}
	}

	return map[string]interface{}{
		"bool": map[string]interface{}{
			"must": mustQueries,
		},
	}
}
