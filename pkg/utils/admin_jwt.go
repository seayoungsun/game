package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kaifa/game-platform/internal/config"
)

// AdminClaims 管理员JWT Claims
type AdminClaims struct {
	AdminID     uint     `json:"admin_id"`
	Username    string   `json:"username"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// GenerateAdminToken 生成管理员JWT Token
func GenerateAdminToken(adminID uint, username string, permissions []string) (string, error) {
	cfg := config.Get()
	if cfg == nil {
		return "", errors.New("配置未加载")
	}

	// 使用JWT密钥（可以和管理员密钥分开，这里先用同一个）
	secretKey := []byte(cfg.JWT.Secret)

	// 创建Claims
	claims := AdminClaims{
		AdminID:     adminID,
		Username:    username,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(cfg.JWT.Expiration) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	// 创建Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签名Token
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ParseAdminToken 解析管理员JWT Token
func ParseAdminToken(tokenString string) (*AdminClaims, error) {
	cfg := config.Get()
	if cfg == nil {
		return nil, errors.New("配置未加载")
	}

	secretKey := []byte(cfg.JWT.Secret)

	// 解析Token
	token, err := jwt.ParseWithClaims(tokenString, &AdminClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("无效的签名方法")
		}
		return secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	// 验证Claims
	if claims, ok := token.Claims.(*AdminClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("无效的Token")
}
