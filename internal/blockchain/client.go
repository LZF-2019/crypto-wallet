package blockchain

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
)

// BlockchainClient 区块链客户端接口
type BlockchainClient interface {
	// GetBalance 查询地址余额
	GetBalance(ctx context.Context, address string) (*big.Int, error)

	// GetNonce 获取地址的nonce
	GetNonce(ctx context.Context, address string) (uint64, error)

	// GetGasPrice 获取当前gas价格
	GetGasPrice(ctx context.Context) (*big.Int, error)

	// EstimateGas 估算gas用量
	EstimateGas(ctx context.Context, from, to string, value *big.Int) (uint64, error)

	// SendTransaction 发送交易
	SendTransaction(ctx context.Context, signedTx *types.Transaction) error

	// GetTransactionReceipt 获取交易回执
	GetTransactionReceipt(ctx context.Context, txHash string) (*types.Receipt, error)

	// GetBlockNumber 获取最新区块号
	GetBlockNumber(ctx context.Context) (uint64, error)

	// CreateWallet 创建钱包
	CreateWallet() (address string, privateKey *ecdsa.PrivateKey, err error)

	// SignTransaction 签名交易
	SignTransaction(tx *types.Transaction, privateKey *ecdsa.PrivateKey, chainID *big.Int) (*types.Transaction, error)

	// GetChainID 获取链ID
	GetChainID() int
}
