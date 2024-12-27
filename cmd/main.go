package main

import (
	"log/slog"
	"net/http"
	"os"

	miniwiki "github.com/mrbelka12000/mini_wiki"
)

func main() {

	cfg, err := miniwiki.GetConfig()
	if err != nil {
		panic(err)
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service_name", cfg.ServiceName)

	db, err := miniwiki.DatabaseConnect(cfg)
	if err != nil {
		log.With("error", err).Error("failed to connect to database")
		return
	}
	defer db.Close()

	storage, err := miniwiki.GetStorage(cfg)
	if err != nil {
		log.With("error", err).Error("failed to connect to storage")
		return
	}

	mux := http.NewServeMux()

	log.Info("service started on port " + cfg.HTTPPort)
	if err := miniwiki.RunService(db, storage, log, mux, cfg); err != nil {
		log.With("error", err).Error("failed to run service")
	}
}
