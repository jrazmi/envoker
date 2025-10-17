package usersrepo

import (
	"errors"

	"github.com/jrazmi/envoker/core/repositories"
	"github.com/jrazmi/envoker/sdk/logger"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

type CreateUser struct {
	Email *string `db:"email"`
}

type UpdateUser struct {
	Email *string `db:"email"`
}

type User struct {
	UserID string `db:"user_id"`
	Email  string `db:"email"`
}
type UserFilter struct {
	ID    *string
	Email *string
}

type UserStorer interface {
	repositories.Store[User, string, *CreateUser, *UpdateUser, UserFilter]
}

type UserRepository struct {
	log *logger.Logger
	repositories.Store[User, string, *CreateUser, *UpdateUser, UserFilter]
}

func NewUserRepository(log *logger.Logger, storer UserStorer) *UserRepository {
	return &UserRepository{
		Store: storer,
		log:   log,
	}
}
