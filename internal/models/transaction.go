package models

import (
	"time"
)

// TransactionStatus 交易状态枚举
type TransactionStatus string

const (
	TxStatusPending   TransactionStatus = "pending"   // 待确认
	TxStatusSuccess   TransactionStatus = "success"   // 成功
	TxStatusFailed    TransactionStatus = "failed"    // 失败
	TxStatusCancelled TransactionStatus = "cancelled" // 已取消
)

// Transaction 交易模型
type Transaction struct {
	ID          uint              `gorm:"primaryKey" json:"id"`
	WalletID    uint              `gorm:"not null;index" json:"wallet_id"`              // 所属钱包ID
	TxHash      string            `gorm:"unique;not null;size:66;index" json:"tx_hash"` // 交易哈希
	FromAddress string            `gorm:"not null;size:42" json:"from_address"`         // 发送方地址
	ToAddress   string            `gorm:"not null;size:42" json:"to_address"`           // 接收方地址
	Amount      string            `gorm:"type:decimal(36,18);not null" json:"amount"`   // 转账金额
	GasPrice    string            `gorm:"type:decimal(36,18)" json:"gas_price"`         // Gas价格
	GasUsed     int64             `json:"gas_used"`                                     // 实际使用的Gas
	GasLimit    int64             `json:"gas_limit"`                                    // Gas限制
	Nonce       uint64            `json:"nonce"`                                        // 交易nonce
	Status      TransactionStatus `gorm:"not null;index;size:20" json:"status"`         // 交易状态
	BlockNumber int64             `json:"block_number"`                                 // 区块号
	ChainID     int               `gorm:"not null" json:"chain_id"`                     // 链ID
	ErrorMsg    string            `gorm:"type:text" json:"error_msg,omitempty"`         // 错误信息（失败时）
	CreatedAt   time.Time         `json:"created_at"`                                   // 创建时间
	ConfirmedAt *time.Time        `json:"confirmed_at,omitempty"`                       // 确认时间
}

// TableName 指定表名
func (Transaction) TableName() string {
	return "transactions"
}

// TransactionCreateRequest 创建交易请求
type TransactionCreateRequest struct {
	FromAddress string `json:"from_address" binding:"required,eth_addr"` // 自定义验证器：eth_addr
	ToAddress   string `json:"to_address" binding:"required,eth_addr"`
	Amount      string `json:"amount" binding:"required,numeric,gt=0"` // 金额必须大于0
	ChainID     int    `json:"chain_id" binding:"required,oneof=1 56 560048"`
	GasLimit    int64  `json:"gas_limit" binding:"omitempty,gt=0"` // 可选，默认21000
}

// TransactionResponse 交易响应
type TransactionResponse struct {
	ID          uint              `json:"id"`
	TxHash      string            `json:"tx_hash"`
	FromAddress string            `json:"from_address"`
	ToAddress   string            `json:"to_address"`
	Amount      string            `json:"amount"`
	GasPrice    string            `json:"gas_price"`
	GasUsed     int64             `json:"gas_used"`
	Status      TransactionStatus `json:"status"`
	BlockNumber int64             `json:"block_number"`
	ChainID     int               `json:"chain_id"`
	ChainName   string            `json:"chain_name"`
	CreatedAt   time.Time         `json:"created_at"`
	ConfirmedAt *time.Time        `json:"confirmed_at,omitempty"`
}

// ToResponse 转换为响应格式
func (t *Transaction) ToResponse() *TransactionResponse {
	chainName := "Unknown"
	switch t.ChainID {
	case 1:
		chainName = "Ethereum"
	case 56:
		chainName = "BSC"
	case 560048:
		chainName = "Hoodi"
	}

	return &TransactionResponse{
		ID:          t.ID,
		TxHash:      t.TxHash,
		FromAddress: t.FromAddress,
		ToAddress:   t.ToAddress,
		Amount:      t.Amount,
		GasPrice:    t.GasPrice,
		GasUsed:     t.GasUsed,
		Status:      t.Status,
		BlockNumber: t.BlockNumber,
		ChainID:     t.ChainID,
		ChainName:   chainName,
		CreatedAt:   t.CreatedAt,
		ConfirmedAt: t.ConfirmedAt,
	}
}

// TransactionListRequest 交易列表查询请求
type TransactionListRequest struct {
	WalletAddress string            `form:"wallet_address" binding:"omitempty,eth_addr"`             // 按钱包地址筛选
	Status        TransactionStatus `form:"status" binding:"omitempty,oneof=pending success failed"` // 按状态筛选
	ChainID       int               `form:"chain_id" binding:"omitempty,oneof=1 56 560048"`          // 按链筛选
	Page          int               `form:"page" binding:"omitempty,min=1"`                          // 页码，默认1
	PageSize      int               `form:"page_size" binding:"omitempty,min=1,max=100"`             // 每页数量，默认20
}

// TransactionListResponse 交易列表响应
type TransactionListResponse struct {
	Total        int64                  `json:"total"`
	Page         int                    `json:"page"`
	PageSize     int                    `json:"page_size"`
	Transactions []*TransactionResponse `json:"transactions"`
}
