package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"hackload/internal/service"
	"hackload/pkg/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type contextKey string

const (
	UserContextKey contextKey = "user"
)

func AuthenticationMiddleware(authService service.AuthenticationService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := telemetry.Global().T().Start(r.Context(), "AuthenticationMiddleware")
			defer span.End()

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

			span.AddEvent("checking authorization header", trace.WithAttributes(
				attribute.String("method", r.Method),
				attribute.String("url", r.URL.String()),
				attribute.String("token", token),
			))

			session, err := authService.GetSession(ctx, token)
			if err != nil {
				if err == service.ErrUnauthorized {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				fmt.Println("ERROR: authService.GetSession:", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			ctx = context.WithValue(ctx, UserContextKey, session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserFromContext(ctx context.Context) (*service.GetSessionResponse, bool) {
	user, ok := ctx.Value(UserContextKey).(*service.GetSessionResponse)
	return user, ok
}
