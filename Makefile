.PHONY: help build run test clean docker-up docker-down migrate

# 默认目标
help:
	@echo "CryptoWallet API - Makefile Commands"
	@echo "===================================="
	@echo "build         - 编译项目"
	@echo "run-api       - 运行API服务"
	@echo "run-worker    - 运行Worker服务"
	@echo "test          - 运行测试"
	@echo "clean         - 清理编译文件"
	@echo "docker-up     - 启动Docker容器"
	@echo "docker-down   - 停止Docker容器"
	@echo "migrate       - 运行数据库迁移"
	@echo "lint          - 代码检查"
	@echo "fmt           - 格式化代码"

# 编译项目
build:
	@echo "Building API server..."
	@go build -o bin/server ./cmd/server
	@echo "Building Worker..."
	@go build -o bin/worker ./cmd/worker
	@echo "Build completed!"

# 运行API服务
run-api:
	@echo "Starting API server..."
	@go run ./cmd/server/main.go

# 运行Worker服务
run-worker:
	@echo "Starting Worker..."
	@go run ./cmd/worker/main.go

# 运行测试
test:
	@echo "Running tests..."
	@go test -v -cover ./...

# 清理编译文件
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -rf logs/
	@echo "Clean completed!"

# 启动Docker容器
docker-up:
	@echo "Starting Docker containers..."
	@docker-compose up -d
	@echo "Containers started!"
	@docker-compose ps

# 停止Docker容器
docker-down:
	@echo "Stopping Docker containers..."
	@docker-compose down
	@echo "Containers stopped!"

# 重启Docker容器
docker-restart:
	@make docker-down
	@make docker-up

# 查看Docker日志
docker-logs:
	@docker-compose logs -f

# 运行数据库迁移
migrate:
	@echo "Running database migrations..."
	@docker exec -i cryptowallet-postgres psql -U postgres -d cryptowallet < migrations/001_init.sql
	@echo "Migration completed!"

# 代码检查
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

# 格式化代码
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w .

# 安装依赖
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# 生成Swagger文档
swagger:
	@echo "Generating Swagger documentation..."
	@swag init -g cmd/server/main.go

# 运行开发环境
dev:
	@make docker-up
	@sleep 5
	@make migrate
	@make run-api

# 生成加密密钥
gen-key:
	@echo "Generating encryption key..."
	@openssl rand -hex 32
