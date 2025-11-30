# CryptoWallet API Service

一个基于Go语言开发的多链钱包管理后端服务，支持以太坊、BSC等区块链，提供钱包创建、余额查询、转账等核心功能。

## 功能特性

- ✅ 用户注册/登录（JWT认证）
- ✅ 多链钱包管理（Ethereum、BSC）
- ✅ 钱包创建与私钥加密存储
- ✅ 实时余额查询（Redis缓存）
- ✅ 转账交易（自动签名与发送）
- ✅ 交易状态监听（RabbitMQ异步处理）
- ✅ RESTful API设计
- ✅ 完整的日志与监控
- ✅ Docker容器化部署

## 技术栈

- **语言**: Go 1.21
- **Web框架**: Gin
- **数据库**: PostgreSQL + GORM
- **缓存**: Redis
- **消息队列**: RabbitMQ
- **区块链**: go-ethereum (geth)
- **日志**: Zap
- **配置**: Viper

## 项目结构
crypto-wallet-api/
├── cmd/
│   ├── server/
│   │   └── main.go                 # API服务入口
│   └── worker/
│       └── main.go                 # 后台任务Worker入口
├── internal/
│   ├── config/
│   │   └── config.go               # 配置管理
│   ├── models/
│   │   ├── user.go                 # 用户模型
│   │   ├── wallet.go               # 钱包模型
│   │   └── transaction.go          # 交易模型
│   ├── repository/
│   │   ├── user_repo.go            # 用户数据访问层
│   │   ├── wallet_repo.go          # 钱包数据访问层
│   │   └── transaction_repo.go     # 交易数据访问层
│   ├── service/
│   │   ├── auth_service.go         # 认证服务
│   │   ├── wallet_service.go       # 钱包服务
│   │   └── transaction_service.go  # 交易服务
│   ├── handler/
│   │   ├── auth_handler.go         # 认证HTTP处理器
│   │   ├── wallet_handler.go       # 钱包HTTP处理器
│   │   └── transaction_handler.go  # 交易HTTP处理器
│   ├── middleware/
│   │   ├── auth.go                 # JWT认证中间件
│   │   ├── logger.go               # 日志中间件
│   │   ├── rate_limit.go           # 限流中间件
│   │   └── cors.go                 # CORS中间件
│   ├── blockchain/
│   │   ├── client.go               # 区块链客户端接口
│   │   └── ethereum.go             # 以太坊实现
│   └── utils/
│       ├── crypto.go               # 加密工具
│       ├── response.go             # 统一响应格式
│       ├── validator.go            # 参数验证
│       └── logger.go               # 日志工具
├── pkg/
│   ├── cache/
│   │   └── redis.go                # Redis缓存封装
│   ├── queue/
│   │   └── rabbitmq.go             # RabbitMQ封装
│   └── database/
│       └── postgres.go             # PostgreSQL连接
├── configs/
│   └── configs.yaml                 # 配置文件
├── migrations/
│   └── 001_init.sql                # 数据库初始化脚本
├── scripts/
│   ├── build.sh                    # 编译脚本
│   └── deploy.sh                   # 部署脚本
├── docker-compose.yml              # Docker编排
├── Dockerfile                      # Docker镜像
├── Makefile                        # Make命令
├── go.mod                          # Go模块依赖
├── go.sum                          # 依赖校验
└── README.md                       # 项目文档
