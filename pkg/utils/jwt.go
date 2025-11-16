package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kaifa/game-platform/internal/config"
)

// JWTClaims JWT声明
type JWTClaims struct {
	UserID uint   `json:"user_id"`
	UID    int64  `json:"uid"`
	Phone  string `json:"phone"`
	jwt.RegisteredClaims
}

// GenerateToken 生成JWT Token
func GenerateToken(userID uint, uid int64, phone string) (string, error) {
	cfg := config.Get()

	now := time.Now()
	expiresIn := time.Duration(cfg.JWT.Expiration) * time.Hour

	claims := JWTClaims{
		UserID: userID,
		UID:    uid,
		Phone:  phone,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "game-platform",
			Subject:   string(rune(userID)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWT.Secret))
}

// ParseToken 解析JWT Token
func ParseToken(tokenString string) (*JWTClaims, error) {
	cfg := config.Get()

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("无效的签名方法")
		}
		return []byte(cfg.JWT.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("无效的token")
}

// ValidateToken 验证Token有效性
func ValidateToken(tokenString string) bool {
	_, err := ParseToken(tokenString)
	return err == nil
}
