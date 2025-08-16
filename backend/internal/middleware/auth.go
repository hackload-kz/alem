package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"hackload/internal/service"
)

type contextKey string

const (
	UserContextKey contextKey = "user"
)

func AuthenticationMiddleware(authService service.AuthenticationService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if !strings.HasPrefix(authHeader, "Basic ") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(authHeader, "Basic ")
			if token == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			session, err := authService.GetSession(r.Context(), token)
			if err != nil {
				if err == service.ErrUnauthorized {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				fmt.Println("ERROR: authService.GetSession:", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserFromContext(ctx context.Context) (*service.GetSessionResponse, bool) {
	user, ok := ctx.Value(UserContextKey).(*service.GetSessionResponse)
	return user, ok
}
