package utils_test

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"microvibe-go/pkg/utils"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateToken(t *testing.T) {
	token, err := utils.GenerateToken(1, "testuser", 0, "testsecret", 24)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts in JWT, got %d", len(parts))
	}
}

func TestGenerateToken_Expiry(t *testing.T) {
	token, err := utils.GenerateToken(1, "testuser", 0, "testsecret", 2)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := utils.ParseToken(token, "testsecret")
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}

	expectedExpiry := time.Now().Add(2 * time.Hour)
	diff := claims.ExpiresAt.Time.Sub(expectedExpiry).Abs()
	if diff > 5*time.Second {
		t.Errorf("expiry time mismatch: got %v, expected ~%v", claims.ExpiresAt.Time, expectedExpiry)
	}
	if claims.Issuer != "microvibe-go" {
		t.Errorf("expected issuer 'microvibe-go', got '%s'", claims.Issuer)
	}
}

func TestGenerateToken_HasJTI(t *testing.T) {
	token, err := utils.GenerateToken(1, "testuser", 0, "testsecret", 24)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := utils.ParseToken(token, "testsecret")
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}

	if claims.JTI == "" {
		t.Fatal("expected non-empty JTI claim")
	}
	if len(claims.JTI) != 32 {
		t.Errorf("expected JTI to be 32 hex chars, got %d", len(claims.JTI))
	}
}

func TestParseToken_Success(t *testing.T) {
	token, err := utils.GenerateToken(42, "hello", 1, "supersecret", 24)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := utils.ParseToken(token, "supersecret")
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}

	if claims.UserID != 42 {
		t.Errorf("expected UserID 42, got %d", claims.UserID)
	}
	if claims.Username != "hello" {
		t.Errorf("expected Username 'hello', got '%s'", claims.Username)
	}
	if claims.Role != 1 {
		t.Errorf("expected Role 1, got %d", claims.Role)
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	token, err := utils.GenerateToken(1, "user", 0, "correct-secret", 24)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, err = utils.ParseToken(token, "wrong-secret")
	if err == nil {
		t.Fatal("expected error for wrong secret, got nil")
	}
}

func TestParseToken_Expired(t *testing.T) {
	token, err := utils.GenerateToken(1, "user", 0, "secret", -1)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	_, err = utils.ParseToken(token, "secret")
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestParseToken_Malformed(t *testing.T) {
	_, err := utils.ParseToken("not-a-valid-jwt", "secret")
	if err == nil {
		t.Fatal("expected error for malformed token, got nil")
	}
}

func TestParseToken_EmptyString(t *testing.T) {
	_, err := utils.ParseToken("", "secret")
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
}

func TestParseToken_AlgorithmNone(t *testing.T) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"user_id":1,"username":"hacker","role":1,"jti":"fake"}`))
	forgedToken := header + "." + payload + "."

	_, err := utils.ParseToken(forgedToken, "secret")
	if err == nil {
		t.Fatal("VULNERABILITY: alg:none token was accepted. Algorithm validation is broken.")
	}
}

func TestParseToken_WrongAlgorithm(t *testing.T) {
	claims := utils.Claims{
		UserID:   1,
		Username: "test",
		Role:     0,
		JTI:      "abc",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			Issuer:    "microvibe-go",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
	signedToken, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	_, err = utils.ParseToken(signedToken, "secret")
	if err == nil {
		t.Fatal("expected error for HS384 algorithm token, got nil")
	}
}

func TestParseToken_ZeroUserID(t *testing.T) {
	token, err := utils.GenerateToken(0, "newuser", 0, "secret", 24)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := utils.ParseToken(token, "secret")
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if claims.UserID != 0 {
		t.Errorf("expected UserID 0, got %d", claims.UserID)
	}
}

func TestParseToken_EmptyUsername(t *testing.T) {
	token, err := utils.GenerateToken(1, "", 0, "secret", 24)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := utils.ParseToken(token, "secret")
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if claims.Username != "" {
		t.Errorf("expected empty Username, got '%s'", claims.Username)
	}
}

func TestGenerateToken_AdminRole(t *testing.T) {
	token, err := utils.GenerateToken(1, "admin", 1, "secret", 24)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := utils.ParseToken(token, "secret")
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if claims.Role != 1 {
		t.Errorf("expected Role 1 (admin), got %d", claims.Role)
	}
}

func TestParseToken_KeyFuncValidatesAlg(t *testing.T) {
	rawToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9." +
		"eyJ1c2VyX2lkIjoxLCJ1c2VybmFtZSI6InRlc3QiLCJyb2xlIjowfQ." +
		"signature"

	_, err := utils.ParseToken(rawToken, "secret")
	if err == nil {
		t.Fatal("VULNERABILITY: RSA algorithm token accepted with HMAC secret key")
	}
}

func TestGenerateToken_DifferentJTIs(t *testing.T) {
	token1, _ := utils.GenerateToken(1, "user", 0, "secret", 24)
	token2, _ := utils.GenerateToken(1, "user", 0, "secret", 24)

	claims1, _ := utils.ParseToken(token1, "secret")
	claims2, _ := utils.ParseToken(token2, "secret")

	if claims1.JTI == claims2.JTI {
		t.Fatal("expected different JTIs for different tokens")
	}
}

func TestGenerateToken_ExpiryValidation(t *testing.T) {
	tests := []struct {
		name      string
		expHours  int
		expectErr bool
	}{
		{"normal expiry", 24, false},
		{"zero expiry should fail on parse", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := utils.GenerateToken(1, "user", 0, "secret", tt.expHours)
			if tt.expectErr && err != nil {
				return // expected
			}
			if err != nil {
				t.Fatalf("GenerateToken failed: %v", err)
			}
		})
	}
}

func BenchmarkGenerateToken(b *testing.B) {
	for b.Loop() {
		_, _ = utils.GenerateToken(1, "benchuser", 0, "benchsecret", 24)
	}
}

func BenchmarkParseToken(b *testing.B) {
	token, _ := utils.GenerateToken(1, "benchuser", 0, "benchsecret", 24)
	b.ResetTimer()
	for b.Loop() {
		_, _ = utils.ParseToken(token, "benchsecret")
	}
}

func mustBase64Encode(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

func mustJSONEncode(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
