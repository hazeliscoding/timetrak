// Package auth provides email/password authentication, session creation,
// and the shared Argon2id hashing helpers used by the signup/login flows
// and by the dev-seed command.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters: m=64MiB, t=3, p=2, 16-byte salt, 32-byte hash.
// These defaults follow OWASP 2023 guidance and match the placeholder used in dev-seed.
const (
	argonMemoryKiB = 64 * 1024
	argonTime      = 3
	argonThreads   = 2
	argonSaltLen   = 16
	argonKeyLen    = 32

	// MinPasswordLen is the documented minimum password length.
	MinPasswordLen = 10
)

// ErrInvalidPasswordHash indicates a stored hash does not match the expected encoding.
var ErrInvalidPasswordHash = errors.New("auth: invalid password hash encoding")

// ErrWeakPassword indicates the supplied password does not meet the documented policy.
var ErrWeakPassword = errors.New("auth: password does not meet minimum requirements")

// ValidatePassword enforces the MVP password policy: length >= MinPasswordLen.
// Stronger rules (character classes, breach lists) are future work.
func ValidatePassword(pw string) error {
	if len(pw) < MinPasswordLen {
		return ErrWeakPassword
	}
	return nil
}

// HashPassword returns an encoded Argon2id hash string.
// Format: $argon2id$v=19$m=<kib>,t=<iters>,p=<threads>$<salt_b64>$<hash_b64>
func HashPassword(pw string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(pw), salt, argonTime, argonMemoryKiB, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemoryKiB, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// VerifyPassword returns nil if pw matches the encoded Argon2id hash.
// It is constant-time in the compare step.
func VerifyPassword(pw, encoded string) error {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return ErrInvalidPasswordHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return ErrInvalidPasswordHash
	}
	var mem uint32
	var iter uint32
	var par uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &iter, &par); err != nil {
		return ErrInvalidPasswordHash
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return ErrInvalidPasswordHash
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return ErrInvalidPasswordHash
	}
	got := argon2.IDKey([]byte(pw), salt, iter, mem, par, uint32(len(want)))
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return errors.New("auth: password mismatch")
	}
	return nil
}
