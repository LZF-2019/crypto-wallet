package main

import (
	"context"
	"crypto-wallet-api/internal/logger"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"crypto-wallet-api/internal/blockchain"
	"crypto-wallet-api/internal/config"
	"crypto-wallet-api/internal/models"
	"crypto-wallet-api/internal/repository"
	"crypto-wallet-api/internal/service"
	"crypto-wallet-api/pkg/cache"
	"crypto-wallet-api/pkg/database"
	"crypto-wallet-api/pkg/queue"
)

func main() {
	// 1. 加载配置
	cfg, err := config.Load("./configs/configs.yaml")
	if err != nil {
		log.Fatalf("Failed to load configs: %v", err)
	}

	// 2. 初始化日志
	if err := logger.InitLogger(
		cfg.Log.Level,
		cfg.Log.Output,
		cfg.Log.FilePath,
		cfg.Log.MaxSize,
		cfg.Log.MaxBackups,
		cfg.Log.MaxAge,
	); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Logger.Sync()

	logger.Info("Starting Transaction Monitor Worker...")

	// 3. 连接数据库
	db, err := database.NewPostgresDB(
		cfg.Database.GetDSN(),
		cfg.Database.MaxOpenConns,
		cfg.Database.MaxIdleConns,
		cfg.Database.ConnMaxLifetime,
	)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}

	// 4. 连接Redis
	redisCache, err := cache.NewRedisCache(
		cfg.Redis.GetRedisAddr(),
		cfg.Redis.Password,
		cfg.Redis.DB,
		cfg.Redis.PoolSize,
		cfg.Redis.MinIdleConns,
	)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisCache.Close()

	// 5. 连接RabbitMQ
	mq, err := queue.NewRabbitMQ(cfg.RabbitMQ.GetRabbitMQURL())
	if err != nil {
		logger.Fatal("Failed to connect to RabbitMQ", zap.Error(err))
	}
	defer mq.Close()

	// 6. 初始化区块链客户端
	ethClient, err := blockchain.NewEthereumClient(
		cfg.Blockchain.Ethereum.RPCURL,
		cfg.Blockchain.Ethereum.ChainID,
	)
	if err != nil {
		logger.Fatal("Failed to create Ethereum client", zap.Error(err))
	}

	// 7. 初始化服务
	txRepo := repository.NewTransactionRepository(db)
	walletRepo := repository.NewWalletRepository(db)
	encryptionKey := []byte("12345678901234567890123456789012")
	walletService := service.NewWalletService(walletRepo, ethClient, redisCache, encryptionKey)
	txService := service.NewTransactionService(txRepo, walletRepo, walletService, ethClient, mq)

	// 8. 创建上下文（支持优雅关闭）
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 9. 启动交易监听消费者
	if err := mq.ConsumeWithContext(ctx, "transaction.created", func(body []byte) error {
		var tx models.Transaction
		if err := json.Unmarshal(body, &tx); err != nil {
			logger.Error("Failed to unmarshal transaction", zap.Error(err))
			return err
		}

		logger.Info("Monitoring transaction", zap.String("tx_hash", tx.TxHash))

		// 轮询监听交易状态（最多5分钟）
		for i := 0; i < 60; i++ {
			time.Sleep(5 * time.Second)

			err := txService.MonitorTransaction(ctx, tx.TxHash)
			if err == nil {
				logger.Info("Transaction confirmed", zap.String("tx_hash", tx.TxHash))
				return nil
			}
		}

		logger.Warn("Transaction confirmation timeout", zap.String("tx_hash", tx.TxHash))
		return nil
	}); err != nil {
		logger.Fatal("Failed to start consumer", zap.Error(err))
	}

	// 10. 启动定时任务：扫描待确认交易
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				scanPendingTransactions(ctx, txService)
			}
		}
	}()

	logger.Info("Worker started successfully")

	// 11. 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down worker...")
	cancel()

	// 等待所有goroutine结束
	time.Sleep(2 * time.Second)
	logger.Info("Worker exited")
}

// scanPendingTransactions 扫描待确认的交易
func scanPendingTransactions(ctx context.Context, txService *service.TransactionService) {
	transactions, err := txService.GetPendingTransactions(ctx)
	if err != nil {
		logger.Error("Failed to get pending transactions", zap.Error(err))
		return
	}

	logger.Info("Scanning pending transactions", zap.Int("count", len(transactions)))

	for _, tx := range transactions {
		// 检查交易是否超时（超过10分钟）
		if time.Since(tx.CreatedAt) > 10*time.Minute {
			logger.Warn("Transaction timeout", zap.String("tx_hash", tx.TxHash))
			continue
		}

		// 监听交易状态
		if err := txService.MonitorTransaction(ctx, tx.TxHash); err != nil {
			logger.Debug("Transaction not confirmed yet", zap.String("tx_hash", tx.TxHash))
		}
	}
}
