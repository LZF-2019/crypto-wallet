package main

import (
	"context"

	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"crypto-wallet-api/internal/blockchain"
	"crypto-wallet-api/internal/config"
	"crypto-wallet-api/internal/handler"
	"crypto-wallet-api/internal/logger"
	"crypto-wallet-api/internal/middleware"
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

	logger.Info("Starting CryptoWallet API Server...")

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
	logger.Info("Database connected successfully")

	// 4. 自动迁移数据库表结构
	if err := database.AutoMigrate(db); err != nil {
		logger.Fatal("Failed to migrate database", zap.Error(err))
	}
	logger.Info("Database migrated successfully")

	// 5. 连接Redis
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
	logger.Info("Redis connected successfully")

	// 6. 连接RabbitMQ
	mq, err := queue.NewRabbitMQ(cfg.RabbitMQ.GetRabbitMQURL())
	if err != nil {
		logger.Fatal("Failed to connect to RabbitMQ", zap.Error(err))
	}
	defer mq.Close()
	logger.Info("RabbitMQ connected successfully")

	// 7. 初始化区块链客户端
	ethClient, err := blockchain.NewEthereumClient(
		cfg.Blockchain.Ethereum.RPCURL,
		cfg.Blockchain.Ethereum.ChainID,
	)
	if err != nil {
		logger.Fatal("Failed to create Ethereum client", zap.Error(err))
	}
	logger.Info("Ethereum client initialized successfully")

	// 8. 生成加密密钥（实际生产环境应从环境变量或KMS获取）
	encryptionKey := []byte("12345678901234567890123456789012") // 32字节密钥

	// 9. 初始化Repository层
	userRepo := repository.NewUserRepository(db)
	walletRepo := repository.NewWalletRepository(db)
	txRepo := repository.NewTransactionRepository(db)

	// 10. 初始化Service层
	authService := service.NewAuthService(userRepo, cfg.JWT.Secret, cfg.JWT.ExpireHours)
	walletService := service.NewWalletService(walletRepo, ethClient, redisCache, encryptionKey)
	txService := service.NewTransactionService(txRepo, walletRepo, walletService, ethClient, mq)

	// 11. 初始化Handler层
	authHandler := handler.NewAuthHandler(authService)
	walletHandler := handler.NewWalletHandler(walletService)
	txHandler := handler.NewTransactionHandler(txService)

	// 12. 初始化Gin引擎
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()

	// 13. 注册全局中间件
	router.Use(middleware.LoggerMiddleware())
	router.Use(middleware.CORSMiddleware())
	router.Use(gin.Recovery())
	router.Use(middleware.RateLimitMiddleware(
		cfg.RateLimit.RequestsPerSecond,
		cfg.RateLimit.Burst,
	))

	// 14. 注册路由
	setupRoutes(router, authHandler, walletHandler, txHandler, authService)

	// 15. 启动HTTP服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// 16. 优雅关闭
	go func() {
		logger.Info("Server started", zap.String("address", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// 17. 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// 18. 优雅关闭（5秒超时）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

// setupRoutes 设置路由
func setupRoutes(
	router *gin.Engine,
	authHandler *handler.AuthHandler,
	walletHandler *handler.WalletHandler,
	txHandler *handler.TransactionHandler,
	authService *service.AuthService,
) {
	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Unix(),
		})
	})

	// API v1路由组
	v1 := router.Group("/api/v1")
	{
		// 认证路由（无需JWT）
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.GET("/profile", middleware.AuthMiddleware(authService), authHandler.GetProfile)
		}

		// 钱包路由（需要JWT）
		wallets := v1.Group("/wallets")
		wallets.Use(middleware.AuthMiddleware(authService))
		{
			wallets.POST("", walletHandler.CreateWallet)
			wallets.GET("", walletHandler.GetWallets)
			wallets.GET("/:address", walletHandler.GetWallet)
			wallets.GET("/:address/balance", walletHandler.GetBalance)
			wallets.PUT("/:address", walletHandler.UpdateWallet)
			wallets.DELETE("/:address", walletHandler.DeleteWallet)
			wallets.GET("/:address/transactions", txHandler.GetWalletTransactions)
		}

		// 交易路由（需要JWT）
		transactions := v1.Group("/transactions")
		transactions.Use(middleware.AuthMiddleware(authService))
		{
			transactions.POST("", txHandler.SendTransaction)
			transactions.GET("", txHandler.ListTransactions)
			transactions.GET("/:tx_hash", txHandler.GetTransaction)
		}
	}
}
