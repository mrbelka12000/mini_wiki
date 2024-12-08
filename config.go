package mini_wiki

import (
	"context"
	"fmt"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

type (
	// Config of service
	Config struct {
		ServiceName    string `env:"SERVICE_NAME,required"`
		HTTPPort       string `env:"HTTP_PORT, default=8081"`
		PGURL          string `env:"PG_URL,required"`
		MinIOAddr      string `env:"MINIO_ADDR,required"`
		MinIOBucket    string `env:"MINIO_BUCKET,default=wiki"`
		MinIOAccessKey string `env:"MINIO_ACCESS_KEY,required"`
		MinIOSecretKey string `env:"MINIO_SECRET_KEY,required"`
	}
)

// GetConfig
func GetConfig() (Config, error) {
	return parseConfig()
}

func parseConfig() (cfg Config, err error) {
	godotenv.Load()

	err = envconfig.Process(context.Background(), &cfg)
	if err != nil {
		return cfg, fmt.Errorf("fill config: %w", err)
	}

	return cfg, nil
}
