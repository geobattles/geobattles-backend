package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"

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

		uid, err := auth.ValidateToken(token)
		if err != nil {
			api.ERROR(w, http.StatusUnauthorized, errors.New("Unauthorized"))
			return
		}
		ctx = context.WithValue(ctx, "uid", uid)

		next(w, r.WithContext(ctx))
	}
}
