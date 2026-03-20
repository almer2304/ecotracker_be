package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims adalah payload JWT
type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWTManager mengelola pembuatan dan validasi token
type JWTManager struct {
	secret        []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewJWTManager(secret string, accessExpiry, refreshExpiry time.Duration) *JWTManager {
	return &JWTManager{
		secret:        []byte(secret),
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

// GenerateAccessToken membuat access token JWT
func (m *JWTManager) GenerateAccessToken(userID, email, role string) (string, time.Time, error) {
	expiresAt := time.Now().Add(m.accessExpiry)
	claims := &JWTClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "ecotracker",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("gagal sign token: %w", err)
	}

	return signed, expiresAt, nil
}

// GenerateRefreshToken membuat refresh token JWT
func (m *JWTManager) GenerateRefreshToken(userID string) (string, time.Time, error) {
	expiresAt := time.Now().Add(m.refreshExpiry)
	claims := &JWTClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "ecotracker-refresh",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("gagal sign refresh token: %w", err)
	}

	return signed, expiresAt, nil
}

// ValidateToken memvalidasi dan mengembalikan claims dari token
func (m *JWTManager) ValidateToken(tokenStr string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("signing method tidak diharapkan: %v", t.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token tidak valid: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("token claims tidak valid")
	}

	return claims, nil
}

// AccessTokenExpirySeconds mengembalikan durasi access token dalam detik
func (m *JWTManager) AccessTokenExpirySeconds() int64 {
	return int64(m.accessExpiry.Seconds())
}
