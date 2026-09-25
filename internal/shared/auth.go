package shared

import (
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"strings"
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

// HIBPValidator implements PasswordValidator using the HaveIBeenPwned k-anonymity API.
type HIBPValidator struct {
	client *http.Client
}

func NewHIBPValidator() *HIBPValidator {
	return &HIBPValidator{
		client: &http.Client{Timeout: 3 * time.Second},
	}
}

func (v *HIBPValidator) IsCompromised(ctx context.Context, password string) (bool, error) {
	// Generate SHA-1 hash of the password
	h := sha1.New()
	h.Write([]byte(password))
	hashStr := fmt.Sprintf("%X", h.Sum(nil))

	// k-anonymity uses the first 5 characters
	prefix := hashStr[:5]
	suffix := hashStr[5:]

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.pwnedpasswords.com/range/"+prefix, nil)
	if err != nil {
		return false, fmt.Errorf("create HIBP request: %w", err)
	}
	req.Header.Set("User-Agent", "Dogfood-Platform")

	resp, err := v.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("do HIBP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("HIBP API returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("read HIBP response: %w", err)
	}

	// Response is a list of suffixes and counts
	body := string(bodyBytes)
	lines := strings.Split(body, "\r\n")
	if len(lines) == 1 {
		lines = strings.Split(body, "\n")
	}

	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if parts[0] == suffix {
			return true, nil // Password found in breach database
		}
	}

	return false, nil
}
