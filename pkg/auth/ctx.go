package auth

import (
	"context"
	"net/http"
)

type ctxKeyType string

const authenticationAccountObjectContextKey ctxKeyType = "authenticated-account-object"

// SetAuthenticationDetails sets user details for this request
func SetAuthenticationDetails(r *http.Request, u *User) *http.Request {
	ctx := context.WithValue(r.Context(), authenticationAccountObjectContextKey, u)
	return r.WithContext(ctx)
}

// GetAccountFromCtx - get current authenticated account info from ctx
func GetAccountFromCtx(ctx context.Context) *User {
	if u := ctx.Value(authenticationAccountObjectContextKey); u != nil {
		return u.(*User)
	}
	return nil
}
