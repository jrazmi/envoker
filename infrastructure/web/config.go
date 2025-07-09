package web

import (
	"time"

	"github.com/jrazmi/envoker/sdk/environment"
)

var (
	globalMWOrder = []string{"cors", "cache", "compression", "ratelimit", "logger", "errors", "metrics", "panics"}
)

type ServerConfig struct {
	Port            string        `env:"PORT" default:":3000"`
	ApiRoute        string        `env:"API_ROUTE" default:"/api/v1"`
	CORSOrigins     []string      `env:"CORS_ORIGINS" default:"http://localhost:3000" separator:","`
	JwtSigningKey   string        `env:"JWT_SIGNING_KEY" required:"true"`
	EnableDebug     bool          `env:"ENABLE_DEBUG" default:"false"`
	ReadTimeout     time.Duration `env:"READ_TIMEOUT" default:"5s"`
	WriteTimeout    time.Duration `env:"WRITE_TIMEOUT" default:"10s"`
	IdleTimeout     time.Duration `env:"IDLE_TIMEOUT" default:"120s"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" default:"20s"`
}

func LoadServerConfig(prefix string) (ServerConfig, error) {
	var cfg ServerConfig
	if err := environment.ParseEnvTags(prefix, &cfg); err != nil {
		return ServerConfig{}, err
	}
	return cfg, nil
}
