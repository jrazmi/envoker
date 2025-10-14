package userspgxstore

//go:generate go run ../../../../../workshop/tools/gen/stores/main.go -entity=User -table=users -pk=user_id

import (
	"github.com/jrazmi/envoker/infrastructure/databases/postgresdb"
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
