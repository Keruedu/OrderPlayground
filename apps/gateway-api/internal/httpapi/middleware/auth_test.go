package middleware

import (
	"context"
	"testing"

	"order-playground/apps/gateway-api/internal/domain"
)

func TestWithUserRoundTrip(t *testing.T) {
	user := domain.UserClaims{Username: "user1", Roles: []string{"user"}}
	ctx := WithUser(context.Background(), user)
	got, ok := UserFromContext(ctx)
	if !ok {
		t.Fatalf("expected user in context")
	}
	if got.Username != "user1" {
		t.Fatalf("unexpected username %s", got.Username)
	}
}
