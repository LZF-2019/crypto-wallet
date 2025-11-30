package blockchain

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// EthereumClient 以太坊客户端实现
type EthereumClient struct {
	client  *ethclient.Client
	chainID int
}

// NewEthereumClient 创建以太坊客户端
func NewEthereumClient(rpcURL string, chainID int) (*EthereumClient, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}

	return &EthereumClient{
		client:  client,
		chainID: chainID,
	}, nil
}

// GetBalance 查询地址余额
func (c *EthereumClient) GetBalance(ctx context.Context, address string) (*big.Int, error) {
	account := common.HexToAddress(address)
	balance, err := c.client.BalanceAt(ctx, account, nil) // nil表示最新区块
	if err != nil {
		return nil, err
	}
	return balance, nil
}

// GetNonce 获取地址的nonce（交易计数）
func (c *EthereumClient) GetNonce(ctx context.Context, address string) (uint64, error) {
	account := common.HexToAddress(address)
	nonce, err := c.client.PendingNonceAt(ctx, account)
	if err != nil {
		return 0, err
	}
	return nonce, nil
}

// GetGasPrice 获取当前建议的gas价格
func (c *EthereumClient) GetGasPrice(ctx context.Context) (*big.Int, error) {
	gasPrice, err := c.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}
	return gasPrice, nil
}

// EstimateGas 估算交易所需的gas
func (c *EthereumClient) EstimateGas(ctx context.Context, from, to string, value *big.Int) (uint64, error) {
	fromAddr := common.HexToAddress(from)
	toAddr := common.HexToAddress(to)

	// 构建消息
	msg := ethereum.CallMsg{
		From:  fromAddr,
		To:    &toAddr,
		Value: value,
	}

	// 估算gas
	gasLimit, err := c.client.EstimateGas(ctx, msg)
	if err != nil {
		return 0, err
	}

	return gasLimit, nil
}

// SendTransaction 发送已签名的交易
func (c *EthereumClient) SendTransaction(ctx context.Context, signedTx *types.Transaction) error {
	return c.client.SendTransaction(ctx, signedTx)
}

// GetTransactionReceipt 获取交易回执（确认交易状态）
func (c *EthereumClient) GetTransactionReceipt(ctx context.Context, txHash string) (*types.Receipt, error) {
	hash := common.HexToHash(txHash)
	receipt, err := c.client.TransactionReceipt(ctx, hash)
	if err != nil {
		return nil, err
	}
	return receipt, nil
}

// GetBlockNumber 获取最新区块号
func (c *EthereumClient) GetBlockNumber(ctx context.Context) (uint64, error) {
	header, err := c.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, err
	}
	return header.Number.Uint64(), nil
}

// CreateWallet 创建新钱包（生成私钥和地址）
func (c *EthereumClient) CreateWallet() (address string, privateKey *ecdsa.PrivateKey, err error) {
	// 生成私钥
	privateKey, err = crypto.GenerateKey()
	if err != nil {
		return "", nil, err
	}

	// 从私钥导出公钥
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", nil, errors.New("error casting public key to ECDSA")
	}

	// 从公钥生成地址
	address = crypto.PubkeyToAddress(*publicKeyECDSA).Hex()

	return address, privateKey, nil
}

// SignTransaction 签名交易
func (c *EthereumClient) SignTransaction(tx *types.Transaction, privateKey *ecdsa.PrivateKey, chainID *big.Int) (*types.Transaction, error) {
	// 使用EIP-155签名（防重放攻击）
	signer := types.NewEIP155Signer(chainID)
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		return nil, err
	}
	return signedTx, nil
}

// GetChainID 获取链ID
func (c *EthereumClient) GetChainID() int {
	return c.chainID
}

// Close 关闭客户端连接
func (c *EthereumClient) Close() {
	c.client.Close()
}
