package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey string

// ContextKeyUserID — ключ для хранения UUID пользователя в контексте запроса.
const ContextKeyUserID contextKey = "user_id"

// JWTAuth возвращает middleware, проверяющий Bearer JWT-токен в заголовке Authorization.
func JWTAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			userID, _ := claims["sub"].(string)
			ctx := context.WithValue(r.Context(), ContextKeyUserID, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext возвращает UUID пользователя после прохождения JWTAuth.
func UserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	raw := ctx.Value(ContextKeyUserID)
	if raw == nil {
		return uuid.Nil, errors.New("middleware: no user in context")
	}
	s, ok := raw.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return uuid.Nil, errors.New("middleware: invalid user id in context")
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}
