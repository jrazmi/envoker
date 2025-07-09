package admin

import (
	"embed"
	"fmt"

	"github.com/jrazmi/envoker/infrastructure/web"
)

//go:embed react/dist
var adminStaticFiles embed.FS

func AddHandlers(app *web.App) *web.App {

	// Use the infrastructure React file server instead of custom handlers
	if err := app.FileServerReact(adminStaticFiles, "react/dist", "/admin/"); err != nil {
		fmt.Printf("Error setting up admin React server: %v\n", err)
		return app
	}

	return app
}
