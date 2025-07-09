package userrepo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jrazmi/envoker/sdk/logger"
)

// Set of error values for CRUD operations on user resource
var (
	ErrNotFound = errors.New("user not found")
)

type Storer interface {
	List(ctx context.Context) ([]User, error)
	GetByID(ctx context.Context, ID string) (User, error)
}

type Repository struct {
	log    *logger.Logger
	storer Storer
}

func NewRepository(log *logger.Logger, storer Storer) *Repository {
	return &Repository{
		log:    log,
		storer: storer,
	}
}

func (r *Repository) List(ctx context.Context) ([]User, error) {
	records, err := r.storer.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("user repository list: %w", err)
	}

	return records, nil
}

func (r *Repository) GetByID(ctx context.Context, ID string) (User, error) {
	record, err := r.storer.GetByID(ctx, ID)
	if err != nil {
		return User{}, fmt.Errorf("user repository get by id: %w", err)
	}
	return record, nil
}
