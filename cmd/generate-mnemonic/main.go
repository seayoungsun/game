package main

import (
	"fmt"
	"os"

	"github.com/kaifa/game-platform/pkg/services"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run main.go <128|256>")
		fmt.Println("  128 = 12个单词（推荐用于测试）")
		fmt.Println("  256 = 24个单词（推荐用于生产）")
		os.Exit(1)
	}

	var entropyBits int
	if os.Args[1] == "128" {
		entropyBits = 128
	} else if os.Args[1] == "256" {
		entropyBits = 256
	} else {
		fmt.Println("错误: entropy bits必须是128或256")
		os.Exit(1)
	}

	mnemonic, err := services.GenerateMnemonic(entropyBits)
	if err != nil {
		fmt.Printf("生成助记词失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=========================================")
	fmt.Println("✅ 助记词生成成功")
	fmt.Println("=========================================")
	fmt.Println(mnemonic)
	fmt.Println("=========================================")
	fmt.Println("⚠️  请妥善保管此助记词，不要泄露给他人！")
	fmt.Println("⚠️  生产环境请使用环境变量或密钥管理服务存储！")
	fmt.Println("=========================================")
}
