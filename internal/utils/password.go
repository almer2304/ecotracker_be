package utils

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword mengenkripsi password dengan bcrypt
func HashPassword(password string, cost int) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("gagal hash password: %w", err)
	}
	return string(hashed), nil
}

// CheckPassword memverifikasi password terhadap hash yang tersimpan
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
