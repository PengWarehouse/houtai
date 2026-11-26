// Package auth 负责签发与校验 JWT（HS256）。
package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims 是 token 里携带的信息。
type Claims struct {
	UserID   int64  `json:"uid"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Manager 负责签发与解析 token。
type Manager struct {
	secret []byte
	ttl    time.Duration
	issuer string
}

// NewManager 创建 Manager，secret 过短会直接报错，避免弱密钥上线。
func NewManager(secret string, ttl time.Duration) (*Manager, error) {
	if len(secret) < 16 {
		return nil, fmt.Errorf("JWT_SECRET 至少需要 16 个字符，当前长度 %d", len(secret))
	}
	if ttl <= 0 {
		ttl = 12 * time.Hour
	}
	return &Manager{
		secret: []byte(secret),
		ttl:    ttl,
		issuer: "goProject",
	}, nil
}

// TTL 返回 token 有效期。
func (m *Manager) TTL() time.Duration {
	return m.ttl
}

// Issue 为用户签发 token。
func (m *Manager) Issue(userID int64, username, role string) (string, error) {
	now := time.Now()

	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("签发 token 失败: %w", err)
	}
	return signed, nil
}

// Parse 校验 token 并返回 claims。
func (m *Manager) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(t *jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(m.issuer),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("token 无效")
	}

	return claims, nil
}

// RandomSecret 生成 32 字节随机密钥（64 位十六进制），用于未配置 JWT_SECRET 时的临时密钥。
func RandomSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成随机密钥失败: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
