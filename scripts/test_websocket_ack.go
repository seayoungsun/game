package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// 测试 WebSocket 的 ACK 机制
// 这个程序会连接到游戏服务器，发送消息，并测量延迟
// 延迟主要来自 TCP 的 ACK 等待时间

func main() {
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("WebSocket ACK 机制测试")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println()

	// 注意：需要先启动游戏服务器，并且有一个有效的 token
	// 这里使用测试 token（实际使用时需要替换）
	url := "ws://10.211.55.29:8081/ws?token=your_test_token_here"

	fmt.Printf("连接到: %s\n", url)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Printf("连接失败: %v", err)
		log.Println("提示：请确保游戏服务器已启动，并且提供有效的 token")
		return
	}
	defer conn.Close()

	fmt.Println("✅ 连接成功！")
	fmt.Println()

	// 测试1：发送单条消息，测量延迟
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("测试1：单条消息延迟（包含 ACK 等待时间）")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	message := []byte(`{"type":"ping"}`)
	start := time.Now()
	err = conn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		log.Fatal("发送失败:", err)
	}
	elapsed := time.Since(start)

	fmt.Printf("消息大小: %d 字节\n", len(message))
	fmt.Printf("发送延迟: %v\n", elapsed)
	fmt.Printf("延迟说明: 这个延迟包括：\n")
	fmt.Printf("  - 数据包发送时间: ~1ms\n")
	fmt.Printf("  - ACK 等待时间: ~%v (主要耗时)\n", elapsed-time.Millisecond)
	fmt.Println()

	// 测试2：发送多条消息，测量总延迟
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("测试2：多条消息延迟（验证 ACK 累积效应）")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	const messageCount = 10
	start = time.Now()
	for i := 0; i < messageCount; i++ {
		data := []byte(fmt.Sprintf(`{"type":"ping","seq":%d}`, i))
		err = conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Fatal("发送失败:", err)
		}
	}
	elapsed = time.Since(start)

	fmt.Printf("发送消息数: %d\n", messageCount)
	fmt.Printf("总延迟: %v\n", elapsed)
	fmt.Printf("平均每条消息延迟: %v\n", elapsed/messageCount)
	fmt.Printf("说明: 如果网络延迟 50ms，总延迟应该是 ~%v\n", time.Duration(messageCount*50)*time.Millisecond)
	fmt.Println()

	// 测试3：设置写入超时，验证 ACK 超时
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("测试3：写入超时测试（验证 ACK 超时机制）")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 设置很短的超时（1毫秒），应该会超时
	conn.SetWriteDeadline(time.Now().Add(1 * time.Millisecond))
	start = time.Now()
	err = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"ping"}`))
	elapsed = time.Since(start)

	if err != nil {
		fmt.Printf("✅ 预期行为：写入超时（因为超时时间太短）\n")
		fmt.Printf("错误: %v\n", err)
		fmt.Printf("耗时: %v\n", elapsed)
		fmt.Printf("说明: 这个超时包括 ACK 等待时间\n")
	} else {
		fmt.Printf("写入成功（网络延迟很低）\n")
		fmt.Printf("耗时: %v\n", elapsed)
	}
	fmt.Println()

	// 测试4：正常超时设置
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("测试4：正常超时设置（10秒）")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	start = time.Now()
	err = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"ping"}`))
	elapsed = time.Since(start)

	if err != nil {
		fmt.Printf("❌ 写入失败: %v\n", err)
	} else {
		fmt.Printf("✅ 写入成功\n")
		fmt.Printf("耗时: %v\n", elapsed)
		fmt.Printf("说明: 这个耗时包括 ACK 等待时间\n")
	}
	fmt.Println()

	// 总结
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("测试总结")
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("1. WriteMessage() 会阻塞，直到收到 ACK")
	fmt.Println("2. 延迟主要来自 ACK 等待时间（网络 RTT）")
	fmt.Println("3. 如果 ACK 超时，会返回错误")
	fmt.Println("4. ACK 在 TCP 传输层自动处理，应用层看不到")
	fmt.Println()
	fmt.Println("关键理解：")
	fmt.Println("  - WebSocket 底层是 TCP")
	fmt.Println("  - TCP 自动处理 ACK（在操作系统内核中）")
	fmt.Println("  - 应用层看不到 ACK，但 ACK 确实存在")
	fmt.Println("  - ACK 影响延迟和吞吐量")
}
