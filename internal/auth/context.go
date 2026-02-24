package auth

import (
	"context"
	"errors"
	"net/http"
)

// ErrNoUser is returned when an authenticated user is expected but not found in the context.
var ErrNoUser = errors.New("no authenticated user in context")

type contextKey string

const userContextKey contextKey = "user"

type ContextUser struct {
	ID        int
	Username  string
	Role      string
	CompanyID int
}

func SetUser(ctx context.Context, user ContextUser) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func GetUser(ctx context.Context) (ContextUser, bool) {
	user, ok := ctx.Value(userContextKey).(ContextUser)
	return user, ok
}

func GetUserFromRequest(r *http.Request) (ContextUser, bool) {
	return GetUser(r.Context())
}

func GetCompanyID(ctx context.Context) (int, error) {
	user, ok := GetUser(ctx)
	if !ok {
		return 0, ErrNoUser
	}
	return user.CompanyID, nil
}
