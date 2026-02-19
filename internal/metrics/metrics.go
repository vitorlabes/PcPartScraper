package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Scraper
	ProductsScraped = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scraper_products_scraped_total",
			Help: "Total number of products scraped",
		},
		[]string{"category"},
	)

	PagesProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scraper_pages_processed_total",
			Help: "Total number of pages processed",
		},
		[]string{"category", "status"},
	)

	ScrapingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "scraper_page_duration_seconds",
			Help:    "Time taken to scrape a page",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"category"},
	)

	CloudflareDetections = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "scraper_cloudflare_detections_total",
			Help: "Total number of Cloudflare challenges detected",
		},
	)

	DuplicatesSkipped = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "scraper_duplicates_skipped_total",
			Help: "Total number of duplicate products skipped",
		},
		[]string{"category"},
	)

	// Consumer
	MessagesProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "consumer_messages_processed_total",
			Help: "Total number of messages processed",
		},
		[]string{"status"},
	)

	MessageProcessingDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "consumer_message_processing_duration_seconds",
			Help:    "Time taken to process a message",
			Buckets: prometheus.DefBuckets,
		},
	)

	DatabaseInserts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "consumer_database_inserts_total",
			Help: "Total number of database inserts",
		},
		[]string{"status"},
	)

	QueueDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "consumer_queue_depth",
			Help: "Current depth of RabbitMQ queue",
		},
	)
)

func StartMetricsServer(addr string) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(addr, nil)
}
