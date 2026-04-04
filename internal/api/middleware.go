package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// authMiddleware enforces JWT auth on all routes except POST /auth/login.
// If no password_hash is configured, auth is disabled.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Auth is optional — disabled when no password_hash is set.
		if s.cfg.Auth.PasswordHash == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Login endpoint is always public.
		if r.Method == http.MethodPost && r.URL.Path == "/auth/login" {
			next.ServeHTTP(w, r)
			return
		}

		token := extractBearer(r)
		if token == "" {
			// SSE/WS clients pass token as query param since EventSource can't set headers.
			token = r.URL.Query().Get("token")
		}
		if token == "" {
			jsonError(w, "authentication required", http.StatusUnauthorized)
			return
		}

		if err := s.validateJWT(token); err != nil {
			jsonError(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) validateJWT(tokenStr string) error {
	_, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(s.cfg.Auth.JWTSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	return err
}

func (s *Server) issueJWT() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "pdarr",
		"exp": time.Now().Add(30 * 24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})
	return token.SignedString([]byte(s.cfg.Auth.JWTSecret))
}

func extractBearer(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(auth, "Bearer "); ok {
		return after
	}
	return ""
}

// requestLogger logs method, path, and duration for every request.
func (s *Server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		s.log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
