package environment

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv(p string) error {
	return godotenv.Load(p)
}

func GetEnvOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func GetEnvKeyPrefix(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return fmt.Sprintf("%s_%s", prefix, key)
}
func GetPrefixEnvOrDefault(prefix, key, fallback string) string {
	if prefix == "" {
		return GetEnvOrDefault(key, fallback)
	}
	return GetEnvOrDefault(fmt.Sprintf("%s_%s", prefix, key), fallback)
}

func GetPrefixEnv(prefix, key string) string {
	if prefix == "" {
		return os.Getenv(key)
	}
	return os.Getenv(fmt.Sprintf("%s_%s", prefix, key))
}
