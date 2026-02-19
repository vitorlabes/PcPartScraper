package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	_ "github.com/lib/pq"
	"github.com/vitor-labes/pc-scraper/internal/domain"
)

type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(databaseURL string) (*ProductRepository, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir conexão: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("erro ao conectar no banco: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	slog.Info("conectado ao PostgreSQL")

	return &ProductRepository{db: db}, nil
}

func (r *ProductRepository) Save(ctx context.Context, product domain.Product) error {
	query := `
		INSERT INTO products (title, brand, price, raw_price, page_number, category)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	var id int
	err := r.db.QueryRowContext(
		ctx,
		query,
		product.Title,
		product.Brand,
		product.Price,
		product.RawPrice,
		product.Page,
		product.Category,
	).Scan(&id)

	if err != nil {
		return fmt.Errorf("erro ao inserir produto: %w", err)
	}

	slog.Info("produto salvo no banco",
		"id", id,
		"title", product.Title,
		"price", product.Price,
	)

	return nil
}

func (r *ProductRepository) FindBestPrices(ctx context.Context, category string) ([]domain.Product, error) {
	query := `
		SELECT title, category, price, raw_price
		FROM v_best_prices
		WHERE category = $1
		ORDER BY price ASC
		LIMIT 20
	`

	rows, err := r.db.QueryContext(ctx, query, category)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar melhores preços: %w", err)
	}
	defer rows.Close()

	var products []domain.Product
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.Title, &p.Category, &p.Price, &p.RawPrice); err != nil {
			return nil, fmt.Errorf("erro ao escanear linha: %w", err)
		}
		products = append(products, p)
	}

	return products, nil
}

func (r *ProductRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(DISTINCT category) as categories,
			MIN(price) as min_price,
			MAX(price) as max_price,
			AVG(price) as avg_price
		FROM products
	`

	var stats struct {
		Total      int
		Categories int
		MinPrice   float64
		MaxPrice   float64
		AvgPrice   float64
	}

	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.Total,
		&stats.Categories,
		&stats.MinPrice,
		&stats.MaxPrice,
		&stats.AvgPrice,
	)

	if err != nil {
		return nil, fmt.Errorf("erro ao buscar estatísticas: %w", err)
	}

	return map[string]interface{}{
		"total_products": stats.Total,
		"categories":     stats.Categories,
		"min_price":      stats.MinPrice,
		"max_price":      stats.MaxPrice,
		"avg_price":      stats.AvgPrice,
	}, nil
}

func (r *ProductRepository) Close() error {
	return r.db.Close()
}
