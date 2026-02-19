package scraper

import (
	"context"

	"github.com/vitor-labes/pc-scraper/internal/domain"
)

type Scraper interface {
	Scrape(ctx context.Context) ([]domain.Product, error)
}
