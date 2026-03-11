package connectrpc

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"github.com/brady1408/atlinks/internal/auth"
)

// AuthInterceptor validates Bearer JWT tokens and populates auth.ContextUser.
func AuthInterceptor(jwt *auth.JWTService) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			token := req.Header().Get("Authorization")
			if token == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, nil)
			}
			token = strings.TrimPrefix(token, "Bearer ")
			if token == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, nil)
			}

			claims, err := jwt.ValidateToken(token)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, err)
			}

			// Non-super_admin must have a company assignment
			if claims.Role != "super_admin" && claims.CompanyID == 0 {
				return nil, connect.NewError(connect.CodePermissionDenied, nil)
			}

			ctx = auth.SetUser(ctx, auth.ContextUser{
				ID:        claims.UserID,
				Username:  claims.Username,
				Role:      claims.Role,
				CompanyID: claims.CompanyID,
			})

			return next(ctx, req)
		}
	}
}
