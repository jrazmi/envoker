package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jrazmi/envoker/core/scaffolding/fop"
	"github.com/jrazmi/envoker/sdk/cryptids"
)

var (
	ErrOperationNotSupported = errors.New("operation not supported")
	ErrNotFound              = errors.New("record not found")
)

type CreatePayload interface {
	GetID() *string
	SetID(id string)
}

type UpdatePayload interface {
	GetID() *string
}

type Reader[T any, F any] interface {
	Get(ctx context.Context, id string, filter F) (T, error)
	List(ctx context.Context, filter F, orderBy fop.By, page fop.PageStringCursor) ([]T, error)
}

type Writer[T any, C CreatePayload] interface {
	Create(ctx context.Context, payload C) (T, error)
}

type Updater[T any, U UpdatePayload] interface {
	Update(ctx context.Context, id string, updates U) error
}

type Deleter interface {
	Delete(ctx context.Context, id string) error
}

type Archiver interface {
	Archive(ctx context.Context, id string) error
}

type Repository[T any, C CreatePayload, U UpdatePayload, F any] struct {
	Reader   Reader[T, F]
	Writer   Writer[T, C]
	Updater  Updater[T, U]
	Deleter  Deleter
	Archiver Archiver
}

func (r *Repository[T, C, U, F]) GenerateID() (string, error) {
	return cryptids.GenerateID()
}

// Add the Create method to CRUDRepository
func (r *Repository[T, C, U, F]) Create(ctx context.Context, payload C) (T, error) {
	var zero T
	if r.Writer == nil {
		return zero, ErrOperationNotSupported
	}
	// Generate ID if not provided
	if payload.GetID() == nil || *payload.GetID() == "" {
		id, err := r.GenerateID()
		if err != nil {
			return zero, fmt.Errorf("generate id: %w", err)
		}
		payload.SetID(id)
	}

	record, err := r.Writer.Create(ctx, payload)
	if err != nil {
		return zero, fmt.Errorf("create record: %w", err)
	}

	return record, nil
}

func (r *Repository[T, C, U, F]) Get(ctx context.Context, id string, filter F) (T, error) {
	var zero T
	if r.Reader == nil {
		return zero, ErrOperationNotSupported
	}
	return r.Reader.Get(ctx, id, filter)
}

func (r *Repository[T, C, U, F]) Update(ctx context.Context, id string, updates U) error {
	if r.Updater == nil {
		return ErrOperationNotSupported
	}
	return r.Updater.Update(ctx, id, updates)
}

func (r *Repository[T, C, U, F]) Delete(ctx context.Context, id string) error {
	if r.Deleter == nil {
		return ErrOperationNotSupported
	}
	return r.Deleter.Delete(ctx, id)
}

func (r *Repository[T, C, U, F]) Archive(ctx context.Context, id string) error {
	if r.Archiver == nil {
		return ErrOperationNotSupported
	}
	return r.Archiver.Archive(ctx, id)
}

func (r *Repository[T, C, U, F]) List(ctx context.Context, filter F, orderBy fop.By, page fop.PageStringCursor) ([]T, error) {
	if r.Reader == nil {
		return nil, ErrOperationNotSupported
	}
	return r.Reader.List(ctx, filter, orderBy, page)
}
