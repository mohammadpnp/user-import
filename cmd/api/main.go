package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	app "github.com/mohammadpnp/user-import/internal/application/user"
	"github.com/mohammadpnp/user-import/internal/bootstrap"
	infrafile "github.com/mohammadpnp/user-import/internal/infrastructure/file"
	"github.com/mohammadpnp/user-import/internal/infrastructure/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatalf("failed to create pgx pool: %v", err)
	}
	defer pool.Close()

	server := bootstrap.NewHTTPServer(db)
	workerCtx, stopWorkers := context.WithCancel(context.Background())
	defer stopWorkers()

	importJobRepo := repository.NewImportJobRepository(db)
	userImporter := repository.NewUserBulkImportRepository(pool)
	sourceReader := infrafile.NewLocalSource(getEnv("IMPORT_BASE_DIR", "."))

	worker := app.NewImportWorker(importJobRepo, sourceReader, userImporter, app.ImportWorkerConfig{
		Workers:       parseWorkerCount(),
		ChunkSize:     parseIntEnv("IMPORT_CHUNK_SIZE", 10000),
		LeaseDuration: time.Duration(parseIntEnv("IMPORT_JOB_LEASE_SECONDS", 60)) * time.Second,
	})
	worker.Start(workerCtx)

	go func() {
		if err := server.Start(":" + port); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	stopWorkers()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}
}

func parseWorkerCount() int {
	workers := parseIntEnv("IMPORT_WORKERS", 10)
	if workers <= 0 {
		return 10
	}
	if workers > 10 {
		return 10
	}
	return workers
}

func parseIntEnv(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
