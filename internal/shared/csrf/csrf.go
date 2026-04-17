// Package csrf implements a signed double-submit CSRF token.
//
// On every request, a CSRF cookie is ensured. Mutating requests
// (POST/PUT/PATCH/DELETE) must include the token as form field
// `csrf_token` or header `X-CSRF-Token`; a missing or mismatched token
// yields HTTP 403.
package csrf

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"time"
)

const (
	cookieName = "tt_csrf"
	headerName = "X-CSRF-Token"
	formField  = "csrf_token"
	ttl        = 12 * time.Hour
)

type ctxKey int

const ctxKeyToken ctxKey = 0

// Middleware ensures a CSRF cookie exists and validates mutating requests.
// `secret` must be at least 32 bytes; `secure` mirrors the session cookie.
func Middleware(secret []byte, secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := ensureCookie(w, r, secret, secure)
			if isMutating(r.Method) {
				supplied := r.Header.Get(headerName)
				if supplied == "" {
					_ = r.ParseForm()
					supplied = r.PostFormValue(formField)
				}
				if !validPair(secret, token, supplied) {
					http.Error(w, "forbidden: invalid CSRF token", http.StatusForbidden)
					return
				}
			}
			ctx := context.WithValue(r.Context(), ctxKeyToken, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Token returns the per-request token string (safe to render into forms).
func Token(r *http.Request) string { return TokenFromContext(r.Context()) }

// TokenFromContext fetches the token stored by Middleware, if any.
func TokenFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyToken).(string)
	return v
}

func isMutating(m string) bool {
	switch m {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

// ensureCookie returns an existing valid token if present, otherwise issues a new one.
func ensureCookie(w http.ResponseWriter, r *http.Request, secret []byte, secure bool) string {
	if c, err := r.Cookie(cookieName); err == nil && validSelf(secret, c.Value) {
		return c.Value
	}
	tok := mint(secret)
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    tok,
		Path:     "/",
		HttpOnly: false, // Readable so forms can echo it server-side (we render, browser submits).
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(ttl),
	})
	return tok
}

func mint(secret []byte) string {
	var b [32]byte
	_, _ = rand.Read(b[:])
	raw := base64.RawURLEncoding.EncodeToString(b[:])
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(raw))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return raw + "." + sig
}

func validSelf(secret []byte, token string) bool {
	raw, sig, ok := split(token)
	if !ok {
		return false
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(raw))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expected))
}

func validPair(secret []byte, cookieToken, supplied string) bool {
	if supplied == "" || cookieToken == "" {
		return false
	}
	if !hmac.Equal([]byte(cookieToken), []byte(supplied)) {
		return false
	}
	return validSelf(secret, cookieToken)
}

func split(token string) (string, string, bool) {
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			return token[:i], token[i+1:], true
		}
	}
	return "", "", false
}
