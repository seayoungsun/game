package services

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	tronCommon "github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"go.uber.org/zap"

	"github.com/kaifa/game-platform/internal/logger"
)

// USDTTransferService USDT转账服务
type USDTTransferService struct {
	ethClient    *ethclient.Client
	tronClient   *client.GrpcClient
	etherscanURL string
	tronAPIURL   string
	hdWallet     *HDWallet
}

// ERC20 USDT 合约地址（主网）
const (
	ERC20USDTContract = "0xdAC17F958D2ee523a2206206994597C13D831ec7" // 以太坊主网USDT合约
	TRC20USDTContract = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"         // 波场USDT合约
)

var (
	usdtTransferServiceInstance *USDTTransferService
	usdtTransferServiceOnce     sync.Once
)

// NewUSDTTransferService 创建USDT转账服务
func NewUSDTTransferService(hdWallet *HDWallet) *USDTTransferService {
	usdtTransferServiceOnce.Do(func() {
		ts := &USDTTransferService{
			hdWallet:     hdWallet,
			etherscanURL: "https://api.etherscan.io/api",
			tronAPIURL:   "https://api.trongrid.io",
		}

		// 初始化以太坊客户端（使用公共节点或Infura）
		// 注意：生产环境应该使用自己的节点或Infura/Alchemy等服务
		ethClient, err := ethclient.Dial("https://eth.llamarpc.com") // 公共RPC节点
		if err != nil {
			logger.Logger.Warn("连接以太坊节点失败，ERC20转账功能将不可用",
				zap.Error(err),
			)
		} else {
			ts.ethClient = ethClient
			logger.Logger.Info("以太坊客户端连接成功")
		}

		// 初始化波场客户端（使用TronGrid公共节点）
		// 注意：生产环境应该使用自己的节点或专用服务
		tronClient := client.NewGrpcClient("grpc.trongrid.io:50051")
		err = tronClient.Start(client.GRPCInsecure())
		if err != nil {
			logger.Logger.Warn("连接波场节点失败，TRC20转账功能将不可用",
				zap.Error(err),
			)
		} else {
			ts.tronClient = tronClient
			logger.Logger.Info("波场客户端连接成功")
		}

		usdtTransferServiceInstance = ts
	})

	return usdtTransferServiceInstance
}

// TransferERC20USDT 转账ERC20 USDT
func (ts *USDTTransferService) TransferERC20USDT(fromAddr common.Address, toAddr common.Address, amount *big.Int, privateKey *ecdsa.PrivateKey) (string, error) {
	if ts.ethClient == nil {
		return "", errors.New("以太坊客户端未初始化")
	}

	// ERC20 Transfer函数签名
	transferFnSignature := []byte("transfer(address,uint256)")
	hash := crypto.Keccak256Hash(transferFnSignature)
	methodID := hash[:4]

	// 编码参数：to地址和amount
	paddedAddress := common.LeftPadBytes(toAddr.Bytes(), 32)
	paddedAmount := common.LeftPadBytes(amount.Bytes(), 32)

	var data []byte
	data = append(data, methodID...)
	data = append(data, paddedAddress...)
	data = append(data, paddedAmount...)

	// 获取链ID（主网为1）
	chainID := big.NewInt(1)

	// 获取nonce
	nonce, err := ts.ethClient.PendingNonceAt(context.Background(), fromAddr)
	if err != nil {
		return "", fmt.Errorf("获取nonce失败: %w", err)
	}

	// 获取Gas价格
	gasPrice, err := ts.ethClient.SuggestGasPrice(context.Background())
	if err != nil {
		return "", fmt.Errorf("获取Gas价格失败: %w", err)
	}

	// Gas限制（ERC20转账通常需要约65000 gas）
	gasLimit := uint64(100000)

	// 构建交易
	tx := types.NewTransaction(nonce, common.HexToAddress(ERC20USDTContract), big.NewInt(0), gasLimit, gasPrice, data)

	// 签名交易
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", fmt.Errorf("签名交易失败: %w", err)
	}

	// 发送交易
	err = ts.ethClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", fmt.Errorf("发送交易失败: %w", err)
	}

	txHash := signedTx.Hash().Hex()

	logger.Logger.Info("ERC20 USDT转账成功",
		zap.String("from", fromAddr.Hex()),
		zap.String("to", toAddr.Hex()),
		zap.String("amount", amount.String()),
		zap.String("tx_hash", txHash),
	)

	return txHash, nil
}

// TransferTRC20USDT 转账TRC20 USDT
func (ts *USDTTransferService) TransferTRC20USDT(fromAddr string, toAddr string, amount *big.Int, privateKey *ecdsa.PrivateKey) (string, error) {
	if ts.tronClient == nil {
		return "", errors.New("波场客户端未初始化")
	}

	// 手续费上限（以SUN为单位，1 TRX = 1,000,000 SUN）
	// TRC20转账通常需要约10 TRX的费用上限
	feeLimit := int64(10000000) // 10 TRX

	// 创建TRC20转账交易
	txExt, err := ts.tronClient.TRC20Send(fromAddr, toAddr, TRC20USDTContract, amount, feeLimit)
	if err != nil {
		return "", fmt.Errorf("创建TRC20转账交易失败: %w", err)
	}

	if txExt == nil || txExt.Transaction == nil {
		return "", errors.New("交易创建失败，返回的交易为空")
	}

	// 使用transaction包的SignTransactionECDSA方法签名交易
	signedTx, err := transaction.SignTransactionECDSA(txExt.Transaction, privateKey)
	if err != nil {
		return "", fmt.Errorf("签名交易失败: %w", err)
	}

	// 广播交易（使用Broadcast方法）
	result, err := ts.tronClient.Broadcast(signedTx)
	if err != nil {
		return "", fmt.Errorf("广播交易失败: %w", err)
	}

	// 检查交易结果
	if result == nil || result.Code != api.Return_SUCCESS {
		msg := "未知错误"
		if result != nil && len(result.Message) > 0 {
			msg = string(result.Message)
		}
		return "", fmt.Errorf("交易失败: Code=%d, Message=%s", result.Code, msg)
	}

	// 提取交易哈希（从TransactionExtention获取）
	txHash := ""
	if txExt != nil && len(txExt.GetTxid()) > 0 {
		txHash = hex.EncodeToString(txExt.GetTxid())
	} else {
		// 如果没有Txid，计算交易的哈希（使用SHA256）
		rawData := signedTx.GetRawData()
		if rawData != nil && len(rawData.GetData()) > 0 {
			// 使用common包的Keccak256计算交易哈希
			hash := tronCommon.Keccak256(rawData.GetData())
			txHash = hex.EncodeToString(hash)
		} else {
			// 最后的备选方案：序列化交易并计算哈希
			// 注意：这只是一个备选方案，实际应该使用正确的交易哈希计算方法
			return "", errors.New("无法获取交易哈希")
		}
	}

	logger.Logger.Info("TRC20 USDT转账成功",
		zap.String("from", fromAddr),
		zap.String("to", toAddr),
		zap.String("amount", amount.String()),
		zap.String("tx_hash", txHash),
	)

	return txHash, nil
}
