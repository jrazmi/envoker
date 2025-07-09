package userpgxstore

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jrazmi/envoker/core/repositories/userrepo"
	"github.com/jrazmi/envoker/infrastructure/datastores/postgresdb"
	"github.com/jrazmi/envoker/sdk/logger"
)

type Store struct {
	log  *logger.Logger
	pool *postgresdb.Pool
}

func NewStore(log *logger.Logger, pool *postgresdb.Pool) *Store {
	return &Store{
		log:  log,
		pool: pool,
	}
}

func (s *Store) List(ctx context.Context) ([]userrepo.User, error) {
	query := `SELECT user_id, email 
			FROM users 
			WHERE status = @status`

	args := pgx.NamedArgs{"status": postgresdb.StatusActive}
	rows, err := s.pool.Query(ctx, query, args)
	if err != nil {
		return nil, postgresdb.HandlePgError(err)
	}
	defer rows.Close()

	sl, err := pgx.CollectRows(rows, pgx.RowToStructByName[userrepo.User])
	return sl, err
}

func (s *Store) GetByID(ctx context.Context, ID string) (userrepo.User, error) {
	query := `SELECT user_id, email 
		FROM users 
		WHERE user_id = @user_id`

	args := pgx.NamedArgs{
		"user_id": ID,
	}

	rows, err := s.pool.Query(ctx, query, args)
	if err != nil {
		return userrepo.User{}, postgresdb.HandlePgError(err)
	}
	defer rows.Close()

	// CollectOneRow returns the first row, or pgx.ErrNoRows if no rows
	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[userrepo.User])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return userrepo.User{}, fmt.Errorf("user with ID %s not found", ID)
		}
		return userrepo.User{}, postgresdb.HandlePgError(err)
	}

	return user, nil
}
