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
	UserID string  `db:"user_id"`
	Email  *string `db:"email"`
}

func (c *CreateUser) GetID() *string {
	return &c.UserID
}
func (c *CreateUser) SetID(ID string) {
	c.UserID = ID
}

type UpdateUser struct {
	UserID *string `db:"user_id"`
	Email  *string `db:"email"`
}

func (u *UpdateUser) GetID() *string {
	return u.UserID
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
	repositories.Reader[User, UserFilter]
	repositories.Writer[User, *CreateUser]
	repositories.Updater[User, *UpdateUser]
	repositories.Deleter
	repositories.Archiver
}

type UserRepository struct {
	log *logger.Logger
	repositories.Repository[User, *CreateUser, *UpdateUser, UserFilter]
}

func NewUserRepository(log *logger.Logger, storer UserStorer) *UserRepository {
	return &UserRepository{
		Repository: repositories.Repository[User, *CreateUser, *UpdateUser, UserFilter]{
			Reader:   storer,
			Writer:   storer,
			Updater:  storer,
			Deleter:  storer,
			Archiver: storer,
		},
		log: log,
	}
}
