package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kaifa/game-platform/internal/config"
	"github.com/kaifa/game-platform/pkg/utils"
)

var (
	targetConnections = flag.Int("connections", 1000, "目标连接数")
	concurrent        = flag.Int("concurrent", 100, "并发连接数")
	baseURL           = flag.String("url", "ws://localhost:8081/ws", "WebSocket服务器基础地址（不含token）")
	duration          = flag.Duration("duration", 60*time.Second, "测试持续时间")
	useSameToken      = flag.Bool("same-token", false, "是否使用同一个token（用于测试单点登录）")
)

func main() {
	flag.Parse()

	// 加载配置（用于生成token）
	_, err := config.Load("")
	if err != nil {
		fmt.Printf("警告: 无法加载配置，将使用固定token: %v\n", err)
		fmt.Println("提示: 如果测试失败，请确保配置文件存在")
	}

	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("WebSocket 连接数压力测试")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Printf("目标连接数: %d\n", *targetConnections)
	fmt.Printf("并发连接数: %d\n", *concurrent)
	fmt.Printf("服务器地址: %s\n", *baseURL)
	fmt.Printf("使用相同token: %v\n", *useSameToken)
	fmt.Printf("测试持续时间: %v\n", *duration)
	fmt.Println()

	// 生成一个基础token（如果使用相同token）
	var baseToken string
	if *useSameToken {
		token, err := utils.GenerateToken(1, 1643534296849182959, "13800138001")
		if err != nil {
			fmt.Printf("错误: 无法生成token: %v\n", err)
			return
		}
		baseToken = token
		fmt.Println("使用固定token（所有连接使用相同userID，会触发单点登录）")
	} else {
		fmt.Println("为每个连接生成不同的token（不同userID）")
	}
	fmt.Println()

	var (
		successCount int64
		failCount    int64
		activeCount  int64
		wg           sync.WaitGroup
		stopChan     = make(chan struct{})
	)

	// 启动统计 goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				active := atomic.LoadInt64(&activeCount)
				success := atomic.LoadInt64(&successCount)
				fail := atomic.LoadInt64(&failCount)

				// 尝试从服务器获取真实连接数
				serverConnections := "N/A"
				// 将 ws://localhost:8081/ws 转换为 http://localhost:8081/stats
				statsURL := strings.Replace(*baseURL, "ws://", "http://", 1)
				statsURL = strings.Replace(statsURL, "/ws", "/stats", 1)
				if resp, err := http.Get(statsURL); err == nil {
					defer resp.Body.Close()
					if resp.StatusCode == http.StatusOK {
						var stats map[string]interface{}
						if json.NewDecoder(resp.Body).Decode(&stats) == nil {
							if conn, ok := stats["connections"].(float64); ok {
								serverConnections = fmt.Sprintf("%.0f", conn)
							}
						}
					}
				}

				fmt.Printf("[%s] 客户端统计: 活跃=%d, 成功=%d, 失败=%d | 服务器连接数: %s\n",
					time.Now().Format("15:04:05"), active, success, fail, serverConnections)
			case <-stopChan:
				return
			}
		}
	}()

	// 启动定时器
	time.AfterFunc(*duration, func() {
		close(stopChan)
		fmt.Println("\n测试时间到，开始关闭连接...")
	})

	start := time.Now()

	// 并发建立连接
	for i := 0; i < *targetConnections; i += *concurrent {
		select {
		case <-stopChan:
			break
		default:
		}

		for j := 0; j < *concurrent && i+j < *targetConnections; j++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				// 生成token
				var token string
				var err error
				if *useSameToken {
					token = baseToken
				} else {
					// 为每个连接生成不同的userID和token
					userID := uint(id + 10000) // 从10000开始，避免与真实用户ID冲突
					uid := int64(1643534296849182959 + id)
					phone := fmt.Sprintf("138%08d", id)
					token, err = utils.GenerateToken(userID, uid, phone)
					if err != nil {
						atomic.AddInt64(&failCount, 1)
						return
					}
				}

				// 构建完整的WebSocket URL
				wsURL := fmt.Sprintf("%s?token=%s", *baseURL, token)

				// 设置连接超时
				dialer := websocket.DefaultDialer
				dialer.HandshakeTimeout = 10 * time.Second

				conn, _, err := dialer.Dial(wsURL, nil)
				if err != nil {
					atomic.AddInt64(&failCount, 1)
					// 记录失败原因（每100个失败记录一次，避免日志过多）
					if atomic.LoadInt64(&failCount)%100 == 1 {
						fmt.Printf("连接失败示例: %v\n", err)
					}
					return
				}
				defer conn.Close()

				atomic.AddInt64(&successCount, 1)
				atomic.AddInt64(&activeCount, 1)
				defer atomic.AddInt64(&activeCount, -1)

				// 保持连接，定期发送 ping
				ticker := time.NewTicker(30 * time.Second)
				defer ticker.Stop()

				// 使用一个goroutine来读取消息
				readDone := make(chan struct{})
				go func() {
					defer close(readDone)
					for {
						conn.SetReadDeadline(time.Now().Add(60 * time.Second))
						_, _, err := conn.ReadMessage()
						if err != nil {
							// 任何错误都表示连接断开
							return
						}
					}
				}()

				// 主循环：发送ping和检查连接状态
				for {
					select {
					case <-ticker.C:
						// 发送 ping
						conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
						if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"ping"}`)); err != nil {
							return
						}
					case <-readDone:
						// 读取goroutine退出，说明连接断开
						return
					case <-stopChan:
						return
					}
				}
			}(i + j)
		}

		// 控制连接建立速度（避免瞬间大量连接导致服务器处理不过来）
		// 200并发时，每批之间等待200ms，给服务器处理时间
		time.Sleep(200 * time.Millisecond)
	}

	// 等待测试结束
	<-stopChan
	time.Sleep(2 * time.Second) // 等待统计更新

	elapsed := time.Since(start)
	success := atomic.LoadInt64(&successCount)
	fail := atomic.LoadInt64(&failCount)
	active := atomic.LoadInt64(&activeCount)

	// 获取服务器端连接数
	serverConnections := "N/A"
	serverRooms := "N/A"
	// 将 ws://localhost:8081/ws 转换为 http://localhost:8081/stats
	statsURL := strings.Replace(*baseURL, "ws://", "http://", 1)
	statsURL = strings.Replace(statsURL, "/ws", "/stats", 1)
	if resp, err := http.Get(statsURL); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var stats map[string]interface{}
			if json.NewDecoder(resp.Body).Decode(&stats) == nil {
				if conn, ok := stats["connections"].(float64); ok {
					serverConnections = fmt.Sprintf("%.0f", conn)
				}
				if rooms, ok := stats["rooms"].(float64); ok {
					serverRooms = fmt.Sprintf("%.0f", rooms)
				}
			}
		}
	} else {
		fmt.Printf("警告: 无法连接到服务器统计接口: %v\n", err)
		fmt.Printf("尝试访问的URL: %s\n", statsURL)
	}

	fmt.Println("\n═══════════════════════════════════════════")
	fmt.Println("测试结果")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Printf("总连接数: %d\n", *targetConnections)
	fmt.Printf("成功连接: %d\n", success)
	fmt.Printf("失败连接: %d\n", fail)
	fmt.Printf("客户端统计 - 当前活跃: %d\n", active)
	fmt.Printf("服务器统计 - 当前连接数: %s\n", serverConnections)
	fmt.Printf("服务器统计 - 房间数: %s\n", serverRooms)
	fmt.Printf("成功率: %.2f%%\n", float64(success)/float64(*targetConnections)*100)
	fmt.Printf("总耗时: %v\n", elapsed)
	if elapsed.Seconds() > 0 {
		fmt.Printf("连接速度: %.2f 连接/秒\n", float64(success)/elapsed.Seconds())
	}

	// 对比客户端和服务器端的连接数
	if serverConnections != "N/A" {
		var serverConn int
		if _, err := fmt.Sscanf(serverConnections, "%d", &serverConn); err == nil {
			if int64(serverConn) != active {
				fmt.Printf("\n⚠️  警告: 客户端活跃连接数 (%d) 与服务器连接数 (%d) 不一致！\n", active, serverConn)
				fmt.Printf("   可能原因：\n")
				fmt.Printf("   1. 连接已断开但客户端未检测到\n")
				fmt.Printf("   2. 服务器端连接被清理\n")
				fmt.Printf("   3. 网络延迟导致状态不同步\n")
			} else {
				fmt.Printf("\n✅ 客户端和服务器端连接数一致\n")
			}
		}
	}
	fmt.Println()

	// 等待所有连接关闭
	fmt.Println("等待所有连接关闭...")
	wg.Wait()

	// 等待 TIME_WAIT 状态释放（TCP 连接关闭后需要等待 60-120 秒）
	// 建议在两次测试之间等待至少 2 分钟
	fmt.Println("\n⚠️  提示: TCP 连接关闭后需要等待 TIME_WAIT 状态释放（约 60-120 秒）")
	fmt.Println("   如果立即进行下一次测试，可能会遇到端口耗尽问题")
	fmt.Println("   建议等待 2-3 分钟后再进行下一次测试")

	fmt.Println("测试完成")
}
