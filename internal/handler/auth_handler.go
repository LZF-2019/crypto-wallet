package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"crypto-wallet-api/internal/models"
	"crypto-wallet-api/internal/service"
	"crypto-wallet-api/internal/utils"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler 创建认证处理器实例
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register 用户注册
// @Summary 用户注册
// @Description 创建新用户账号
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body models.UserCreateRequest true "注册信息"
// @Success 200 {object} utils.Response{data=models.UserResponse}
// @Failure 400 {object} utils.Response
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	// 1. 绑定请求参数
	var req models.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid request parameters")
		return
	}

	// 2. 调用服务层
	user, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		utils.ErrorWithDetail(c, http.StatusBadRequest, utils.CodeInvalidParams, err.Error(), err)
		return
	}

	// 3. 返回响应
	utils.SuccessWithMessage(c, "registration successful", user.ToResponse())
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录获取JWT Token
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body models.UserLoginRequest true "登录信息"
// @Success 200 {object} utils.Response{data=map[string]interface{}}
// @Failure 400 {object} utils.Response
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	// 1. 绑定请求参数
	var req models.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "invalid request parameters")
		return
	}

	// 2. 调用服务层
	token, user, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		utils.ErrorWithDetail(c, http.StatusUnauthorized, utils.CodeUnauthorized, err.Error(), err)
		return
	}

	// 3. 返回响应（包含Token和用户信息）
	utils.Success(c, gin.H{
		"token": token,
		"user":  user.ToResponse(),
	})
}

// GetProfile 获取用户信息
// @Summary 获取用户信息
// @Description 获取当前登录用户的信息
// @Tags 认证
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.Response{data=models.UserResponse}
// @Failure 401 {object} utils.Response
// @Router /api/v1/auth/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	// 1. 从上下文获取用户ID（由中间件注入）
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "unauthorized")
		return
	}

	// 2. 调用服务层
	user, err := h.authService.GetProfile(c.Request.Context(), userID.(uint))
	if err != nil {
		utils.NotFound(c, "user not found")
		return
	}

	// 3. 返回响应
	utils.Success(c, user.ToResponse())
}
