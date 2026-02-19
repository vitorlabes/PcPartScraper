package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vitor-labes/pc-scraper/internal/config"
	"github.com/vitor-labes/pc-scraper/internal/export"
	"github.com/vitor-labes/pc-scraper/internal/metrics"
	"github.com/vitor-labes/pc-scraper/internal/queue"
	"github.com/vitor-labes/pc-scraper/internal/scraper"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Metrics
	go func() {
		slog.Info("iniciando servidor de métricas", "port", "2114")
		if err := metrics.StartMetricsServer(":2114"); err != nil {
			log.Fatalf("erro ao iniciar servidor de métricas: %v", err)
		}
	}()

	cfg := config.NewDefault()

	// Environment
	rabbitmqURL := getEnv("RABBITMQ_URL", "amqp://scraper:scraper123@localhost:5672/")
	queueName := getEnv("QUEUE_NAME", "product_prices")

	// Filter use example: ./scraper -gpus="RTX 4060,RX 5070" -cpus="i7 14700,Ryzen 7 7700"
	gpuTargetsRaw := getEnv("GPU_TARGETS", "")
	cpuTargetsRaw := getEnv("CPU_TARGETS", "")

	cfg.Headless = getEnvBool("HEADLESS", cfg.Headless)

	// Filters
	if gpuTargetsRaw != "" {
		for i, cat := range cfg.Categories {
			if cat.Name == "GPU" {
				cfg.Categories[i].Targets = strings.Split(gpuTargetsRaw, ",")
				slog.Info("alvos de GPU configurados", "targets", cfg.Categories[i].Targets)
			}
		}
	}

	if cpuTargetsRaw != "" {
		for i, cat := range cfg.Categories {
			if cat.Name == "CPU" {
				cfg.Categories[i].Targets = strings.Split(cpuTargetsRaw, ",")
				slog.Info("alvos de CPU configurados", "targets", cfg.Categories[i].Targets)
			}
		}
	}

	slog.Info("iniciando scraper",
		"max_pages", cfg.MaxPages,
		"headless", cfg.Headless,
		"queue", queueName,
	)

	// Connect RabbitMQ
	publisher, err := queue.NewPublisher(rabbitmqURL, queueName)
	if err != nil {
		log.Fatalf("erro ao conectar no RabbitMQ: %v", err)
	}
	defer publisher.Close()

	// Timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Execute
	products, err := scraper.ScrapePichau(ctx, cfg)
	if err != nil {
		log.Fatalf("erro no scraper: %v", err)
	}

	slog.Info("scraping concluído",
		"total_products_found", len(products),
	)

	// Publish on queue
	publishedCount := 0
	for _, product := range products {
		if err := publisher.Publish(ctx, product); err != nil {
			slog.Error("erro ao publicar produto",
				"title", product.Title,
				"error", err,
			)
			continue
		}
		publishedCount++
	}

	slog.Info("publicação finalizada",
		"total_published", publishedCount,
		"failed", len(products)-publishedCount,
	)

	// Export
	slog.Info("gerando arquivo CSV...")
	if err := export.ToCSV(products); err != nil {
		slog.Error("erro ao exportar CSV", "error", err)
	} else {
		slog.Info("CSV gerado com sucesso na pasta exports/")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		b, err := strconv.ParseBool(value)
		if err == nil {
			return b
		}
	}
	return defaultValue
}
