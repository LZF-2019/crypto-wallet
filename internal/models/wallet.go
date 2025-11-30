package models

import (
	"time"
)

// Wallet 钱包模型
type Wallet struct {
	ID                  uint          `gorm:"primaryKey" json:"id"`
	UserID              uint          `gorm:"not null;index" json:"user_id"`                     // 所属用户ID
	Address             string        `gorm:"unique;not null;size:42;index" json:"address"`      // 钱包地址
	PrivateKeyEncrypted string        `gorm:"not null;type:text" json:"-"`                       // 加密的私钥，不返回给前端
	ChainID             int           `gorm:"not null" json:"chain_id"`                          // 链ID：1=Ethereum, 56=BSC
	Balance             string        `gorm:"type:decimal(36,18);default:0" json:"balance"`      // 余额（字符串避免精度问题）
	Name                string        `gorm:"size:100" json:"name,omitempty"`                    // 钱包名称（可选）
	Transactions        []Transaction `gorm:"foreignKey:WalletID" json:"transactions,omitempty"` // 关联交易
	CreatedAt           time.Time     `json:"created_at"`
	UpdatedAt           time.Time     `json:"updated_at"`
}

// TableName 指定表名
func (Wallet) TableName() string {
	return "wallets"
}

// WalletCreateRequest 创建钱包请求
type WalletCreateRequest struct {
	ChainID int    `json:"chain_id" binding:"required,oneof=1 56 560048"` // 只支持1(Ethereum)和56(BSC) 560048(Hoodi)
	Name    string `json:"name" binding:"max=100"`                        // 可选的钱包名称
}

// WalletResponse 钱包响应
type WalletResponse struct {
	ID        uint      `json:"id"`
	Address   string    `json:"address"`
	ChainID   int       `json:"chain_id"`
	ChainName string    `json:"chain_name"` // 链名称（前端展示用）
	Balance   string    `json:"balance"`
	Name      string    `json:"name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ToResponse 转换为响应格式
func (w *Wallet) ToResponse() *WalletResponse {
	chainName := "Unknown"
	switch w.ChainID {
	case 1:
		chainName = "Ethereum"
	case 56:
		chainName = "BSC"
	case 560048:
		chainName = "Hoodi"
	}

	return &WalletResponse{
		ID:        w.ID,
		Address:   w.Address,
		ChainID:   w.ChainID,
		ChainName: chainName,
		Balance:   w.Balance,
		Name:      w.Name,
		CreatedAt: w.CreatedAt,
	}
}

// WalletListResponse 钱包列表响应
type WalletListResponse struct {
	Total   int64             `json:"total"`
	Wallets []*WalletResponse `json:"wallets"`
}
