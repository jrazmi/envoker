// Package mid provides app level middleware support.
package mid

import (
	"context"
	"errors"

	"github.com/jrazmi/envoker/infrastructure/web"
)

// isError tests if the Encoder has an error inside of it.
func isError(e web.Encoder) error {
	err, isError := e.(error)
	if isError {
		return err
	}
	return nil
}

// =============================================================================

type ctxKey int

const (
	claimKey ctxKey = iota + 1
	userIDKey
)

// func setClaims(ctx context.Context, claims auths.AuthUser) context.Context {
// 	return context.WithValue(ctx, claimKey, claims)
// }

// // GetClaims returns the claims from the context.
// func GetClaims(ctx context.Context) (auths.AuthUser, error) {
// 	v, ok := ctx.Value(claimKey).(auths.AuthUser)
// 	if !ok {
// 		return auths.AuthUser{}, errors.New("no user claims")
// 	}
// 	return v, nil
// }

func setUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserID returns the user id from the context.
func GetUserID(ctx context.Context) (string, error) {
	v, ok := ctx.Value(userIDKey).(string)
	if !ok {
		return "", errors.New("user id not found in context")
	}

	return v, nil
}

// func setTeamPermission(ctx context.Context, permission permissionrepository.Permission) context.Context {
// 	return context.WithValue(ctx, permissionKey, permission)
// }

// func GetTeamPermission(ctx context.Context) (permissionrepository.Permission, error) {
// 	v, ok := ctx.Value(permissionKey).(permissionrepository.Permission)
// 	if !ok {
// 		return permissionrepository.Permission{}, errors.New("permission not found in context")
// 	}
// 	return v, nil
// }

// func setUser(ctx context.Context, usr userbus.User) context.Context {
// 	return context.WithValue(ctx, userKey, usr)
// }

// // GetUser returns the user from the context.
// func GetUser(ctx context.Context) (userbus.User, error) {
// 	v, ok := ctx.Value(userKey).(userbus.User)
// 	if !ok {
// 		return userbus.User{}, errors.New("user not found in context")
// 	}

// 	return v, nil
// }

// func setProduct(ctx context.Context, prd productbus.Product) context.Context {
// 	return context.WithValue(ctx, productKey, prd)
// }

// func setTran(ctx context.Context, tx sqldb.CommitRollbacker) context.Context {
// 	return context.WithValue(ctx, trKey, tx)
// }

// // GetTran retrieves the value that can manage a transaction.
// func GetTran(ctx context.Context) (sqldb.CommitRollbacker, error) {
// 	v, ok := ctx.Value(trKey).(sqldb.CommitRollbacker)
// 	if !ok {
// 		return nil, errors.New("transaction not found in context")
// 	}

// 	return v, nil
// }
