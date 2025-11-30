package service

import (
	"context"

	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"go.uber.org/zap"

	"crypto-wallet-api/internal/blockchain"
	"crypto-wallet-api/internal/logger"
	"crypto-wallet-api/internal/models"
	"crypto-wallet-api/internal/repository"
	"crypto-wallet-api/internal/utils"
	"crypto-wallet-api/pkg/queue"
)

// TransactionService 交易服务
type TransactionService struct {
	txRepo           *repository.TransactionRepository
	walletRepo       *repository.WalletRepository
	walletService    *WalletService
	blockchainClient blockchain.BlockchainClient
	queue            *queue.RabbitMQ
}

// NewTransactionService 创建交易服务实例
func NewTransactionService(
	txRepo *repository.TransactionRepository,
	walletRepo *repository.WalletRepository,
	walletService *WalletService,
	blockchainClient blockchain.BlockchainClient,
	queue *queue.RabbitMQ,
) *TransactionService {
	return &TransactionService{
		txRepo:           txRepo,
		walletRepo:       walletRepo,
		walletService:    walletService,
		blockchainClient: blockchainClient,
		queue:            queue,
	}
}

// SendTransaction 发起转账交易
func (s *TransactionService) SendTransaction(ctx context.Context, userID uint, req *models.TransactionCreateRequest) (*models.Transaction, error) {
	// 1. 验证发送方钱包所有权
	wallet, err := s.walletRepo.GetByAddress(ctx, req.FromAddress)
	if err != nil {
		return nil, err
	}
	if wallet.UserID != userID {
		return nil, errors.New("wallet not found")
	}

	// 2. 验证链ID匹配
	if wallet.ChainID != req.ChainID {
		return nil, errors.New("chain_id mismatch")
	}

	// 3. 检查余额是否充足
	balance, err := s.walletService.GetBalance(ctx, userID, req.FromAddress)
	if err != nil {
		return nil, err
	}

	// 转换金额
	amount := new(big.Int)
	amount.SetString(req.Amount, 10)

	// 获取gas价格
	gasPrice, err := s.blockchainClient.GetGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	// 设置gas limit（如果未指定，使用默认值21000）
	gasLimit := req.GasLimit
	if gasLimit == 0 {
		gasLimit = 21000
	}

	// 计算总费用：amount + gas费用
	gasFee := new(big.Int).Mul(gasPrice, big.NewInt(gasLimit))
	totalCost := new(big.Int).Add(amount, gasFee)

	if balance.Cmp(totalCost) < 0 {
		return nil, errors.New("insufficient balance")
	}

	// 4. 获取私钥
	privateKey, err := s.walletService.GetPrivateKey(ctx, req.FromAddress)
	if err != nil {
		return nil, err
	}

	// 5. 获取nonce
	nonce, err := s.blockchainClient.GetNonce(ctx, req.FromAddress)
	if err != nil {
		return nil, err
	}

	// 6. 构建交易
	toAddress := common.HexToAddress(req.ToAddress)
	tx := types.NewTransaction(
		nonce,
		toAddress,
		amount,
		uint64(gasLimit),
		gasPrice,
		nil, // data字段为空（普通转账）
	)

	// 7. 签名交易
	chainID := big.NewInt(int64(wallet.ChainID))
	signedTx, err := s.blockchainClient.SignTransaction(tx, privateKey, chainID)
	if err != nil {
		return nil, err
	}

	// 8. 发送交易到链上
	if err := s.blockchainClient.SendTransaction(ctx, signedTx); err != nil {
		return nil, err
	}

	// 9. 保存交易记录到数据库
	transaction := &models.Transaction{
		WalletID:    wallet.ID,
		TxHash:      signedTx.Hash().Hex(),
		FromAddress: req.FromAddress,
		ToAddress:   req.ToAddress,
		Amount:      utils.WeiToEthString(amount),
		GasPrice:    gasPrice.String(),
		GasLimit:    gasLimit,
		Nonce:       nonce,
		Status:      models.TxStatusPending,
		ChainID:     wallet.ChainID,
	}

	if err := s.txRepo.Create(ctx, transaction); err != nil {
		return nil, err
	}

	// 10. 发送消息到队列（异步监听交易状态）
	if err := s.queue.Publish("transaction.created", transaction); err != nil {
		logger.Warn("failed to publish transaction to queue",
			zap.String("tx_hash", transaction.TxHash),
			zap.Error(err),
		)
	}

	return transaction, nil
}

// GetTransaction 获取交易详情
func (s *TransactionService) GetTransaction(ctx context.Context, userID uint, txHash string) (*models.Transaction, error) {
	// 1. 查询交易
	tx, err := s.txRepo.GetByTxHash(ctx, txHash)
	if err != nil {
		return nil, err
	}

	// 2. 验证所有权
	wallet, err := s.walletRepo.GetByID(ctx, tx.WalletID)
	if err != nil {
		return nil, err
	}
	if wallet.UserID != userID {
		return nil, errors.New("transaction not found")
	}

	return tx, nil
}

// ListTransactions 查询交易列表
func (s *TransactionService) ListTransactions(ctx context.Context, userID uint, req *models.TransactionListRequest) (*models.TransactionListResponse, error) {
	// 1. 如果指定了钱包地址，验证所有权
	if req.WalletAddress != "" {
		wallet, err := s.walletRepo.GetByAddress(ctx, req.WalletAddress)
		if err != nil {
			return nil, err
		}
		if wallet.UserID != userID {
			return nil, errors.New("wallet not found")
		}
	} else {
		// 2. 如果未指定钱包地址，查询用户所有钱包的交易
		wallets, err := s.walletRepo.GetByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}

		// 构建钱包ID列表（这里简化处理，实际应优化查询）
		if len(wallets) > 0 {
			req.WalletAddress = wallets[0].Address // 临时方案
		}
	}

	// 3. 查询交易列表
	transactions, total, err := s.txRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}

	// 4. 转换为响应格式
	txResponses := make([]*models.TransactionResponse, len(transactions))
	for i, tx := range transactions {
		txResponses[i] = tx.ToResponse()
	}

	return &models.TransactionListResponse{
		Total:        total,
		Page:         req.Page,
		PageSize:     req.PageSize,
		Transactions: txResponses,
	}, nil
}

// MonitorTransaction 监听交易状态（后台任务调用）
func (s *TransactionService) MonitorTransaction(ctx context.Context, txHash string) error {
	// 1. 查询交易回执
	receipt, err := s.blockchainClient.GetTransactionReceipt(ctx, txHash)
	if err != nil {
		// 交易尚未确认
		return err
	}

	// 2. 判断交易状态
	status := models.TxStatusFailed
	if receipt.Status == 1 {
		status = models.TxStatusSuccess
	}

	// 3. 更新交易状态
	if err := s.txRepo.UpdateStatus(ctx, txHash, status, receipt.BlockNumber.Int64()); err != nil {
		return err
	}

	// 4. 如果交易成功，更新钱包余额
	if status == models.TxStatusSuccess {
		tx, err := s.txRepo.GetByTxHash(ctx, txHash)
		if err != nil {
			return err
		}

		wallet, err := s.walletRepo.GetByID(ctx, tx.WalletID)
		if err != nil {
			return err
		}

		// 异步更新余额
		go s.walletService.updateBalanceAsync(context.Background(), wallet.Address)
	}

	logger.Info("transaction confirmed",
		zap.String("tx_hash", txHash),
		zap.String("status", string(status)),
		zap.Int64("block_number", receipt.BlockNumber.Int64()),
	)

	return nil
}

// GetPendingTransactions 获取所有待确认的交易
func (s *TransactionService) GetPendingTransactions(ctx context.Context) ([]*models.Transaction, error) {
	return s.txRepo.GetPendingTransactions(ctx)
}
