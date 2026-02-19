package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vitor-labes/pc-scraper/internal/domain"
	"github.com/vitor-labes/pc-scraper/internal/metrics"
	"github.com/vitor-labes/pc-scraper/internal/queue"
	"github.com/vitor-labes/pc-scraper/internal/repository"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Metrics
	go func() {
		slog.Info("iniciando servidor de métricas", "port", "2113")
		if err := metrics.StartMetricsServer(":2113"); err != nil {
			log.Fatalf("erro ao iniciar servidor de métricas: %v", err)
		}
	}()

	// Environment
	databaseURL := getEnv("DATABASE_URL", "postgres://scraper:scraper123@localhost:5432/products?sslmode=disable")
	rabbitmqURL := getEnv("RABBITMQ_URL", "amqp://scraper:scraper123@localhost:5672/")
	queueName := getEnv("QUEUE_NAME", "product_prices")

	slog.Info("iniciando consumer",
		"queue", queueName,
	)

	// Conection
	repo, err := repository.NewProductRepository(databaseURL)
	if err != nil {
		log.Fatalf("erro ao conectar no banco: %v", err)
	}
	defer repo.Close()

	// Save metrics DB
	handler := func(ctx context.Context, product domain.Product) error {
		startTime := time.Now()

		err := repo.Save(ctx, product)

		duration := time.Since(startTime).Seconds()
		metrics.MessageProcessingDuration.Observe(duration)

		if err != nil {
			metrics.MessagesProcessed.WithLabelValues("error").Inc()
			metrics.DatabaseInserts.WithLabelValues("error").Inc()
			return err
		}

		metrics.MessagesProcessed.WithLabelValues("success").Inc()
		metrics.DatabaseInserts.WithLabelValues("success").Inc()
		return nil
	}

	// Create consumer
	consumer, err := queue.NewConsumer(rabbitmqURL, queueName, handler)
	if err != nil {
		log.Fatalf("erro ao criar consumer: %v", err)
	}
	defer consumer.Close()

	// Context cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Goroutine consumer
	errChan := make(chan error, 1)
	go func() {
		errChan <- consumer.Start(ctx)
	}()

	// Await signal
	select {
	case sig := <-sigChan:
		slog.Info("sinal recebido, encerrando...", "signal", sig)
		cancel()
	case err := <-errChan:
		if err != nil && err != context.Canceled {
			log.Fatalf("erro no consumer: %v", err)
		}
	}

	slog.Info("consumer encerrado com sucesso")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
