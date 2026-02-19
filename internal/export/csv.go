package export

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/vitor-labes/pc-scraper/internal/domain"
)

const outputDir = "exports"

func ToCSV(products []domain.Product) error {
	if len(products) == 0 {
		return fmt.Errorf("nenhum produto para exportar")
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("erro ao criar diretório exports: %w", err)
	}

	filename := fmt.Sprintf("products_%s.csv",
		time.Now().Format("20060102_150405"))

	filepath := filepath.Join(outputDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("erro ao criar arquivo: %w", err)
	}
	defer file.Close()

	file.WriteString("\uFEFF")

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{
		"Categoria", "Marca", "Título", "Preço", "Preço Raw", "Página",
	}); err != nil {
		return fmt.Errorf("erro ao escrever cabeçalho: %w", err)
	}

	sortedProducts := make([]domain.Product, len(products))
	copy(sortedProducts, products)

	sort.Slice(sortedProducts, func(i, j int) bool {
		if sortedProducts[i].Category != sortedProducts[j].Category {
			return sortedProducts[i].Category < sortedProducts[j].Category
		}
		return sortedProducts[i].Price < sortedProducts[j].Price
	})

	for _, p := range sortedProducts {
		if err := writer.Write([]string{
			p.Category,
			p.Brand,
			p.Title,
			fmt.Sprintf("%.2f", p.Price),
			p.RawPrice,
			strconv.Itoa(p.Page),
		}); err != nil {
			slog.Error("erro ao escrever linha",
				"product", p.Title,
				"error", err,
			)
			continue
		}
	}

	if err := writer.Error(); err != nil {
		return fmt.Errorf("erro ao finalizar escrita: %w", err)
	}

	slog.Info("CSV exportado com sucesso",
		"filepath", filepath,
		"total_products", len(products),
	)

	return nil
}
