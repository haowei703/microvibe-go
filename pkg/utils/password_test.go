package utils_test

import (
	"microvibe-go/pkg/utils"
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	hash, err := utils.HashPassword("mypassword123")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == "mypassword123" {
		t.Fatal("hash should not equal plaintext password")
	}
}

func TestCheckPassword_Correct(t *testing.T) {
	hash, err := utils.HashPassword("correct_password")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if !utils.CheckPassword("correct_password", hash) {
		t.Fatal("CheckPassword should return true for correct password")
	}
}

func TestCheckPassword_Wrong(t *testing.T) {
	hash, err := utils.HashPassword("real_password")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if utils.CheckPassword("wrong_password", hash) {
		t.Fatal("CheckPassword should return false for wrong password")
	}
}

func TestCheckPassword_EmptyHash(t *testing.T) {
	if utils.CheckPassword("anything", "") {
		t.Fatal("CheckPassword should return false for empty hash")
	}
}

func TestCheckPassword_EmptyPassword(t *testing.T) {
	hash, err := utils.HashPassword("")
	if err != nil {
		t.Fatalf("HashPassword failed for empty string: %v", err)
	}

	if utils.CheckPassword("", hash) {
		t.Log("empty password correctly matches empty password hash")
	}

	if utils.CheckPassword("something", hash) {
		t.Fatal("non-empty password should not match empty password hash")
	}
}

func TestCheckPassword_CaseSensitive(t *testing.T) {
	hash, err := utils.HashPassword("MyPassword")
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if utils.CheckPassword("mypassword", hash) {
		t.Fatal("CheckPassword should be case-sensitive")
	}
}

func TestHashPassword_LongInput(t *testing.T) {
	longPass := strings.Repeat("a", 72)
	hash, err := utils.HashPassword(longPass)
	if err != nil {
		t.Fatalf("HashPassword failed for 72-char password: %v", err)
	}
	if !utils.CheckPassword(longPass, hash) {
		t.Fatal("CheckPassword should work for 72-char passwords")
	}
}

func TestHashPassword_BcryptLimit(t *testing.T) {
	longPass := strings.Repeat("b", 100)
	_, err := utils.HashPassword(longPass)
	if err != nil {
		t.Logf("bcrypt rejects passwords > 72 bytes (expected): %v", err)
	} else {
		t.Log("bcrypt accepted 100-char password (implementation specific)")
	}
}

func TestGenerateRandomPassword_Length(t *testing.T) {
	lengths := []int{8, 16, 32, 64}
	for _, l := range lengths {
		pw := utils.GenerateRandomPassword(l)
		if len(pw) != l {
			t.Errorf("expected length %d, got %d", l, len(pw))
		}
	}
}

func TestGenerateRandomPassword_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		pw := utils.GenerateRandomPassword(16)
		if seen[pw] {
			t.Fatal("GenerateRandomPassword produced duplicate password")
		}
		seen[pw] = true
	}
}

func TestGenerateRandomPassword_NotEmpty(t *testing.T) {
	pw := utils.GenerateRandomPassword(16)
	if pw == "" {
		t.Fatal("expected non-empty password")
	}
}

func TestHashPassword_SpecialChars(t *testing.T) {
	special := "!@#$%^&*()_+-=[]{}|;':\",./<>?`~"
	hash, err := utils.HashPassword(special)
	if err != nil {
		t.Fatalf("HashPassword failed for special chars: %v", err)
	}
	if !utils.CheckPassword(special, hash) {
		t.Fatal("CheckPassword should work with special characters")
	}
}

func TestHashPassword_Unicode(t *testing.T) {
	unicode := "密码测试パスワード🔐"
	hash, err := utils.HashPassword(unicode)
	if err != nil {
		t.Fatalf("HashPassword failed for unicode: %v", err)
	}
	if !utils.CheckPassword(unicode, hash) {
		t.Fatal("CheckPassword should work with unicode characters")
	}
}

func BenchmarkHashPassword(b *testing.B) {
	for b.Loop() {
		_, _ = utils.HashPassword("benchmark_password")
	}
}

func BenchmarkCheckPassword(b *testing.B) {
	hash, _ := utils.HashPassword("benchmark_password")
	b.ResetTimer()
	for b.Loop() {
		utils.CheckPassword("benchmark_password", hash)
	}
}
