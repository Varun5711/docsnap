package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/docsnap/docsnap/services/api/internal/ai"
	"github.com/docsnap/docsnap/services/api/internal/config"
	"github.com/docsnap/docsnap/services/api/internal/evidence"
	"github.com/docsnap/docsnap/services/api/internal/flare"
	"github.com/docsnap/docsnap/services/api/internal/httpapi"
	"github.com/docsnap/docsnap/services/api/internal/store"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()
	repo, err := store.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer repo.Close()

	extractor := ai.Extractor(ai.NewRuleExtractor())
	if cfg.GroqAPIKey != "" {
		extractor = ai.NewGroqExtractor(cfg.GroqAPIKey, cfg.GroqBaseURL, cfg.GroqModel)
	}

	hasher := evidence.NewHasher()
	anchor := flare.NewSimulatedClient()
	server := httpapi.NewServer(cfg, repo, extractor, hasher, anchor)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("docsnap api listening on %s", cfg.Addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
