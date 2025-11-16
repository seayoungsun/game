package utils

import (
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword 加密密码
func HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// CheckPassword 验证密码
func CheckPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// GenerateRandomString 生成随机字符串
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// GenerateUID 生成用户ID（使用雪花算法）
// 特点：唯一、有序、高性能、分布式友好
func GenerateUID() (int64, error) {
	snowflake := GetSnowflakeGenerator()
	if snowflake == nil {
		// 如果雪花算法未初始化，降级到随机算法
		return generateRandomUID()
	}

	return snowflake.GenerateID()
}

// generateRandomUID 生成随机UID（降级方案）
func generateRandomUID() (int64, error) {
	buf := make([]byte, 4) // 使用4字节（32位），生成更短的ID
	if _, err := rand.Read(buf); err != nil {
		return 0, err
	}

	var num uint32
	for i := 0; i < 4; i++ {
		num = num<<8 | uint32(buf[i])
	}

	// 映射到 [100000000, 999999999] (9位数字)
	uid := int64(100000000 + (num % 900000000))

	return uid, nil
}

// GenerateOrderID 生成订单号
func GenerateOrderID(prefix string) (string, error) {
	timestamp := []byte(base64.URLEncoding.EncodeToString([]byte(prefix)))
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	orderID := append(timestamp, base64.URLEncoding.EncodeToString(randomBytes)...)
	if len(orderID) > 32 {
		return string(orderID[:32]), nil
	}
	return string(orderID), nil
}
