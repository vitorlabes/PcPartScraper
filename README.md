# 🖥️ PC Scraper

A distributed web scraper built in Go that collects GPU and CPU prices from [Pichau](https://www.pichau.com.br), publishes them to a RabbitMQ queue, persists data to PostgreSQL, and exposes metrics via Prometheus + Grafana.

## Architecture

```
┌─────────────┐     ┌───────────────┐     ┌────────────┐
│   Scraper   │────▶│   RabbitMQ    │────▶│  Consumer  │
│ (Playwright)│     │ product_prices│     │            │
└─────────────┘     └───────────────┘     └─────┬──────┘
      │                                          │
      │ metrics :2114                            │ metrics :2113
      │                                    ┌─────▼──────┐
      │                                    │ PostgreSQL │
      │                                    └────────────┘
      │
      ▼
┌────────────┐     ┌────────────┐
│ Prometheus │────▶│  Grafana   │
│   :9090    │     │   :3000    │
└────────────┘     └────────────┘
```

- **Scraper** — Uses Playwright (Chromium) to scrape product listings, with Cloudflare bypass handling, duplicate filtering, and configurable category targets. Publishes products to RabbitMQ and exports a CSV.
- **Consumer** — Reads messages from RabbitMQ and persists them to PostgreSQL with insert metrics.
- **Prometheus** — Scrapes `/metrics` from both services.
- **Grafana** — Dashboards for scraping activity, success rates, DB inserts, and more. *(Currently being implemented)*

## Stack

| Layer | Technology |
|---|---|
| Language | Go 1.22 |
| Browser Automation | Playwright (Chromium) |
| Message Queue | RabbitMQ 3.12 |
| Database | PostgreSQL 15 |
| Metrics | Prometheus + Grafana |
| Containerization | Docker / Docker Compose |

## Getting Started

### Prerequisites

- Docker & Docker Compose
- A `.env` file (see below)

### Environment Variables

Create a `.env` file at the project root:

```env
POSTGRES_USER=your_postgres_user
POSTGRES_PASSWORD=your_postgres_password
POSTGRES_DB=your_database_name

RABBITMQ_USER=your_rabbitmq_user
RABBITMQ_PASSWORD=your_rabbitmq_password

GRAFANA_USER=your_grafana_user
GRAFANA_PASSWORD=your_grafana_password
```

### Running

```bash
docker compose up --build
```

This will start:

| Service | URL |
|---|---|
| RabbitMQ Management | http://localhost:15672 |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 |
| Scraper metrics | http://localhost:2114/metrics |
| Consumer metrics | http://localhost:2113/metrics |

## Configuration

The scraper supports optional environment variables to filter specific products:

```bash
# Only scrape specific GPU/CPU models (comma-separated)
GPU_TARGETS="RTX 4060,RX 7600"
CPU_TARGETS="i7 14700,Ryzen 7 7700"

# Run browser in headless mode (default: true in Docker)
HEADLESS=true
```

Default scraper config (defined in `internal/config/config.go`):

| Parameter | Default |
|---|---|
| Max pages per category | 5 |
| Wait between pages | 4–9s (randomized) |
| Delay between categories | 10s |
| Cloudflare wait | 30s |
| Retry attempts | 3 |

## Metrics

### Scraper (`:2114/metrics`)

| Metric | Description |
|---|---|
| `scraper_products_scraped_total` | Total products scraped, by category |
| `scraper_pages_processed_total` | Pages processed, by category and status |
| `scraper_page_duration_seconds` | Scraping duration histogram per page |
| `scraper_cloudflare_detections_total` | Number of Cloudflare challenges hit |
| `scraper_duplicates_skipped_total` | Duplicate products skipped per category |

### Consumer (`:2113/metrics`)

| Metric | Description |
|---|---|
| `consumer_messages_processed_total` | Messages processed, by status (success/error) |
| `consumer_message_processing_duration_seconds` | Processing time histogram |
| `consumer_database_inserts_total` | DB insert attempts, by status |
| `consumer_queue_depth` | Current RabbitMQ queue depth |

## Grafana Dashboards

Dashboards are provisioned automatically from `configs/dashboards/`. The **PC Scraper - Overview** dashboard includes:

- Total products scraped
- Messages processed
- Success rate (gauge)
- Cloudflare detections
- Products scraped by category (timeseries)
- Scraping duration p95 (timeseries)
- Database insert status (pie chart)
- Duplicates skipped over time
- Average message processing duration

> **Note:** Grafana integration is currently being implemented.

## Database Schema

```sql
-- Main table
products (id, title, brand, price, raw_price, page_number, category, scraped_at)

-- Price change history
price_history (id, product_title, category, old_price, new_price, changed_at)

-- View: best price per product
v_best_prices
```

## CSV Export

After each scraper run, a timestamped CSV is saved to `./exports/`:

```
exports/products_20240315_143022.csv
```

Columns: `Categoria, Marca, Título, Preço, Preço Raw, Página`
