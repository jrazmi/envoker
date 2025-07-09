package web

import (
	"log"
	"net/http"
)

func NewWebServer(cfg ServerConfig, handler http.Handler, log *log.Logger) http.Server {
	return http.Server{
		Addr:         cfg.Port,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
		ErrorLog:     log,
	}
}
