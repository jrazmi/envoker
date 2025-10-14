// bridge/repositories/tasksrepobridge/tasksrepobridge.go
package tasksrepobridge

import (
	"github.com/jrazmi/envoker/core/repositories/tasksrepo"
)

type bridge struct {
	tasksRepository *tasksrepo.Repository
}

func newBridge(tasksRepository *tasksrepo.Repository) *bridge {
	return &bridge{
		tasksRepository: tasksRepository,
	}
}
