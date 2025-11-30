package handler

import (
	"math/big"
	"net/http"

	"github.com/gin-gonic/gin"

	"crypto-wallet-api/internal/models"
	"crypto-wallet-api/internal/service"
	"crypto-wallet-api/internal/utils"
)

// WalletHandler 钱包处理器
type WalletHandler struct {
	walletService *service.WalletService
}

// NewWalletHandler 创建钱包处理器实例
func NewWalletHandler(walletService *service.WalletService) *WalletHandler {
	return &WalletHandler{
		walletService: walletService,
	}
}

// CreateWallet 创建钱包
// @Summary 创建钱包
// @Description 为当前用户创建新的区块链钱包
// @Tags 钱包
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.WalletCreateRequest true "创建钱包请求"
// @Success 200 {object} utils.Response{data=models.WalletResponse}
// @Failure 400 {object} utils.Response
// @Router /api/v1/wallets [post]
func (h *WalletHandler) CreateWallet(c *gin.Context) {
	// 1. 获取用户ID
	userID, _ := c.Get("user_id")

	// 2. 绑定请求参数
	var req models.WalletCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid request parameters")
		return
	}

	// 3. 调用服务层
	wallet, err := h.walletService.CreateWallet(c.Request.Context(), userID.(uint), &req)
	if err != nil {
		utils.InternalError(c, err)
		return
	}

	// 4. 返回响应
	utils.SuccessWithMessage(c, "wallet created successfully", wallet.ToResponse())
}

// GetWallets 获取钱包列表
// @Summary 获取钱包列表
// @Description 获取当前用户的所有钱包
// @Tags 钱包
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.Response{data=models.WalletListResponse}
// @Failure 401 {object} utils.Response
// @Router /api/v1/wallets [get]
func (h *WalletHandler) GetWallets(c *gin.Context) {
	// 1. 获取用户ID
	userID, _ := c.Get("user_id")

	// 2. 调用服务层
	wallets, err := h.walletService.GetUserWallets(c.Request.Context(), userID.(uint))
	if err != nil {
		utils.DatabaseError(c, err)
		return
	}

	// 3. 转换为响应格式
	walletResponses := make([]*models.WalletResponse, len(wallets))
	for i, wallet := range wallets {
		walletResponses[i] = wallet.ToResponse()
	}

	// 4. 返回响应
	utils.Success(c, &models.WalletListResponse{
		Total:   int64(len(walletResponses)),
		Wallets: walletResponses,
	})
}

// GetWallet 获取钱包详情
// @Summary 获取钱包详情
// @Description 根据地址获取钱包详细信息
// @Tags 钱包
// @Produce json
// @Security BearerAuth
// @Param address path string true "钱包地址"
// @Success 200 {object} utils.Response{data=models.WalletResponse}
// @Failure 404 {object} utils.Response
// @Router /api/v1/wallets/{address} [get]
func (h *WalletHandler) GetWallet(c *gin.Context) {
	// 1. 获取用户ID和钱包地址
	userID, _ := c.Get("user_id")
	address := c.Param("address")

	// 2. 调用服务层
	wallet, err := h.walletService.GetWalletByAddress(c.Request.Context(), userID.(uint), address)
	if err != nil {
		utils.NotFound(c, "wallet not found")
		return
	}

	// 3. 返回响应
	utils.Success(c, wallet.ToResponse())
}

// GetBalance 查询钱包余额
// @Summary 查询钱包余额
// @Description 实时查询钱包在链上的余额
// @Tags 钱包
// @Produce json
// @Security BearerAuth
// @Param address path string true "钱包地址"
// @Success 200 {object} utils.Response{data=map[string]interface{}}
// @Failure 404 {object} utils.Response
// @Router /api/v1/wallets/{address}/balance [get]
func (h *WalletHandler) GetBalance(c *gin.Context) {
	// 1. 获取用户ID和钱包地址
	userID, _ := c.Get("user_id")
	address := c.Param("address")

	// 2. 调用服务层
	balance, err := h.walletService.GetBalance(c.Request.Context(), userID.(uint), address)
	if err != nil {
		utils.BlockchainError(c, err)
		return
	}

	// 3. 返回响应（Wei和Ether两种单位）
	utils.Success(c, gin.H{
		"address":     address,
		"balance_wei": balance.String(),
		"balance_eth": weiToEther(balance),
	})
}

// UpdateWallet 更新钱包信息
// @Summary 更新钱包信息
// @Description 更新钱包名称等信息
// @Tags 钱包
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param address path string true "钱包地址"
// @Param request body map[string]string true "更新信息"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Router /api/v1/wallets/{address} [put]
func (h *WalletHandler) UpdateWallet(c *gin.Context) {
	// 1. 获取用户ID和钱包地址
	userID, _ := c.Get("user_id")
	address := c.Param("address")

	// 2. 绑定请求参数
	var req struct {
		Name string `json:"name" binding:"max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid request parameters")
		return
	}

	// 3. 调用服务层
	if err := h.walletService.UpdateWallet(c.Request.Context(), userID.(uint), address, req.Name); err != nil {
		utils.InternalError(c, err)
		return
	}

	// 4. 返回响应
	utils.SuccessWithMessage(c, "wallet updated successfully", nil)
}

// DeleteWallet 删除钱包
// @Summary 删除钱包
// @Description 删除指定钱包（余额必须为0）
// @Tags 钱包
// @Produce json
// @Security BearerAuth
// @Param address path string true "钱包地址"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Router /api/v1/wallets/{address} [delete]
func (h *WalletHandler) DeleteWallet(c *gin.Context) {
	// 1. 获取用户ID和钱包地址
	userID, _ := c.Get("user_id")
	address := c.Param("address")

	// 2. 调用服务层
	if err := h.walletService.DeleteWallet(c.Request.Context(), userID.(uint), address); err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, utils.CodeInvalidParams, err.Error(), err)
		return
	}

	// 3. 返回响应
	utils.SuccessWithMessage(c, "wallet deleted successfully", nil)
}

// weiToEther 将Wei转换为Ether（辅助函数）
func weiToEther(wei *big.Int) string {
	// 1 Ether = 10^18 Wei
	ether := new(big.Float).SetInt(wei)
	ether.Quo(ether, big.NewFloat(1e18))
	return ether.Text('f', 6) // 保留6位小数
}
