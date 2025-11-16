package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/kaifa/game-platform/pkg/utils"
)

func main() {
	// 初始化雪花算法（机器ID=0）
	if err := utils.InitSnowflake(0); err != nil {
		panic(err)
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("🌟 雪花算法 UID 生成测试")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	// 测试1：顺序生成10个ID
	fmt.Println("【测试1】顺序生成10个UID：")
	for i := 0; i < 10; i++ {
		uid, _ := utils.GenerateUID()
		info := utils.ParseSnowflakeID(uid)
		fmt.Printf("%2d. UID: %19d  时间: %s  序列号: %4d\n",
			i+1, uid, info["time"], info["sequence"])
	}

	// 测试2：并发生成（测试线程安全）
	fmt.Println("\n【测试2】并发生成1000个UID（10个goroutine）：")

	var wg sync.WaitGroup
	uidChan := make(chan int64, 1000)

	startTime := time.Now()

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				uid, _ := utils.GenerateUID()
				uidChan <- uid
			}
		}()
	}

	wg.Wait()
	close(uidChan)

	duration := time.Since(startTime)

	// 检查唯一性
	uidMap := make(map[int64]bool)
	duplicates := 0
	for uid := range uidChan {
		if uidMap[uid] {
			duplicates++
		}
		uidMap[uid] = true
	}

	fmt.Printf("生成数量: %d\n", len(uidMap))
	fmt.Printf("耗时: %v\n", duration)
	fmt.Printf("平均: %v/个\n", duration/time.Duration(len(uidMap)))
	fmt.Printf("重复数量: %d\n", duplicates)

	if duplicates == 0 {
		fmt.Println("✅ 唯一性测试通过！")
	} else {
		fmt.Println("❌ 发现重复ID！")
	}

	// 测试3：性能测试
	fmt.Println("\n【测试3】性能测试（生成10万个UID）：")

	startTime = time.Now()
	for i := 0; i < 100000; i++ {
		utils.GenerateUID()
	}
	duration = time.Since(startTime)

	qps := float64(100000) / duration.Seconds()

	fmt.Printf("总耗时: %v\n", duration)
	fmt.Printf("平均耗时: %v/个\n", duration/100000)
	fmt.Printf("QPS: %.0f/秒\n", qps)

	// 测试4：解析ID示例
	fmt.Println("\n【测试4】解析UID示例：")

	uid, _ := utils.GenerateUID()
	info := utils.ParseSnowflakeID(uid)

	fmt.Printf("UID: %d\n", uid)
	fmt.Printf("详细信息：\n")
	fmt.Printf("  - 生成时间: %s\n", info["time"])
	fmt.Printf("  - 时间戳: %d\n", info["timestamp"])
	fmt.Printf("  - 机器ID: %d\n", info["machine_id"])
	fmt.Printf("  - 序列号: %d\n", info["sequence"])

	// 测试5：多机器ID测试
	fmt.Println("\n【测试5】不同机器ID生成的UID：")

	for machineID := 0; machineID < 5; machineID++ {
		utils.InitSnowflake(int64(machineID))
		uid, _ := utils.GenerateUID()
		info := utils.ParseSnowflakeID(uid)
		fmt.Printf("机器%d: UID=%19d  序列号=%4d\n",
			machineID, uid, info["sequence"])
	}

	fmt.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ 所有测试完成！")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}
