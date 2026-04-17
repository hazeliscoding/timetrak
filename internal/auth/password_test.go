package auth

import (
	"strings"
	"testing"
)

func TestHashAndVerifyPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Fatalf("unexpected hash format: %q", hash)
	}
	if err := VerifyPassword("correct-horse-battery-staple", hash); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := VerifyPassword("wrong-password-haha", hash); err == nil {
		t.Fatalf("expected wrong password to fail")
	}
}

func TestValidatePassword(t *testing.T) {
	if err := ValidatePassword("short"); err == nil {
		t.Fatalf("expected rejection for short password")
	}
	if err := ValidatePassword("1234567890"); err != nil {
		t.Fatalf("expected 10-char password to pass: %v", err)
	}
}

func TestVerifyRejectsMalformedHash(t *testing.T) {
	if err := VerifyPassword("any", "not-a-valid-hash"); err == nil {
		t.Fatalf("expected malformed hash to be rejected")
	}
}
