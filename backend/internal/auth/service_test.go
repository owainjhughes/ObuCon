package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func newTestService(secret string) *Service {
	// userRepo is left nil — crypto paths don't touch it.
	return &Service{jwtSecret: secret}
}

func TestGenerateAndValidateToken_RoundTrip(t *testing.T) {
	svc := newTestService("test-secret")

	token, err := svc.generateToken(42, "alice@example.com")
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	if token == "" {
		t.Fatal("generateToken returned empty string")
	}

	userID, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if userID != 42 {
		t.Errorf("userID: got %d, want 42", userID)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	signer := newTestService("secret-a")
	verifier := newTestService("secret-b")

	token, err := signer.generateToken(7, "x@y.com")
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}

	if _, err := verifier.ValidateToken(token); err == nil {
		t.Error("expected error when validating with different secret, got nil")
	}
}

func TestValidateToken_Malformed(t *testing.T) {
	svc := newTestService("test-secret")
	if _, err := svc.ValidateToken("not-a-jwt"); err == nil {
		t.Error("expected error for malformed token, got nil")
	}
	if _, err := svc.ValidateToken(""); err == nil {
		t.Error("expected error for empty token, got nil")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	secret := "test-secret"
	svc := newTestService(secret)

	// Manually build a JWT whose exp is in the past, signed with the matching secret.
	claims := jwt.MapClaims{
		"user_id": uint(1),
		"email":   "expired@example.com",
		"exp":     time.Now().Add(-1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}

	if _, err := svc.ValidateToken(signed); err == nil {
		t.Error("expected error for expired token, got nil")
	}
}

func TestBcryptHashAndCompare(t *testing.T) {
	password := "correct horse battery staple"

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword: %v", err)
	}

	if err := bcrypt.CompareHashAndPassword(hash, []byte(password)); err != nil {
		t.Errorf("expected match for same password, got: %v", err)
	}

	if err := bcrypt.CompareHashAndPassword(hash, []byte("wrong password")); err == nil {
		t.Error("expected mismatch for different password, got nil")
	}

	// Sanity: two hashes of the same password differ (bcrypt salts internally).
	hash2, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword (second): %v", err)
	}
	if strings.EqualFold(string(hash), string(hash2)) {
		t.Error("expected two bcrypt hashes of the same password to differ (salted)")
	}
}
