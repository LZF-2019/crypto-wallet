package utils

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

// CustomValidator 自定义验证器
var CustomValidator *validator.Validate

// InitValidator 初始化验证器
func InitValidator() {
	CustomValidator = validator.New()

	// 注册自定义验证规则
	CustomValidator.RegisterValidation("eth_addr", validateEthAddress)
}

// validateEthAddress 验证以太坊地址格式
func validateEthAddress(fl validator.FieldLevel) bool {
	address := fl.Field().String()
	// 以太坊地址格式：0x开头，后跟40个十六进制字符
	matched, _ := regexp.MatchString(`^0x[0-9a-fA-F]{40}$`, address)
	return matched
}

// ValidateStruct 验证结构体
func ValidateStruct(s interface{}) error {
	return CustomValidator.Struct(s)
}
