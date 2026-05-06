package middleware

import (
	"context"

	"order-playground/apps/gateway-api/internal/domain"
)

type contextKey string

const userContextKey contextKey = "user"

func WithUser(ctx context.Context, user domain.UserClaims) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) (domain.UserClaims, bool) {
	user, ok := ctx.Value(userContextKey).(domain.UserClaims)
	return user, ok
}
