package api

import (
	"fmt"

	"github.com/jrazmi/envoker/infrastructure/web"
)

func AddHandlers(app *web.App) *web.App {
	fmt.Println("API HANDLER")
	return app
}
