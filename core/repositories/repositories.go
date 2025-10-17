package repositories

import (
	"context"
	"errors"

	"github.com/jrazmi/envoker/core/scaffolding/fop"
)

var (
	ErrOperationNotSupported = errors.New("operation not supported")
	ErrNotFound              = errors.New("record not found")
)

// Store is a unified interface for all CRUD operations
type Store[T any, ID comparable, C any, U any, F any] interface {
	Create(ctx context.Context, payload C) (T, error)
	Get(ctx context.Context, id ID, filter F) (T, error)
	List(ctx context.Context, filter F, orderBy fop.By, page fop.PageStringCursor) ([]T, error)
	Update(ctx context.Context, id ID, updates U) error
	Delete(ctx context.Context, id ID) error
	Archive(ctx context.Context, id ID) error
}
