// Package mid provides app level middleware support.
package mid

import (
	"context"
	"errors"

	"github.com/jrazmi/envoker/infrastructure/web"
)

type ctxKey int

const (
	claimKey ctxKey = iota + 1
	userIDKey
)

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

// isError tests if the Encoder has an error inside of it.
func isError(e web.Encoder) error {
	err, isError := e.(error)
	if isError {
		return err
	}
	return nil
}
