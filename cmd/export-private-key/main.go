package main

import (
	"crypto/ecdsa"
	"flag"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/kaifa/game-platform/internal/config"
	"github.com/kaifa/game-platform/pkg/services"
)

func main() {
	var userID uint
	var chainType string
	var mnemonic string

	flag.UintVar(&userID, "user-id", 0, "用户ID")
	flag.StringVar(&chainType, "chain-type", "", "链类型: trc20 或 erc20")
	flag.StringVar(&mnemonic, "mnemonic", "", "助记词（如果不提供，从配置文件读取）")
	flag.Parse()

	if userID == 0 {
		fmt.Println("错误: 必须指定 --user-id")
		flag.Usage()
		os.Exit(1)
	}

	if chainType != "trc20" && chainType != "erc20" {
		fmt.Println("错误: chain-type 必须是 trc20 或 erc20")
		flag.Usage()
		os.Exit(1)
	}

	// 如果没有提供助记词，尝试从配置文件读取
	if mnemonic == "" {
		cfg, err := config.Load("")
		if err != nil {
			fmt.Printf("错误: 无法加载配置: %v\n", err)
			fmt.Println("提示: 请使用 --mnemonic 参数直接提供助记词")
			os.Exit(1)
		}

		if cfg.Payment.MasterMnemonic == "" {
			fmt.Println("错误: 配置文件中未找到助记词，请使用 --mnemonic 参数")
			os.Exit(1)
		}

		mnemonic = cfg.Payment.MasterMnemonic
		fmt.Println("✓ 从配置文件读取助记词")
	}

	// 创建HD钱包
	hdWallet, err := services.NewHDWallet(mnemonic)
	if err != nil {
		fmt.Printf("错误: 创建HD钱包失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=========================================")
	fmt.Printf("用户ID: %d\n", userID)
	fmt.Printf("链类型: %s\n", chainType)
	fmt.Println("=========================================")

	var address string
	var privateKey *ecdsa.PrivateKey
	var path string

	if chainType == "trc20" {
		path = services.GetTronPath(0, uint32(userID))
		address, privateKey, err = hdWallet.DeriveTronAddress(0, uint32(userID))
		if err != nil {
			fmt.Printf("错误: 派生波场地址失败: %v\n", err)
			os.Exit(1)
		}
	} else {
		path = services.GetEthereumPath(0, uint32(userID))
		ethAddr, pk, err2 := hdWallet.DeriveEthereumAddress(0, uint32(userID))
		if err2 != nil {
			fmt.Printf("错误: 派生以太坊地址失败: %v\n", err2)
			os.Exit(1)
		}
		address = ethAddr.Hex()
		privateKey = pk
	}

	// 导出私钥
	privateKeyBytes := crypto.FromECDSA(privateKey)
	privateKeyHex := fmt.Sprintf("%x", privateKeyBytes)

	fmt.Println("\n✅ 派生成功！")
	fmt.Println("=========================================")
	fmt.Printf("BIP44路径: %s\n", path)
	fmt.Printf("地址: %s\n", address)
	fmt.Printf("私钥 (64位十六进制): %s\n", privateKeyHex)
	fmt.Println("=========================================")
	fmt.Println("\n⚠️  安全提示：")
	fmt.Println("1. 私钥请妥善保管，不要泄露给他人")
	fmt.Println("2. 不要将私钥提交到代码仓库")
	fmt.Println("3. 建议仅在需要时导出私钥")
	fmt.Println("\n📝 导入到钱包：")
	fmt.Println("MetaMask: 账户 → 导入账户 → 私钥 → 粘贴私钥")
	fmt.Println("TP钱包: 导入钱包 → 私钥导入 → 粘贴私钥")
	fmt.Println("=========================================")
}
