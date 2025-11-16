package services

import (
	"crypto/ecdsa"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
	"go.uber.org/zap"

	"github.com/kaifa/game-platform/internal/logger"
)

// HDWallet HD钱包（分层确定性钱包）
type HDWallet struct {
	masterKey *bip32.Key
	mnemonic  string
}

// NewHDWallet 创建新的HD钱包（从助记词生成）
func NewHDWallet(mnemonic string) (*HDWallet, error) {
	if mnemonic == "" {
		return nil, errors.New("助记词不能为空")
	}

	// 验证助记词
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("无效的助记词")
	}

	// 从助记词生成种子
	seed := bip39.NewSeed(mnemonic, "")

	// 生成主密钥
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return nil, fmt.Errorf("生成主密钥失败: %w", err)
	}

	logger.Logger.Info("HD钱包创建成功",
		zap.Bool("has_master_key", masterKey != nil),
	)

	return &HDWallet{
		masterKey: masterKey,
		mnemonic:  mnemonic,
	}, nil
}

// GenerateMnemonic 生成新的助记词
// entropyBits: 128(12个单词), 256(24个单词)
func GenerateMnemonic(entropyBits int) (string, error) {
	if entropyBits != 128 && entropyBits != 256 {
		return "", errors.New("entropy bits must be 128 or 256")
	}

	entropy, err := bip39.NewEntropy(entropyBits)
	if err != nil {
		return "", fmt.Errorf("生成熵失败: %w", err)
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("生成助记词失败: %w", err)
	}

	return mnemonic, nil
}

// DeriveEthereumAddress 派生以太坊地址
// path格式: m/44'/60'/account'/0/address_index
// account: 账户索引（通常为0）
// addressIndex: 地址索引（可以使用用户ID）
func (w *HDWallet) DeriveEthereumAddress(account, addressIndex uint32) (common.Address, *ecdsa.PrivateKey, error) {
	// BIP44路径: m/44'/60'/account'/0/address_index
	// purpose = 44' (强化派生)
	// coin_type = 60' (以太坊)
	// account = account'
	// change = 0 (外部地址)
	// address_index = addressIndex

	path := []uint32{
		44 + bip32.FirstHardenedChild,      // purpose' = 44'
		60 + bip32.FirstHardenedChild,      // coin_type' = 60' (以太坊)
		account + bip32.FirstHardenedChild, // account'
		0,                                  // change (外部地址)
		addressIndex,                       // address_index
	}

	// 派生密钥
	key := w.masterKey
	for _, index := range path {
		childKey, err := key.NewChildKey(index)
		if err != nil {
			return common.Address{}, nil, fmt.Errorf("派生子密钥失败: %w", err)
		}
		key = childKey
	}

	// 从派生的密钥获取私钥
	privateKey, err := crypto.ToECDSA(key.Key)
	if err != nil {
		return common.Address{}, nil, fmt.Errorf("转换为ECDSA私钥失败: %w", err)
	}

	// 从私钥获取公钥
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return common.Address{}, nil, errors.New("无法获取ECDSA公钥")
	}

	// 生成以太坊地址
	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	return address, privateKey, nil
}

// DeriveEthereumAddressByUserID 根据用户ID派生以太坊地址
func (w *HDWallet) DeriveEthereumAddressByUserID(userID uint) (common.Address, error) {
	// 使用用户ID作为address_index，账户使用0
	address, _, err := w.DeriveEthereumAddress(0, uint32(userID))
	if err != nil {
		return common.Address{}, err
	}
	return address, nil
}

// DeriveMasterEthereumAddress 派生主钱包以太坊地址（account=0, index=0）
func (w *HDWallet) DeriveMasterEthereumAddress() (common.Address, *ecdsa.PrivateKey, error) {
	return w.DeriveEthereumAddress(0, 0)
}

// DeriveTronAddress 派生波场地址
// path格式: m/44'/195'/account'/0/address_index
// account: 账户索引（通常为0）
// addressIndex: 地址索引（可以使用用户ID）
func (w *HDWallet) DeriveTronAddress(account, addressIndex uint32) (string, *ecdsa.PrivateKey, error) {
	// BIP44路径: m/44'/195'/account'/0/address_index
	// purpose = 44' (强化派生)
	// coin_type = 195' (波场)
	// account = account'
	// change = 0 (外部地址)
	// address_index = addressIndex

	path := []uint32{
		44 + bip32.FirstHardenedChild,      // purpose' = 44'
		195 + bip32.FirstHardenedChild,     // coin_type' = 195' (波场)
		account + bip32.FirstHardenedChild, // account'
		0,                                  // change (外部地址)
		addressIndex,                       // address_index
	}

	// 派生密钥
	key := w.masterKey
	for _, index := range path {
		childKey, err := key.NewChildKey(index)
		if err != nil {
			return "", nil, fmt.Errorf("派生子密钥失败: %w", err)
		}
		key = childKey
	}

	// 从派生的密钥获取私钥
	privateKey, err := crypto.ToECDSA(key.Key)
	if err != nil {
		return "", nil, fmt.Errorf("转换为ECDSA私钥失败: %w", err)
	}

	// 从私钥获取公钥
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", nil, errors.New("无法获取ECDSA公钥")
	}

	// 生成以太坊格式地址（波场使用相同的椭圆曲线）
	ethereumAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// 转换为波场地址格式（Base58编码，前缀0x41）
	tronAddress, err := ethereumToTronAddress(ethereumAddress)
	if err != nil {
		return "", nil, fmt.Errorf("转换为波场地址失败: %w", err)
	}

	return tronAddress, privateKey, nil
}

// DeriveTronAddressByUserID 根据用户ID派生波场地址
func (w *HDWallet) DeriveTronAddressByUserID(userID uint) (string, error) {
	// 使用用户ID作为address_index，账户使用0
	address, _, err := w.DeriveTronAddress(0, uint32(userID))
	if err != nil {
		return "", err
	}
	return address, nil
}

// DeriveMasterTronAddress 派生主钱包波场地址（account=0, index=0）
func (w *HDWallet) DeriveMasterTronAddress() (string, *ecdsa.PrivateKey, error) {
	return w.DeriveTronAddress(0, 0)
}

// ethereumToTronAddress 将以太坊地址转换为波场地址
// 波场使用与以太坊相同的椭圆曲线（secp256k1），地址格式不同
func ethereumToTronAddress(ethAddr common.Address) (string, error) {
	// 以太坊地址去掉0x前缀，获取20字节地址
	addrBytes := ethAddr.Bytes()

	// 波场地址前缀：0x41 (对应字符 'T')
	tronBytes := append([]byte{0x41}, addrBytes...)

	// 计算校验和（双SHA256）
	hash1 := crypto.Keccak256(tronBytes)
	hash2 := crypto.Keccak256(hash1)
	checksum := hash2[:4]

	// 组合地址和校验和
	fullBytes := append(tronBytes, checksum...)

	// Base58编码
	address := base58.Encode(fullBytes)

	return address, nil
}

// GetPath 获取BIP44路径字符串（用于调试和记录）
func GetPath(coinType uint32, account, addressIndex uint32) string {
	return fmt.Sprintf("m/44'/%d'/%d'/0/%d", coinType, account, addressIndex)
}

// GetEthereumPath 获取以太坊BIP44路径字符串
func GetEthereumPath(account, addressIndex uint32) string {
	return GetPath(60, account, addressIndex)
}

// GetTronPath 获取波场BIP44路径字符串
func GetTronPath(account, addressIndex uint32) string {
	return GetPath(195, account, addressIndex)
}
