// Package http contains shared HTTP middleware and HTMX helpers.
package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

type ctxKey int

const (
	ctxKeyRequestID ctxKey = iota
	ctxKeyLogger
)

// RequestID middleware assigns an X-Request-Id to every request (generating one if absent)
// and stores it in the context.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-Id")
		if rid == "" {
			var b [8]byte
			_, _ = rand.Read(b[:])
			rid = hex.EncodeToString(b[:])
		}
		w.Header().Set("X-Request-Id", rid)
		ctx := context.WithValue(r.Context(), ctxKeyRequestID, rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFromContext returns the request ID stashed by the RequestID middleware.
func RequestIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyRequestID).(string)
	return v
}

// Logging middleware emits a single structured log entry per request with latency
// and status. It must run after RequestID.
func Logging(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			lw := &loggingWriter{ResponseWriter: w, status: 200}
			rid := RequestIDFromContext(r.Context())
			logger := base.With("request_id", rid)
			ctx := context.WithValue(r.Context(), ctxKeyLogger, logger)
			next.ServeHTTP(lw, r.WithContext(ctx))
			logger.Info("http_request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", lw.status,
				"bytes", lw.bytes,
				"duration_ms", time.Since(start).Milliseconds(),
				"htmx", r.Header.Get("HX-Request") == "true",
			)
		})
	}
}

// LoggerFromContext returns the request-scoped logger (falls back to slog default).
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKeyLogger).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// Recover middleware turns panics into 500s with a logged stack trace.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				LoggerFromContext(r.Context()).Error("panic",
					"error", rec,
					"stack", string(debug.Stack()),
				)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type loggingWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *loggingWriter) WriteHeader(code int) { w.status = code; w.ResponseWriter.WriteHeader(code) }
func (w *loggingWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// IsHTMX reports whether the request originated from an HTMX call.
func IsHTMX(r *http.Request) bool { return r.Header.Get("HX-Request") == "true" }

// TriggerEvent instructs HTMX to fire the named client-side event after the swap.
// Multiple events can be set by calling this repeatedly with comma-separated names.
func TriggerEvent(w http.ResponseWriter, events ...string) {
	if len(events) == 0 {
		return
	}
	existing := w.Header().Get("HX-Trigger")
	for _, e := range events {
		if existing == "" {
			existing = e
		} else {
			existing += "," + e
		}
	}
	w.Header().Set("HX-Trigger", existing)
}

// Redirect instructs HTMX (when present) to navigate; falls back to a 303 for plain requests.
func Redirect(w http.ResponseWriter, r *http.Request, to string) {
	if IsHTMX(r) {
		w.Header().Set("HX-Redirect", to)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, to, http.StatusSeeOther)
}
