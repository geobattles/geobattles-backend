package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/slinarji/go-geo-server/pkg/api"
	"github.com/slinarji/go-geo-server/pkg/auth"
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

		fmt.Println(r.Header)

		ctx := r.Context()
		token := r.Header.Get("Authorization")

		claims, err := auth.ValidateToken(token)
		if err != nil {
			api.ERROR(w, http.StatusUnauthorized, errors.New("Unauthorized"))
			return
		}
		ctx = context.WithValue(ctx, "uid", claims.UID)
		ctx = context.WithValue(ctx, "displayname", claims.DisplayName)

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

		slog.Info("Socket auth middleware", "TOKEN", token)

		ctx := r.Context()

		claims, err := auth.ValidateToken(token)
		if err != nil {
			api.ERROR(w, http.StatusUnauthorized, errors.New("Unauthorized"))
			return
		}
		ctx = context.WithValue(ctx, "uid", claims.UID)
		ctx = context.WithValue(ctx, "displayname", claims.DisplayName)

		next(w, r.WithContext(ctx))
	}
}
