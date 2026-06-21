package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/subbu/family_tree/handlers/response"
	"github.com/subbu/family_tree/models"
	"github.com/subbu/family_tree/services/auth"
)

type contextKey string

const userContextKey contextKey = "user"

func WithUser(authService auth.Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			response.Error(w, http.StatusUnauthorized, "missing bearer token")
			return
		}

		token := strings.TrimPrefix(header, "Bearer ")
		user, err := authService.ValidateToken(r.Context(), token)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserFromContext(ctx context.Context) (models.User, bool) {
	user, ok := ctx.Value(userContextKey).(models.User)
	return user, ok
}