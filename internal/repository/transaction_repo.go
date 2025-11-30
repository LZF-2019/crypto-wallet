package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"crypto-wallet-api/internal/models"
)

// TransactionRepository 交易数据访问层
type TransactionRepository struct {
	db *gorm.DB
}

// NewTransactionRepository 创建交易仓库实例
func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

// Create 创建交易记录
func (r *TransactionRepository) Create(ctx context.Context, tx *models.Transaction) error {
	return r.db.WithContext(ctx).Create(tx).Error
}

// GetByID 根据ID查询交易
func (r *TransactionRepository) GetByID(ctx context.Context, id uint) (*models.Transaction, error) {
	var tx models.Transaction
	err := r.db.WithContext(ctx).First(&tx, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("transaction not found")
		}
		return nil, err
	}
	return &tx, nil
}

// GetByTxHash 根据交易哈希查询
func (r *TransactionRepository) GetByTxHash(ctx context.Context, txHash string) (*models.Transaction, error) {
	var tx models.Transaction
	err := r.db.WithContext(ctx).Where("tx_hash = ?", txHash).First(&tx).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("transaction not found")
		}
		return nil, err
	}
	return &tx, nil
}

// GetByWalletID 查询钱包的所有交易
func (r *TransactionRepository) GetByWalletID(ctx context.Context, walletID uint, page, pageSize int) ([]*models.Transaction, int64, error) {
	var transactions []*models.Transaction
	var total int64

	// 计算总数
	if err := r.db.WithContext(ctx).Model(&models.Transaction{}).Where("wallet_id = ?", walletID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := r.db.WithContext(ctx).
		Where("wallet_id = ?", walletID).
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&transactions).Error

	return transactions, total, err
}

// List 查询交易列表（支持多条件筛选）
func (r *TransactionRepository) List(ctx context.Context, req *models.TransactionListRequest) ([]*models.Transaction, int64, error) {
	var transactions []*models.Transaction
	var total int64

	// 构建查询条件
	query := r.db.WithContext(ctx).Model(&models.Transaction{})

	// 按钱包地址筛选
	if req.WalletAddress != "" {
		// 需要先查询钱包ID
		var wallet models.Wallet
		if err := r.db.WithContext(ctx).Where("address = ?", req.WalletAddress).First(&wallet).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return []*models.Transaction{}, 0, nil
			}
			return nil, 0, err
		}
		query = query.Where("wallet_id = ?", wallet.ID)
	}

	// 按状态筛选
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// 按链ID筛选
	if req.ChainID > 0 {
		query = query.Where("chain_id = ?", req.ChainID)
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	err := query.
		Order("created_at DESC").
		Limit(req.PageSize).
		Offset(offset).
		Find(&transactions).Error

	return transactions, total, err
}

// UpdateStatus 更新交易状态
func (r *TransactionRepository) UpdateStatus(ctx context.Context, txHash string, status models.TransactionStatus, blockNumber int64) error {
	updates := map[string]interface{}{
		"status":       status,
		"block_number": blockNumber,
	}

	// 如果交易成功或失败，记录确认时间
	if status == models.TxStatusSuccess || status == models.TxStatusFailed {
		updates["confirmed_at"] = gorm.Expr("NOW()")
	}

	return r.db.WithContext(ctx).
		Model(&models.Transaction{}).
		Where("tx_hash = ?", txHash).
		Updates(updates).Error
}

// Update 更新交易信息
func (r *TransactionRepository) Update(ctx context.Context, tx *models.Transaction) error {
	return r.db.WithContext(ctx).Save(tx).Error
}

// GetPendingTransactions 查询所有待确认的交易
func (r *TransactionRepository) GetPendingTransactions(ctx context.Context) ([]*models.Transaction, error) {
	var transactions []*models.Transaction
	err := r.db.WithContext(ctx).
		Where("status = ?", models.TxStatusPending).
		Order("created_at ASC").
		Find(&transactions).Error
	return transactions, err
}

// CountByStatus 统计指定状态的交易数量
func (r *TransactionRepository) CountByStatus(ctx context.Context, status models.TransactionStatus) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Transaction{}).Where("status = ?", status).Count(&count).Error
	return count, err
}
