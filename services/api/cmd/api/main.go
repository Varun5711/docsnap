package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/docsnap/docsnap/services/api/internal/ai"
	"github.com/docsnap/docsnap/services/api/internal/config"
	"github.com/docsnap/docsnap/services/api/internal/evidence"
	"github.com/docsnap/docsnap/services/api/internal/flare"
	"github.com/docsnap/docsnap/services/api/internal/httpapi"
	"github.com/docsnap/docsnap/services/api/internal/search"
	"github.com/docsnap/docsnap/services/api/internal/storage"
	"github.com/docsnap/docsnap/services/api/internal/store"
	"github.com/docsnap/docsnap/services/api/internal/tee"
	"github.com/docsnap/docsnap/services/api/internal/verdict"
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
	anchor, err := flare.NewClient(ctx, flare.Config{
		Mode:                  cfg.FlareMode,
		RPCURL:                cfg.FlareRPCURL,
		ContractAddress:       cfg.ContractAddr,
		PrivateKey:            cfg.FlareKey,
		TEEReporterPrivateKey: cfg.TEEKey,
	})
	if err != nil {
		log.Fatal(err)
	}
	objectStore, err := storage.NewS3(ctx, cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket, cfg.S3UseSSL)
	if err != nil {
		log.Fatal(err)
	}
	certifier := tee.Certifier(tee.NewLocalCertifier())
	if cfg.TEEURL != "" {
		certifier = tee.NewHTTPClient(cfg.TEEURL)
	}
	searchProvider := search.NewTavilyProvider(cfg.TavilyAPIKey)
	verdictGenerator := verdict.NewGenerator(cfg.GroqAPIKey, cfg.GroqBaseURL, cfg.VerdictModel)
	server := httpapi.NewServer(cfg, repo, extractor, hasher, anchor, objectStore, certifier, searchProvider, verdictGenerator)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	shutdownCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-shutdownCtx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()

	log.Printf("docsnap api listening on %s", cfg.Addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
