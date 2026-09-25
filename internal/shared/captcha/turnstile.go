package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type TurnstileValidator struct {
	secretKey string
	client    *http.Client
}

func NewTurnstileValidator(secretKey string) *TurnstileValidator {
	return &TurnstileValidator{
		secretKey: secretKey,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (v *TurnstileValidator) Verify(ctx context.Context, token string, remoteIP string) (bool, error) {
	if token == "" {
		return false, nil // Explicitly missing
	}

	form := url.Values{}
	form.Add("secret", v.secretKey)
	form.Add("response", token)
	if remoteIP != "" {
		form.Add("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://challenges.cloudflare.com/turnstile/v0/siteverify", strings.NewReader(form.Encode()))
	if err != nil {
		return false, fmt.Errorf("create captcha request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := v.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("captcha request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool     `json:"success"`
		Error   []string `json:"error-codes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("decode captcha response: %w", err)
	}

	return result.Success, nil
}
