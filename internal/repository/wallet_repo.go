package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"crypto-wallet-api/internal/models"
)

// WalletRepository 钱包数据访问层
type WalletRepository struct {
	db *gorm.DB
}

// NewWalletRepository 创建钱包仓库实例
func NewWalletRepository(db *gorm.DB) *WalletRepository {
	return &WalletRepository{db: db}
}

// Create 创建钱包
func (r *WalletRepository) Create(ctx context.Context, wallet *models.Wallet) error {
	return r.db.WithContext(ctx).Create(wallet).Error
}

// GetByID 根据ID查询钱包
func (r *WalletRepository) GetByID(ctx context.Context, id uint) (*models.Wallet, error) {
	var wallet models.Wallet
	err := r.db.WithContext(ctx).First(&wallet, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("wallet not found")
		}
		return nil, err
	}
	return &wallet, nil
}

// GetByAddress 根据地址查询钱包
func (r *WalletRepository) GetByAddress(ctx context.Context, address string) (*models.Wallet, error) {
	var wallet models.Wallet
	err := r.db.WithContext(ctx).Where("address = ?", address).First(&wallet).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("wallet not found")
		}
		return nil, err
	}
	return &wallet, nil
}

// GetByUserID 查询用户的所有钱包
func (r *WalletRepository) GetByUserID(ctx context.Context, userID uint) ([]*models.Wallet, error) {
	var wallets []*models.Wallet
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&wallets).Error
	return wallets, err
}

// GetByUserIDAndChainID 查询用户在指定链上的钱包
func (r *WalletRepository) GetByUserIDAndChainID(ctx context.Context, userID uint, chainID int) ([]*models.Wallet, error) {
	var wallets []*models.Wallet
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND chain_id = ?", userID, chainID).
		Order("created_at DESC").
		Find(&wallets).Error
	return wallets, err
}

// UpdateBalance 更新钱包余额
func (r *WalletRepository) UpdateBalance(ctx context.Context, address string, balance string) error {
	return r.db.WithContext(ctx).
		Model(&models.Wallet{}).
		Where("address = ?", address).
		Update("balance", balance).Error
}

// Update 更新钱包信息
func (r *WalletRepository) Update(ctx context.Context, wallet *models.Wallet) error {
	return r.db.WithContext(ctx).Save(wallet).Error
}

// Delete 删除钱包
func (r *WalletRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Wallet{}, id).Error
}

// ExistsByAddress 检查地址是否已存在
func (r *WalletRepository) ExistsByAddress(ctx context.Context, address string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Wallet{}).Where("address = ?", address).Count(&count).Error
	return count > 0, err
}

// Count 统计用户钱包数量
func (r *WalletRepository) Count(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Wallet{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}
