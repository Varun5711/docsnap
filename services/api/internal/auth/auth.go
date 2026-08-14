package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const SessionDuration = 30 * 24 * time.Hour

var (
	ErrInvalidEmail    = errors.New("invalid email")
	ErrWeakPassword    = errors.New("password must be at least 8 characters")
	ErrInvalidCreds    = errors.New("invalid email or password")
	ErrEmailTaken      = errors.New("an account with this email already exists")
	ErrSessionNotFound = errors.New("session not found or expired")
)

func ValidateEmail(email string) error {
	if _, err := mail.ParseAddress(email); err != nil {
		return ErrInvalidEmail
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrWeakPassword
	}
	return nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func NewSessionToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return "tls_" + hex.EncodeToString(b)
}

func ConstantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
