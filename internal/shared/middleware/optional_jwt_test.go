package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type optionalJWTTestCache struct{}

func (optionalJWTTestCache) Get(context.Context, string) ([]byte, error) { return nil, nil }
func (optionalJWTTestCache) Set(context.Context, string, []byte, time.Duration) error {
	return nil
}
func (optionalJWTTestCache) Delete(context.Context, string) error { return nil }

func TestOptionalJWTMiddleware_AllowsAnonymousRequest(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	OptionalJWTMiddleware("secret", nil)(next).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/events", nil))

	if !called || recorder.Code != http.StatusNoContent {
		t.Fatalf("anonymous request status = %d, handler called = %t", recorder.Code, called)
	}
}

func TestOptionalJWTMiddleware_RejectsMalformedBearerToken(t *testing.T) {
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler should not run for a malformed bearer token")
	})
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	req.Header.Set("Authorization", "Bearer malformed")
	recorder := httptest.NewRecorder()

	OptionalJWTMiddleware("secret", nil)(next).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestOptionalJWTMiddleware_AddsUserContextForValidToken(t *testing.T) {
	const userID = "a1f74f4c-7d3c-477f-b9b1-4d2a4f62e994"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,
		"jti": "test-token-id",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	var gotUserID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = GetUserID(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/events/example", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	recorder := httptest.NewRecorder()

	OptionalJWTMiddleware("test-secret", optionalJWTTestCache{})(next).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent || gotUserID != userID {
		t.Fatalf("status = %d, userID = %q", recorder.Code, gotUserID)
	}
}
