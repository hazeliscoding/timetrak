// Package session implements PostgreSQL-backed server-side sessions with
// signed, HttpOnly, SameSite=Lax cookies carrying only the session ID.
package session

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	cookieName = "tt_session"
	defaultTTL = 30 * 24 * time.Hour
	cookiePath = "/"
)

// ErrNotFound is returned when a session id is not present or expired.
var ErrNotFound = errors.New("session: not found or expired")

// Session is what a handler typically needs to know about the current request.
type Session struct {
	ID                uuid.UUID
	UserID            uuid.UUID
	ActiveWorkspaceID *uuid.UUID
	ExpiresAt         time.Time
}

// Store persists sessions in PostgreSQL.
type Store struct {
	pool   *pgxpool.Pool
	secret []byte
	secure bool
}

// NewStore returns a Store. `secret` must be at least 32 bytes. `secure` controls the Secure cookie flag.
func NewStore(pool *pgxpool.Pool, secret []byte, secure bool) (*Store, error) {
	if len(secret) < 32 {
		return nil, errors.New("session: secret must be at least 32 bytes")
	}
	return &Store{pool: pool, secret: secret, secure: secure}, nil
}

// Create inserts a new session for the given user and writes the signed cookie.
func (s *Store) Create(ctx context.Context, w http.ResponseWriter, userID uuid.UUID) (Session, error) {
	id := uuid.New()
	expires := time.Now().UTC().Add(defaultTTL)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (id, user_id, expires_at, created_at)
		VALUES ($1, $2, $3, now())
	`, id, userID, expires)
	if err != nil {
		return Session{}, err
	}
	s.writeCookie(w, id, expires)
	return Session{ID: id, UserID: userID, ExpiresAt: expires}, nil
}

// Load returns the session referenced by the request cookie, or ErrNotFound.
func (s *Store) Load(ctx context.Context, r *http.Request) (Session, error) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return Session{}, ErrNotFound
	}
	id, ok := s.verify(c.Value)
	if !ok {
		return Session{}, ErrNotFound
	}
	var sess Session
	var activeWs *uuid.UUID
	err = s.pool.QueryRow(ctx, `
		SELECT id, user_id, active_workspace_id, expires_at
		FROM sessions
		WHERE id = $1 AND expires_at > now()
	`, id).Scan(&sess.ID, &sess.UserID, &activeWs, &sess.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, err
	}
	sess.ActiveWorkspaceID = activeWs
	return sess, nil
}

// SetActiveWorkspace updates the active_workspace_id on the stored session row.
// Callers must have verified membership before calling.
func (s *Store) SetActiveWorkspace(ctx context.Context, id, workspaceID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE sessions SET active_workspace_id = $1 WHERE id = $2`, workspaceID, id)
	return err
}

// Destroy removes the session row and clears the cookie.
func (s *Store) Destroy(ctx context.Context, w http.ResponseWriter, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     cookiePath,
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
	return nil
}

// CookieName returns the name used for the session cookie.
func (s *Store) CookieName() string { return cookieName }

// writeCookie writes the signed session cookie to the response.
func (s *Store) writeCookie(w http.ResponseWriter, id uuid.UUID, expires time.Time) {
	signed := s.sign(id)
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    signed,
		Path:     cookiePath,
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
	})
}

// sign encodes an HMAC-authenticated session id as "<id_hex>.<mac_b64>".
func (s *Store) sign(id uuid.UUID) string {
	raw := id.String()
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(raw))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return raw + "." + sig
}

// verify checks the HMAC and returns the underlying uuid.
func (s *Store) verify(value string) (uuid.UUID, bool) {
	dot := -1
	for i := 0; i < len(value); i++ {
		if value[i] == '.' {
			dot = i
			break
		}
	}
	if dot < 0 {
		return uuid.Nil, false
	}
	raw := value[:dot]
	sig := value[dot+1:]
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(raw))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

// RandomSecret returns a cryptographically random secret suitable for NewStore.
// Useful for bootstrapping dev environments.
func RandomSecret(nBytes int) ([]byte, error) {
	if nBytes < 32 {
		nBytes = 32
	}
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}
