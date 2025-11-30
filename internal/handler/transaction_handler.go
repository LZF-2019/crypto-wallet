package handler

import (
	"github.com/gin-gonic/gin"

	"crypto-wallet-api/internal/models"
	"crypto-wallet-api/internal/service"
	"crypto-wallet-api/internal/utils"
)

// TransactionHandler 交易处理器
type TransactionHandler struct {
	txService *service.TransactionService
}

// NewTransactionHandler 创建交易处理器实例
func NewTransactionHandler(txService *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{
		txService: txService,
	}
}

// SendTransaction 发起转账
// @Summary 发起转账
// @Description 创建并发送区块链转账交易
// @Tags 交易
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.TransactionCreateRequest true "转账请求"
// @Success 200 {object} utils.Response{data=models.TransactionResponse}
// @Failure 400 {object} utils.Response
// @Router /api/v1/transactions [post]
func (h *TransactionHandler) SendTransaction(c *gin.Context) {
	// 1. 获取用户ID
	userID, _ := c.Get("user_id")

	// 2. 绑定请求参数
	var req models.TransactionCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid request parameters")
		return
	}

	// 3. 调用服务层
	tx, err := h.txService.SendTransaction(c.Request.Context(), userID.(uint), &req)
	if err != nil {
		utils.BlockchainError(c, err)
		return
	}

	// 4. 返回响应
	utils.SuccessWithMessage(c, "transaction sent successfully", tx.ToResponse())
}

// GetTransaction 获取交易详情
// @Summary 获取交易详情
// @Description 根据交易哈希获取交易详细信息
// @Tags 交易
// @Produce json
// @Security BearerAuth
// @Param tx_hash path string true "交易哈希"
// @Success 200 {object} utils.Response{data=models.TransactionResponse}
// @Failure 404 {object} utils.Response
// @Router /api/v1/transactions/{tx_hash} [get]
func (h *TransactionHandler) GetTransaction(c *gin.Context) {
	// 1. 获取用户ID和交易哈希
	userID, _ := c.Get("user_id")
	txHash := c.Param("tx_hash")

	// 2. 调用服务层
	tx, err := h.txService.GetTransaction(c.Request.Context(), userID.(uint), txHash)
	if err != nil {
		utils.NotFound(c, "transaction not found")
		return
	}

	// 3. 返回响应
	utils.Success(c, tx.ToResponse())
}

// ListTransactions 查询交易列表
// @Summary 查询交易列表
// @Description 查询用户的交易记录（支持分页和筛选）
// @Tags 交易
// @Produce json
// @Security BearerAuth
// @Param wallet_address query string false "钱包地址"
// @Param status query string false "交易状态" Enums(pending, success, failed)
// @Param chain_id query int false "链ID" Enums(1, 56)
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} utils.Response{data=models.TransactionListResponse}
// @Failure 400 {object} utils.Response
// @Router /api/v1/transactions [get]
func (h *TransactionHandler) ListTransactions(c *gin.Context) {
	// 1. 获取用户ID
	userID, _ := c.Get("user_id")

	// 2. 绑定查询参数
	var req models.TransactionListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequest(c, "invalid query parameters")
		return
	}

	// 3. 调用服务层
	resp, err := h.txService.ListTransactions(c.Request.Context(), userID.(uint), &req)
	if err != nil {
		utils.DatabaseError(c, err)
		return
	}

	// 4. 返回响应
	utils.Success(c, resp)
}

// GetWalletTransactions 获取指定钱包的交易记录
// @Summary 获取钱包交易记录
// @Description 获取指定钱包地址的所有交易记录
// @Tags 交易
// @Produce json
// @Security BearerAuth
// @Param address path string true "钱包地址"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} utils.Response{data=models.TransactionListResponse}
// @Failure 400 {object} utils.Response
// @Router /api/v1/wallets/{address}/transactions [get]
func (h *TransactionHandler) GetWalletTransactions(c *gin.Context) {
	// 1. 获取用户ID和钱包地址
	userID, _ := c.Get("user_id")
	address := c.Param("address")

	// 2. 绑定分页参数
	var req models.TransactionListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequest(c, "invalid query parameters")
		return
	}
	req.WalletAddress = address

	// 3. 调用服务层
	resp, err := h.txService.ListTransactions(c.Request.Context(), userID.(uint), &req)
	if err != nil {
		utils.DatabaseError(c, err)
		return
	}

	// 4. 返回响应
	utils.Success(c, resp)
}
