package services

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"go.uber.org/zap"

	"github.com/kaifa/game-platform/internal/logger"
)

// GasManager Gas费用管理器
type GasManager struct {
	ethClient  *ethclient.Client
	tronClient *client.GrpcClient
	hdWallet   *HDWallet
}

// NewGasManager 创建Gas管理器
func NewGasManager(ethClient *ethclient.Client, tronClient *client.GrpcClient, hdWallet *HDWallet) *GasManager {
	return &GasManager{
		ethClient:  ethClient,
		tronClient: tronClient,
		hdWallet:   hdWallet,
	}
}

// GetETHBalance 获取ETH余额（以太坊主币）
func (gm *GasManager) GetETHBalance(address common.Address) (*big.Float, error) {
	if gm.ethClient == nil {
		return nil, errors.New("以太坊客户端未初始化")
	}

	balance, err := gm.ethClient.BalanceAt(context.Background(), address, nil)
	if err != nil {
		return nil, fmt.Errorf("获取ETH余额失败: %w", err)
	}

	// 转换为ETH（18位小数）
	ethBalance := new(big.Float).Quo(new(big.Float).SetInt(balance), big.NewFloat(1e18))
	return ethBalance, nil
}

// GetTRXBalance 获取TRX余额（波场主币）
func (gm *GasManager) GetTRXBalance(address string) (*big.Float, error) {
	if gm.tronClient == nil {
		return nil, errors.New("波场客户端未初始化")
	}

	account, err := gm.tronClient.GetAccount(address)
	if err != nil {
		return nil, fmt.Errorf("获取账户信息失败: %w", err)
	}

	// TRX余额（以SUN为单位，1 TRX = 1,000,000 SUN）
	balance := big.NewInt(account.GetBalance())

	// 转换为TRX（6位小数）
	trxBalance := new(big.Float).Quo(new(big.Float).SetInt(balance), big.NewFloat(1e6))
	return trxBalance, nil
}

// EstimateGasFee 估算Gas费用
func (gm *GasManager) EstimateGasFee(chainType string, gasLimit uint64) (*big.Float, error) {
	if chainType == "erc20" {
		if gm.ethClient == nil {
			return nil, errors.New("以太坊客户端未初始化")
		}

		gasPrice, err := gm.ethClient.SuggestGasPrice(context.Background())
		if err != nil {
			return nil, fmt.Errorf("获取Gas价格失败: %w", err)
		}

		// 计算总费用（gasLimit * gasPrice）
		totalGas := new(big.Int).Mul(big.NewInt(int64(gasLimit)), gasPrice)

		// 转换为ETH（18位小数）
		gasFee := new(big.Float).Quo(new(big.Float).SetInt(totalGas), big.NewFloat(1e18))
		return gasFee, nil
	} else if chainType == "trc20" {
		// TRC20转账费用估算（以TRX为单位）
		// 典型费用：5-10 TRX
		estimatedFee := big.NewFloat(0.01) // 默认10 TRX（预留更多）
		return estimatedFee, nil
	}

	return nil, fmt.Errorf("不支持的链类型: %s", chainType)
}

// TransferETH 转账ETH（从主钱包转入Gas费用）
func (gm *GasManager) TransferETH(toAddr common.Address, amount *big.Float, privateKey *ecdsa.PrivateKey) (string, error) {
	if gm.ethClient == nil {
		return "", errors.New("以太坊客户端未初始化")
	}

	// 从私钥获取发送地址
	fromAddr := crypto.PubkeyToAddress(*privateKey.Public().(*ecdsa.PublicKey))

	// 获取nonce
	nonce, err := gm.ethClient.PendingNonceAt(context.Background(), fromAddr)
	if err != nil {
		return "", fmt.Errorf("获取nonce失败: %w", err)
	}

	// 获取Gas价格
	gasPrice, err := gm.ethClient.SuggestGasPrice(context.Background())
	if err != nil {
		return "", fmt.Errorf("获取Gas价格失败: %w", err)
	}

	// 估算Gas限制（ETH转账通常需要21000 gas）
	gasLimit := uint64(21000)

	// 转换金额（ETH转Wei）
	amountWei := new(big.Int)
	amount.Mul(amount, big.NewFloat(1e18)).Int(amountWei)

	// 构建交易
	tx := types.NewTransaction(nonce, toAddr, amountWei, gasLimit, gasPrice, nil)

	// 签名交易
	chainID := big.NewInt(1) // 主网
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", fmt.Errorf("签名交易失败: %w", err)
	}

	// 发送交易
	err = gm.ethClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", fmt.Errorf("发送交易失败: %w", err)
	}

	txHash := signedTx.Hash().Hex()

	logger.Logger.Info("ETH转账成功",
		zap.String("from", fromAddr.Hex()),
		zap.String("to", toAddr.Hex()),
		zap.String("amount", amount.String()),
		zap.String("tx_hash", txHash),
	)

	return txHash, nil
}

// TransferTRX 转账TRX（从主钱包转入Gas费用）
func (gm *GasManager) TransferTRX(toAddr string, amount *big.Float, privateKey *ecdsa.PrivateKey) (string, error) {
	if gm.tronClient == nil {
		return "", errors.New("波场客户端未初始化")
	}

	// 转换金额（TRX转SUN）
	amountSun := new(big.Int)
	amount.Mul(amount, big.NewFloat(1e6)).Int(amountSun)

	// 从主钱包派生地址和私钥（用于发送TRX）
	masterAddr, masterPrivateKey, err := gm.hdWallet.DeriveMasterTronAddress()
	if err != nil {
		return "", fmt.Errorf("派生主钱包地址失败: %w", err)
	}

	fromAddr := masterAddr

	// 创建TRX转账交易
	txExt, err := gm.tronClient.Transfer(fromAddr, toAddr, amountSun.Int64())
	if err != nil {
		return "", fmt.Errorf("创建TRX转账交易失败: %w", err)
	}

	if txExt == nil || txExt.Transaction == nil {
		return "", errors.New("交易创建失败，返回的交易为空")
	}

	tx := txExt

	// 签名交易（使用主钱包私钥）
	signedTx, err := transaction.SignTransactionECDSA(tx.Transaction, masterPrivateKey)
	if err != nil {
		return "", fmt.Errorf("签名交易失败: %w", err)
	}

	// 广播交易
	result, err := gm.tronClient.Broadcast(signedTx)
	if err != nil {
		return "", fmt.Errorf("广播交易失败: %w", err)
	}

	if result == nil || result.Code != api.Return_SUCCESS {
		msg := "未知错误"
		if result != nil && len(result.Message) > 0 {
			msg = string(result.Message)
		}
		return "", fmt.Errorf("交易失败: Code=%d, Message=%s", result.Code, msg)
	}

	// 提取交易哈希
	txHash := hex.EncodeToString(tx.GetTxid())

	logger.Logger.Info("TRX转账成功",
		zap.String("from", fromAddr),
		zap.String("to", toAddr),
		zap.String("amount", amount.String()),
		zap.String("tx_hash", txHash),
	)

	return txHash, nil
}

// EnsureGasBalance 确保地址有足够的Gas费用
// 如果Gas不足，从主钱包转入
func (gm *GasManager) EnsureGasBalance(address string, chainType string, requiredGas *big.Float, privateKey *ecdsa.PrivateKey) (bool, error) {
	var currentBalance *big.Float
	var err error

	if chainType == "erc20" {
		addr := common.HexToAddress(address)
		currentBalance, err = gm.GetETHBalance(addr)
		if err != nil {
			return false, fmt.Errorf("获取ETH余额失败: %w", err)
		}
	} else if chainType == "trc20" {
		currentBalance, err = gm.GetTRXBalance(address)
		if err != nil {
			return false, fmt.Errorf("获取TRX余额失败: %w", err)
		}
	} else {
		return false, fmt.Errorf("不支持的链类型: %s", chainType)
	}

	// 比较余额
	if currentBalance.Cmp(requiredGas) >= 0 {
		// 余额充足
		return true, nil
	}

	// 余额不足，从主钱包转入
	gasAmount := new(big.Float).Sub(requiredGas, currentBalance)
	// 额外加10%作为缓冲
	gasAmount.Mul(gasAmount, big.NewFloat(1.1))

	logger.Logger.Info("Gas余额不足，从主钱包转入",
		zap.String("address", address),
		zap.String("chain_type", chainType),
		zap.String("current_balance", currentBalance.String()),
		zap.String("required_gas", requiredGas.String()),
		zap.String("transfer_amount", gasAmount.String()),
	)

	var txHash string
	if chainType == "erc20" {
		toAddr := common.HexToAddress(address)
		txHash, err = gm.TransferETH(toAddr, gasAmount, privateKey)
		if err != nil {
			return false, fmt.Errorf("转入ETH失败: %w", err)
		}
	} else if chainType == "trc20" {
		txHash, err = gm.TransferTRX(address, gasAmount, privateKey)
		if err != nil {
			return false, fmt.Errorf("转入TRX失败: %w", err)
		}
	}

	logger.Logger.Info("Gas费用已转入，等待确认",
		zap.String("address", address),
		zap.String("chain_type", chainType),
		zap.String("amount", gasAmount.String()),
		zap.String("tx_hash", txHash),
	)

	// 返回false表示需要等待确认
	return false, nil
}
