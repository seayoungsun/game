package services

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"go.uber.org/zap"

	"github.com/kaifa/game-platform/internal/database"
	"github.com/kaifa/game-platform/internal/logger"
	"github.com/kaifa/game-platform/pkg/models"
)

// CollectionService USDT归集服务
type CollectionService struct {
	ethClient       *ethclient.Client
	tronClient      *client.GrpcClient
	transferService *USDTTransferService
	gasManager      *GasManager
	hdWallet        *HDWallet
}

// NewCollectionService 创建USDT归集服务
func NewCollectionService(ethClient *ethclient.Client, tronClient *client.GrpcClient, transferService *USDTTransferService, gasManager *GasManager, hdWallet *HDWallet) *CollectionService {
	return &CollectionService{
		ethClient:       ethClient,
		tronClient:      tronClient,
		transferService: transferService,
		gasManager:      gasManager,
		hdWallet:        hdWallet,
	}
}

// GetERC20USDTBalance 获取ERC20 USDT余额
func (cs *CollectionService) GetERC20USDTBalance(address common.Address) (*big.Float, error) {
	if cs.ethClient == nil {
		return nil, errors.New("以太坊客户端未初始化")
	}

	// ERC20 balanceOf(address) 函数签名
	balanceOfSig := []byte("balanceOf(address)")
	hash := crypto.Keccak256Hash(balanceOfSig)
	methodID := hash[:4]

	// 编码参数：地址（32字节，左填充）
	paddedAddress := common.LeftPadBytes(address.Bytes(), 32)

	var data []byte
	data = append(data, methodID...)
	data = append(data, paddedAddress...)

	// 调用合约
	contractAddr := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7") // ERC20 USDT合约
	msg := ethereum.CallMsg{
		To:   &contractAddr,
		Data: data,
	}
	result, err := cs.ethClient.CallContract(context.Background(), msg, nil)
	if err != nil {
		return nil, fmt.Errorf("调用合约失败: %w", err)
	}

	// 解析余额（uint256）
	balance := new(big.Int).SetBytes(result)

	// 转换为USDT（6位小数）
	usdtBalance := new(big.Float).Quo(new(big.Float).SetInt(balance), big.NewFloat(1e6))
	return usdtBalance, nil
}

// GetTRC20USDTBalance 获取TRC20 USDT余额
func (cs *CollectionService) GetTRC20USDTBalance(address string) (*big.Float, error) {
	if cs.tronClient == nil {
		return nil, errors.New("波场客户端未初始化")
	}

	// 调用TRC20合约的balanceOf方法
	balance, err := cs.tronClient.TRC20ContractBalance(address, "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t") // TRC20 USDT合约
	if err != nil {
		return nil, fmt.Errorf("获取TRC20余额失败: %w", err)
	}

	// 转换为USDT（6位小数）
	usdtBalance := new(big.Float).Quo(new(big.Float).SetInt(balance), big.NewFloat(1e6))
	return usdtBalance, nil
}

// CollectUSDT 归集USDT（从派生地址归集到主钱包）
func (cs *CollectionService) CollectUSDT(userID uint, chainType string) (string, error) {
	// 1. 获取用户充值地址
	var depositAddr models.UserDepositAddress
	if err := database.DB.Where("user_id = ? AND chain_type = ?", userID, chainType).First(&depositAddr).Error; err != nil {
		return "", fmt.Errorf("未找到充值地址: %w", err)
	}

	// 2. 检查USDT余额
	var usdtBalance *big.Float
	var err error

	if chainType == "erc20" {
		addr := common.HexToAddress(depositAddr.Address)
		usdtBalance, err = cs.GetERC20USDTBalance(addr)
		if err != nil {
			return "", fmt.Errorf("获取ERC20余额失败: %w", err)
		}
	} else if chainType == "trc20" {
		usdtBalance, err = cs.GetTRC20USDTBalance(depositAddr.Address)
		if err != nil {
			return "", fmt.Errorf("获取TRC20余额失败: %w", err)
		}
	} else {
		return "", fmt.Errorf("不支持的链类型: %s", chainType)
	}

	// 检查是否有余额
	minBalance := big.NewFloat(0.000001) // 最小归集金额（0.000001 USDT）
	if usdtBalance.Cmp(minBalance) < 0 {
		return "", errors.New("余额不足，无需归集")
	}

	// 3. 估算Gas费用
	gasLimit := uint64(100000) // ERC20转账通常需要约100000 gas
	requiredGas, err := cs.gasManager.EstimateGasFee(chainType, gasLimit)
	if err != nil {
		return "", fmt.Errorf("估算Gas费用失败: %w", err)
	}

	// 4. 确保Gas余额充足
	masterPrivateKey := cs.getMasterPrivateKey(chainType)
	hasEnoughGas, err := cs.gasManager.EnsureGasBalance(depositAddr.Address, chainType, requiredGas, masterPrivateKey)
	if err != nil {
		return "", fmt.Errorf("确保Gas余额失败: %w", err)
	}

	if !hasEnoughGas {
		// Gas费用已转入，需要等待确认
		logger.Logger.Info("Gas费用已转入，等待确认后再归集",
			zap.Uint("user_id", userID),
			zap.String("chain_type", chainType),
			zap.String("address", depositAddr.Address),
		)
		return "", errors.New("Gas费用已转入，请稍后重试（等待确认）")
	}

	// 5. 派生地址的私钥（用于签名转账）
	var fromAddr common.Address
	var fromAddrTron string
	var privateKey *ecdsa.PrivateKey

	if chainType == "erc20" {
		fromAddr, privateKey, err = cs.hdWallet.DeriveEthereumAddress(0, uint32(userID))
		if err != nil {
			return "", fmt.Errorf("派生以太坊地址失败: %w", err)
		}
	} else if chainType == "trc20" {
		fromAddrTron, privateKey, err = cs.hdWallet.DeriveTronAddress(0, uint32(userID))
		if err != nil {
			return "", fmt.Errorf("派生波场地址失败: %w", err)
		}
	}

	// 6. 获取主钱包地址
	masterAddr, _, err := cs.getMasterAddress(chainType)
	if err != nil {
		return "", fmt.Errorf("获取主钱包地址失败: %w", err)
	}

	// 7. 转换金额（USDT转最小单位）
	amountInt := new(big.Int)
	usdtBalance.Mul(usdtBalance, big.NewFloat(1e6)).Int(amountInt)

	// 8. 执行USDT转账
	var txHash string
	if chainType == "erc20" {
		toAddr := common.HexToAddress(masterAddr)
		txHash, err = cs.transferService.TransferERC20USDT(fromAddr, toAddr, amountInt, privateKey)
		if err != nil {
			return "", fmt.Errorf("ERC20转账失败: %w", err)
		}
	} else if chainType == "trc20" {
		txHash, err = cs.transferService.TransferTRC20USDT(fromAddrTron, masterAddr, amountInt, privateKey)
		if err != nil {
			return "", fmt.Errorf("TRC20转账失败: %w", err)
		}
	}

	logger.Logger.Info("USDT归集成功",
		zap.Uint("user_id", userID),
		zap.String("chain_type", chainType),
		zap.String("from_address", depositAddr.Address),
		zap.String("to_address", masterAddr),
		zap.String("amount", usdtBalance.String()),
		zap.String("tx_hash", txHash),
	)

	return txHash, nil
}

// getMasterAddress 获取主钱包地址
func (cs *CollectionService) getMasterAddress(chainType string) (string, *ecdsa.PrivateKey, error) {
	if chainType == "erc20" {
		addr, privateKey, err := cs.hdWallet.DeriveMasterEthereumAddress()
		if err != nil {
			return "", nil, err
		}
		return addr.Hex(), privateKey, nil
	} else if chainType == "trc20" {
		addr, privateKey, err := cs.hdWallet.DeriveMasterTronAddress()
		if err != nil {
			return "", nil, err
		}
		return addr, privateKey, nil
	}
	return "", nil, fmt.Errorf("不支持的链类型: %s", chainType)
}

// getMasterPrivateKey 获取主钱包私钥
func (cs *CollectionService) getMasterPrivateKey(chainType string) *ecdsa.PrivateKey {
	_, privateKey, err := cs.getMasterAddress(chainType)
	if err != nil {
		logger.Logger.Error("获取主钱包私钥失败", zap.Error(err))
		return nil
	}
	return privateKey
}

// BatchCollectUSDT 批量归集USDT
func (cs *CollectionService) BatchCollectUSDT(chainType string, limit int) error {
	// 查询有余额的充值地址
	var depositAddrs []models.UserDepositAddress
	if err := database.DB.Where("chain_type = ?", chainType).Limit(limit).Find(&depositAddrs).Error; err != nil {
		return fmt.Errorf("查询充值地址失败: %w", err)
	}

	for _, depositAddr := range depositAddrs {
		// 检查余额（快速检查，避免无余额地址）
		var balance *big.Float
		var err error

		if chainType == "erc20" {
			addr := common.HexToAddress(depositAddr.Address)
			balance, err = cs.GetERC20USDTBalance(addr)
			if err != nil {
				logger.Logger.Warn("获取ERC20余额失败",
					zap.Uint("user_id", depositAddr.UserID),
					zap.String("address", depositAddr.Address),
					zap.Error(err),
				)
				continue
			}
		} else if chainType == "trc20" {
			balance, err = cs.GetTRC20USDTBalance(depositAddr.Address)
			if err != nil {
				logger.Logger.Warn("获取TRC20余额失败",
					zap.Uint("user_id", depositAddr.UserID),
					zap.String("address", depositAddr.Address),
					zap.Error(err),
				)
				continue
			}
		}

		// 检查是否有余额（最小归集金额）
		minBalance := big.NewFloat(0.000001)
		if balance.Cmp(minBalance) < 0 {
			continue
		}

		// 执行归集
		_, err = cs.CollectUSDT(depositAddr.UserID, chainType)
		if err != nil {
			logger.Logger.Warn("归集失败",
				zap.Uint("user_id", depositAddr.UserID),
				zap.String("chain_type", chainType),
				zap.Error(err),
			)
			continue
		}

		// 避免请求过快
		time.Sleep(2 * time.Second)
	}

	return nil
}
