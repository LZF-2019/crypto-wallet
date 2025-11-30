package service

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
	"go.uber.org/zap"

	"crypto-wallet-api/internal/blockchain"
	"crypto-wallet-api/internal/logger"
	"crypto-wallet-api/internal/models"
	"crypto-wallet-api/internal/repository"
	"crypto-wallet-api/internal/utils"
	"crypto-wallet-api/pkg/cache"
)

// WalletService 钱包服务
type WalletService struct {
	walletRepo       *repository.WalletRepository
	blockchainClient blockchain.BlockchainClient
	cache            *cache.RedisCache
	encryptionKey    []byte // 用于加密私钥的密钥
}

// NewWalletService 创建钱包服务实例
func NewWalletService(
	walletRepo *repository.WalletRepository,
	blockchainClient blockchain.BlockchainClient,
	cache *cache.RedisCache,
	encryptionKey []byte,
) *WalletService {
	return &WalletService{
		walletRepo:       walletRepo,
		blockchainClient: blockchainClient,
		cache:            cache,
		encryptionKey:    encryptionKey,
	}
}

// CreateWallet 创建新钱包
func (s *WalletService) CreateWallet(ctx context.Context, userID uint, req *models.WalletCreateRequest) (*models.Wallet, error) {
	// 1. 生成钱包地址和私钥
	address, privateKey, err := s.blockchainClient.CreateWallet()
	if err != nil {
		return nil, err
	}

	// 2. 导出私钥为十六进制字符串
	privateKeyBytes := crypto.FromECDSA(privateKey)
	privateKeyHex := hex.EncodeToString(privateKeyBytes)

	// 3. 加密私钥
	encryptedKey, err := utils.EncryptAES(privateKeyHex, s.encryptionKey)
	if err != nil {
		return nil, err
	}

	// 4. 创建钱包对象
	wallet := &models.Wallet{
		UserID:              userID,
		Address:             address,
		PrivateKeyEncrypted: encryptedKey,
		ChainID:             req.ChainID,
		Balance:             "0",
		Name:                req.Name,
	}

	// 5. 保存到数据库
	if err := s.walletRepo.Create(ctx, wallet); err != nil {
		return nil, err
	}

	// 6. 异步查询链上余额并更新
	go s.updateBalanceAsync(context.Background(), wallet.Address)

	return wallet, nil
}

// GetWalletByAddress 根据地址查询钱包
func (s *WalletService) GetWalletByAddress(ctx context.Context, userID uint, address string) (*models.Wallet, error) {
	// 1. 查询钱包
	wallet, err := s.walletRepo.GetByAddress(ctx, address)
	if err != nil {
		return nil, err
	}

	// 2. 验证所有权
	if wallet.UserID != userID {
		return nil, errors.New("wallet not found")
	}

	return wallet, nil
}

// GetUserWallets 获取用户的所有钱包
func (s *WalletService) GetUserWallets(ctx context.Context, userID uint) ([]*models.Wallet, error) {
	return s.walletRepo.GetByUserID(ctx, userID)
}

// GetBalance 查询钱包余额（实时从链上查询）
func (s *WalletService) GetBalance(ctx context.Context, userID uint, address string) (*big.Int, error) {
	// 1. 验证钱包所有权
	_, err := s.GetWalletByAddress(ctx, userID, address)
	if err != nil {
		return nil, err
	}

	// 2. 先查缓存
	cacheKey := "balance:" + address
	if cachedBalance, err := s.cache.Get(ctx, cacheKey); err == nil {
		balance := new(big.Int)
		balance.SetString(cachedBalance, 10)
		return balance, nil
	}

	// 3. 从链上查询
	balance, err := s.blockchainClient.GetBalance(ctx, address)
	if err != nil {
		return nil, err
	}

	// 4. 写入缓存（30秒过期）
	s.cache.Set(ctx, cacheKey, balance.String(), 30)

	// 5. 异步更新数据库
	go s.walletRepo.UpdateBalance(context.Background(), address, balance.String())

	return balance, nil
}

// UpdateWallet 更新钱包信息（仅支持更新名称）
func (s *WalletService) UpdateWallet(ctx context.Context, userID uint, address string, name string) error {
	// 1. 验证钱包所有权
	wallet, err := s.GetWalletByAddress(ctx, userID, address)
	if err != nil {
		return err
	}

	// 2. 更新名称
	wallet.Name = name

	// 3. 保存到数据库
	return s.walletRepo.Update(ctx, wallet)
}

// DeleteWallet 删除钱包
func (s *WalletService) DeleteWallet(ctx context.Context, userID uint, address string) error {
	// 1. 验证钱包所有权
	wallet, err := s.GetWalletByAddress(ctx, userID, address)
	if err != nil {
		return err
	}

	// 2. 检查余额是否为0（安全考虑）
	balance, err := s.GetBalance(ctx, userID, address)
	if err != nil {
		return err
	}

	if balance.Cmp(big.NewInt(0)) > 0 {
		return errors.New("cannot delete wallet with non-zero balance")
	}

	// 3. 删除钱包
	return s.walletRepo.Delete(ctx, wallet.ID)
}

// GetPrivateKey 获取解密后的私钥（内部使用，不对外暴露）
func (s *WalletService) GetPrivateKey(ctx context.Context, address string) (*ecdsa.PrivateKey, error) {
	// 1. 查询钱包
	wallet, err := s.walletRepo.GetByAddress(ctx, address)
	if err != nil {
		return nil, err
	}

	// 2. 解密私钥
	privateKeyHex, err := utils.DecryptAES(wallet.PrivateKeyEncrypted, s.encryptionKey)
	if err != nil {
		return nil, err
	}

	// 3. 转换为ecdsa.PrivateKey
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, err
	}

	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

// updateBalanceAsync 异步更新余额
func (s *WalletService) updateBalanceAsync(ctx context.Context, address string) {
	balance, err := s.blockchainClient.GetBalance(ctx, address)
	if err != nil {
		logger.Error("failed to update balance",
			zap.String("address", address),
			zap.Error(err),
		)
		return
	}

	// 更新数据库
	if err := s.walletRepo.UpdateBalance(ctx, address, balance.String()); err != nil {
		logger.Error("failed to save balance to database",
			zap.String("address", address),
			zap.Error(err),
		)
	}

	// 更新缓存
	cacheKey := "balance:" + address
	s.cache.Set(ctx, cacheKey, balance.String(), 30)
}
