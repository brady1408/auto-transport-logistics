package auth

import (
	"context"
	"log"
	"net/http"
)

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

func GetCompanyID(ctx context.Context) int {
	user, ok := GetUser(ctx)
	if !ok {
		log.Println("ERROR: GetCompanyID called without authenticated user in context")
		return 0
	}
	return user.CompanyID
}
