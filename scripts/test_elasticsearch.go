package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/kaifa/game-platform/internal/config"
	"github.com/kaifa/game-platform/internal/logger"
)

func main() {
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🔍 Elasticsearch 连接测试工具")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// 初始化日志
	logCfg := config.LogConfig{
		Level:      "debug",
		OutputPath: "./logs",
		MaxSize:    100,
		MaxBackups: 7,
		MaxAge:     30,
	}
	if err := logger.InitLogger(logCfg); err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// 加载配置
	cfg, err := config.Load("")
	if err != nil {
		fmt.Printf("❌ 加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 检查配置
	if len(cfg.ES.Addresses) == 0 {
		fmt.Println("❌ Elasticsearch 地址未配置")
		os.Exit(1)
	}

	fmt.Printf("📍 Elasticsearch 地址: %v\n", cfg.ES.Addresses)
	if cfg.ES.Username != "" {
		fmt.Printf("👤 用户名: %s\n", cfg.ES.Username)
		fmt.Printf("🔐 密码: %s\n", maskPassword(cfg.ES.Password))
	}
	fmt.Println()

	// 测试每个地址
	for i, address := range cfg.ES.Addresses {
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("测试连接 #%d: %s\n", i+1, address)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

		if err := testConnection(address, cfg.ES.Username, cfg.ES.Password); err != nil {
			fmt.Printf("❌ 连接失败: %v\n", err)
			fmt.Println()
			continue
		}

		fmt.Printf("✅ 连接成功！\n")
		fmt.Println()
	}
}

func testConnection(address, username, password string) error {
	// 解析 URL
	parsedURL, err := url.Parse(address)
	if err != nil {
		return fmt.Errorf("无效的 URL: %w", err)
	}

	host := parsedURL.Host
	if host == "" {
		return fmt.Errorf("URL 中没有主机信息")
	}

	// 测试 1: DNS 解析
	fmt.Println("1️⃣  测试 DNS 解析...")
	addrs, err := net.LookupHost(parsedURL.Hostname())
	if err != nil {
		return fmt.Errorf("DNS 解析失败: %w", err)
	}
	fmt.Printf("   ✅ 解析到 IP: %v\n", addrs)

	// 测试 2: TCP 连接
	fmt.Println("2️⃣  测试 TCP 连接...")
	conn, err := net.DialTimeout("tcp", host, 5*time.Second)
	if err != nil {
		return fmt.Errorf("TCP 连接失败: %w", err)
	}
	conn.Close()
	fmt.Printf("   ✅ TCP 连接成功\n")

	// 测试 3: HTTP/HTTPS 连接（不使用认证）
	fmt.Println("3️⃣  测试 HTTP/HTTPS 连接（无认证）...")
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ResponseHeaderTimeout: 10 * time.Second,
			DisableKeepAlives:     false,
			// 跳过 TLS 证书验证（用于开发环境）
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	testURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)
	resp, err := client.Get(testURL)
	if err != nil {
		fmt.Printf("   ❌ HTTP 连接失败: %v\n", err)
		fmt.Printf("   💡 提示: 这可能是由于:\n")
		fmt.Printf("      - Elasticsearch 服务未运行\n")
		fmt.Printf("      - 防火墙阻止连接\n")
		fmt.Printf("      - SSL/TLS 配置问题（如果使用 HTTPS）\n")
		return fmt.Errorf("HTTP 连接失败: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("   ✅ 状态码: %d\n", resp.StatusCode)
	if resp.StatusCode == 401 {
		fmt.Println("   ⚠️  需要认证")
		fmt.Printf("   📝 响应: %s\n", string(body))
	}

	// 测试 4: 使用认证的 Info API
	if username != "" && password != "" {
		fmt.Println("4️⃣  测试 Info API（使用认证）...")
		infoURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

		req, err := http.NewRequest("GET", infoURL, nil)
		if err != nil {
			return fmt.Errorf("创建请求失败: %w", err)
		}
		req.SetBasicAuth(username, password)

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("Info API 请求失败: %w", err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("   状态码: %d\n", resp.StatusCode)

		if resp.StatusCode == 200 {
			var info map[string]interface{}
			if err := json.Unmarshal(body, &info); err == nil {
				if version, ok := info["version"].(map[string]interface{}); ok {
					if number, ok := version["number"].(string); ok {
						fmt.Printf("   ✅ Elasticsearch 版本: %s\n", number)
					}
				}
				if clusterName, ok := info["cluster_name"].(string); ok {
					fmt.Printf("   ✅ 集群名称: %s\n", clusterName)
				}
			}
		} else {
			fmt.Printf("   ⚠️  响应: %s\n", string(body))
			if resp.StatusCode == 401 {
				return fmt.Errorf("认证失败，请检查用户名和密码")
			}
		}

		// 测试 5: 集群健康检查
		fmt.Println("5️⃣  测试集群健康状态...")
		healthURL := fmt.Sprintf("%s://%s/_cluster/health", parsedURL.Scheme, parsedURL.Host)
		req, err = http.NewRequest("GET", healthURL, nil)
		if err == nil {
			req.SetBasicAuth(username, password)
			resp, err := client.Do(req)
			if err == nil {
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				if resp.StatusCode == 200 {
					var health map[string]interface{}
					if err := json.Unmarshal(body, &health); err == nil {
						if status, ok := health["status"].(string); ok {
							fmt.Printf("   ✅ 集群状态: %s\n", status)
						}
					}
				}
			}
		}
	} else {
		fmt.Println("4️⃣  跳过认证测试（未配置用户名/密码）")
	}

	return nil
}

func maskPassword(password string) string {
	if len(password) == 0 {
		return "(未设置)"
	}
	if len(password) <= 4 {
		return "****"
	}
	return password[:2] + "****" + password[len(password)-2:]
}
