package service

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"crypto-wallet-api/internal/models"
	"crypto-wallet-api/internal/repository"
)

// AuthService 认证服务
type AuthService struct {
	userRepo  *repository.UserRepository
	jwtSecret string
	jwtExpire int // 小时
}

// NewAuthService 创建认证服务实例
func NewAuthService(userRepo *repository.UserRepository, jwtSecret string, jwtExpire int) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
		jwtExpire: jwtExpire,
	}
}

// Register 用户注册
func (s *AuthService) Register(ctx context.Context, req *models.UserCreateRequest) (*models.User, error) {
	// 1. 检查邮箱是否已存在
	exists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("email already exists")
	}

	// 2. 检查用户名是否已存在
	exists, err = s.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("username already exists")
	}

	// 3. 创建用户对象
	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
	}

	// 4. 加密密码
	if err := user.SetPassword(req.Password); err != nil {
		return nil, err
	}

	// 5. 保存到数据库
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// Login 用户登录
func (s *AuthService) Login(ctx context.Context, req *models.UserLoginRequest) (string, *models.User, error) {
	// 1. 根据邮箱查询用户
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return "", nil, errors.New("invalid email or password")
	}

	// 2. 验证密码
	if !user.CheckPassword(req.Password) {
		return "", nil, errors.New("invalid email or password")
	}

	// 3. 生成JWT Token
	token, err := s.GenerateToken(user.ID)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}

// GenerateToken 生成JWT Token
func (s *AuthService) GenerateToken(userID uint) (string, error) {
	// 创建Claims
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * time.Duration(s.jwtExpire)).Unix(), // 过期时间
		"iat":     time.Now().Unix(),                                             // 签发时间
	}

	// 创建Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签名
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken 验证JWT Token
func (s *AuthService) ValidateToken(tokenString string) (uint, error) {
	// 解析Token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("invalid signing method")
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return 0, err
	}

	// 验证Token有效性
	if !token.Valid {
		return 0, errors.New("invalid token")
	}

	// 提取Claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errors.New("invalid token claims")
	}

	// 提取用户ID
	userID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, errors.New("invalid user_id in token")
	}

	return uint(userID), nil
}

// GetProfile 获取用户信息
func (s *AuthService) GetProfile(ctx context.Context, userID uint) (*models.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}
