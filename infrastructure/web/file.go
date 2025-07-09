package web

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"regexp"
)

// FileServerReact starts a file server based on the specified file system and
// directory inside that file system for a statically built react webapp.
func (a *App) FileServerReact(static embed.FS, dir string, path string) error {
	fileMatcher := regexp.MustCompile(`\.[a-zA-Z]*$`)

	fSys, err := fs.Sub(static, dir)
	if err != nil {
		return fmt.Errorf("switching to static folder: %w", err)
	}

	fileServer := http.StripPrefix(path, http.FileServer(http.FS(fSys)))

	h := func(w http.ResponseWriter, r *http.Request) {
		if !fileMatcher.MatchString(r.URL.Path) {
			p, err := static.ReadFile(fmt.Sprintf("%s/index.html", dir))
			if err != nil {
				a.log.Error(r.Context(), "FileServerReact", "index.html not found", "ERROR", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(p)
			return
		}

		fileServer.ServeHTTP(w, r)
	}

	a.mux.HandleFunc(fmt.Sprintf("GET %s", path), h)

	return nil
}

// FileServer starts a file server based on the specified file system and
// directory inside that file system.
func (a *App) FileServer(static embed.FS, dir string, path string) error {
	fSys, err := fs.Sub(static, dir)
	if err != nil {
		return fmt.Errorf("switching to static folder: %w", err)
	}

	fileServer := http.StripPrefix(path, http.FileServer(http.FS(fSys)))

	a.mux.Handle(fmt.Sprintf("GET %s", path), fileServer)

	return nil
}
