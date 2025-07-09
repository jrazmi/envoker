// DONT LIKE
package repositories

import (
	"github.com/jrazmi/envoker/core/repositories/userrepo"
	"github.com/jrazmi/envoker/core/repositories/userrepo/stores/userpgxstore"
	"github.com/jrazmi/envoker/infrastructure/datastores/postgresdb"
	"github.com/jrazmi/envoker/sdk/logger"
)

type EnvokerRepositories struct {
	UserRepository    *userrepo.Repository
	ContentRepository string
}

func NewPostgresRepositories(log *logger.Logger, pool *postgresdb.Pool) EnvokerRepositories {
	userStore := userpgxstore.NewStore(log, pool)
	userRepository := userrepo.NewRepository(log, userStore)

	return EnvokerRepositories{
		UserRepository:    userRepository,
		ContentRepository: "foo",
	}
}

// func NewSQLiteRepositories(log *logger.Logger, datastore string){}
// func NewMYSQLRepositories(log *logger.Logger, datastore string){}
// func NewFirestoreRepositories(log *logger.Logger, datastore string){}
// func NewDynamoDBRepositories(log *logger.Logger, datastore string){}
