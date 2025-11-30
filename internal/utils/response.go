package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`            // 业务状态码：0表示成功，非0表示失败
	Message string      `json:"message"`         // 响应消息
	Data    interface{} `json:"data,omitempty"`  // 响应数据
	Error   string      `json:"error,omitempty"` // 错误详情（仅开发环境）
}

// 业务状态码定义
const (
	CodeSuccess             = 0     // 成功
	CodeInvalidParams       = 10001 // 参数错误
	CodeUnauthorized        = 10002 // 未授权
	CodeForbidden           = 10003 // 禁止访问
	CodeNotFound            = 10004 // 资源不存在
	CodeInternalError       = 10005 // 内部错误
	CodeDatabaseError       = 10006 // 数据库错误
	CodeBlockchainError     = 10007 // 区块链交互错误
	CodeInsufficientBalance = 10008 // 余额不足
	CodeDuplicateResource   = 10009 // 资源重复
)

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 成功响应（自定义消息）
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: message,
		Data:    data,
	})
}

// ErrorJson 错误响应
func ErrorJson(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
	})
}

// ErrorWithDetail 错误响应（包含详细错误信息）
func ErrorWithDetail(c *gin.Context, httpStatus int, code int, message string, err error) {
	resp := Response{
		Code:    code,
		Message: message,
	}

	// 开发环境返回详细错误
	if gin.Mode() == gin.DebugMode && err != nil {
		resp.Error = err.Error()
	}

	c.JSON(httpStatus, resp)
}

// BadRequest 400错误
func BadRequest(c *gin.Context, message string) {
	ErrorJson(c, http.StatusBadRequest, CodeInvalidParams, message)
}

// Unauthorized 401错误
func Unauthorized(c *gin.Context, message string) {
	ErrorJson(c, http.StatusUnauthorized, CodeUnauthorized, message)
}

// Forbidden 403错误
func Forbidden(c *gin.Context, message string) {
	ErrorJson(c, http.StatusForbidden, CodeForbidden, message)
}

// NotFound 404错误
func NotFound(c *gin.Context, message string) {
	ErrorJson(c, http.StatusNotFound, CodeNotFound, message)
}

// InternalError 500错误
func InternalError(c *gin.Context, err error) {
	ErrorWithDetail(c, http.StatusInternalServerError, CodeInternalError, "internal server error", err)
}

// DatabaseError 数据库错误
func DatabaseError(c *gin.Context, err error) {
	ErrorWithDetail(c, http.StatusInternalServerError, CodeDatabaseError, "database error", err)
}

// BlockchainError 区块链错误
func BlockchainError(c *gin.Context, err error) {
	ErrorWithDetail(c, http.StatusBadGateway, CodeBlockchainError, "blockchain interaction error", err)
}
