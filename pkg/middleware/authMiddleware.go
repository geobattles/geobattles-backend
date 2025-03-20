package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/geobattles/geobattles-backend/pkg/api"
	"github.com/geobattles/geobattles-backend/pkg/auth"
)

// authorization middleware. handles cors and validates jwt token
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			return
		}

		slog.Debug("Auth middleware", "header", r.Header)

		ctx := r.Context()
		token := r.Header.Get("Authorization")

		claims, err := auth.ValidateToken(token)
		if err != nil {
			api.ERROR(w, http.StatusUnauthorized, errors.New("Unauthorized"))
			return
		}
		// ctx = context.WithValue(ctx, uidKey, claims.UID)
		// ctx = context.WithValue(ctx, displayNameKey, claims.DisplayName)

		ctx = context.WithValue(ctx, api.UidKey, claims.UID)
		ctx = context.WithValue(ctx, api.DisplayNameKey, claims.DisplayName)

		next(w, r.WithContext(ctx))
	}
}

// authorization middleware. handles cors and validates jwt token
func SocketAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			return
		}

		secWebSocketProtocol := r.Header.Get("Sec-WebSocket-Protocol")

		// Extract the JWT token from the Sec-WebSocket-Protocol header
		protocols := strings.Split(secWebSocketProtocol, ",")
		if len(protocols) < 2 {
			api.ERROR(w, http.StatusUnauthorized, errors.New("Unauthorized"))
			return
		}
		token := strings.TrimSpace(protocols[1])

		slog.Debug("Socket auth middleware", "TOKEN", token)

		ctx := r.Context()

		claims, err := auth.ValidateToken(token)
		if err != nil {
			api.ERROR(w, http.StatusUnauthorized, errors.New("Unauthorized"))
			return
		}
		// ctx = context.WithValue(ctx, uidKey, claims.UID)
		// ctx = context.WithValue(ctx, displayNameKey, claims.DisplayName)

		ctx = context.WithValue(ctx, api.UidKey, claims.UID)
		ctx = context.WithValue(ctx, api.DisplayNameKey, claims.DisplayName)

		next(w, r.WithContext(ctx))
	}
}
