package shared

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type BcryptHasher struct{}

func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{}
}

func (h *BcryptHasher) Hash(plain string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(bytes), err
}

func (h *BcryptHasher) Verify(hash, plain string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	return err == nil
}

type JWTIssuer struct {
	secret             string
	accessTTLMinutes   int
	refreshTTLDays     int
}

func NewJWTIssuer(secret string, accessTTLMinutes, refreshTTLDays int) *JWTIssuer {
	return &JWTIssuer{
		secret:           secret,
		accessTTLMinutes: accessTTLMinutes,
		refreshTTLDays:   refreshTTLDays,
	}
}

func (j *JWTIssuer) IssueAccessToken(userID, email string, isAdmin bool) (string, error) {
	claims := jwt.MapClaims{
		"sub":     userID,
		"email":   email,
		"isAdmin": isAdmin,
		"exp":     time.Now().Add(time.Duration(j.accessTTLMinutes) * time.Minute).Unix(),
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secret))
}

func (j *JWTIssuer) IssueRefreshToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"exp": time.Now().Add(time.Duration(j.refreshTTLDays) * 24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secret))
}
