package scraper

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/vitor-labes/pc-scraper/internal/config"
	"github.com/vitor-labes/pc-scraper/internal/domain"
	"github.com/vitor-labes/pc-scraper/internal/metrics"
)

type PichauScraper struct {
	cfg  *config.Config
	seen map[string]bool
}

func NewPichauScraper(cfg *config.Config) *PichauScraper {
	return &PichauScraper{
		cfg:  cfg,
		seen: make(map[string]bool),
	}
}

func ScrapePichau(ctx context.Context, cfg *config.Config) ([]domain.Product, error) {
	scraper := NewPichauScraper(cfg)
	return scraper.Scrape(ctx)
}

func (s *PichauScraper) Scrape(ctx context.Context) ([]domain.Product, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("erro ao iniciar playwright: %w", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(s.cfg.Headless),
		Args: []string{
			"--disable-blink-features=AutomationControlled",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir navegador: %w", err)
	}
	defer browser.Close()

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(s.cfg.UserAgent),
	})
	if err != nil {
		return nil, fmt.Errorf("erro ao criar contexto: %w", err)
	}

	page, err := context.NewPage()
	if err != nil {
		return nil, fmt.Errorf("erro ao criar página: %w", err)
	}

	var allProducts []domain.Product

	for i, category := range s.cfg.Categories {
		slog.Info("iniciando coleta", "category", category.Name)

		products, err := s.scrapeCategory(ctx, page, category)
		if err != nil {
			slog.Error("erro ao scrapear categoria",
				"category", category.Name,
				"error", err,
			)
			continue
		}

		allProducts = append(allProducts, products...)

		// Pause
		if i < len(s.cfg.Categories)-1 {
			slog.Info("pausa entre categorias", "duration", s.cfg.PageDelay)
			time.Sleep(s.cfg.PageDelay)
		}
	}

	return allProducts, nil
}

func (s *PichauScraper) scrapeCategory(
	ctx context.Context,
	page playwright.Page,
	category config.CategoryConfig,
) ([]domain.Product, error) {
	var products []domain.Product

	for pageNum := 1; pageNum <= s.cfg.MaxPages; pageNum++ {
		select {
		case <-ctx.Done():
			return products, ctx.Err()
		default:
		}

		startTime := time.Now()

		url := fmt.Sprintf("%s?page=%d", category.URL, pageNum)
		slog.Info("acessando página",
			"category", category.Name,
			"page", pageNum,
			"url", url,
		)

		if err := s.navigateToPage(page, url); err != nil {
			slog.Error("erro ao navegar",
				"page", pageNum,
				"error", err,
			)
			metrics.PagesProcessed.WithLabelValues(category.Name, "error").Inc()
			break
		}

		if s.detectCloudflare(page) {
			slog.Warn("cloudflare detectado, aguardando resolução manual")
			metrics.CloudflareDetections.Inc()
			time.Sleep(s.cfg.CloudflareWait)
		}

		s.simulateHumanBehavior(page)

		locator := page.Locator(".MuiCard-root")
		count, err := locator.Count()
		if err != nil {
			slog.Error("erro ao contar cards", "error", err)
			continue
		}

		if count == 0 {
			slog.Warn("nenhum card encontrado, tentando novamente")
			time.Sleep(5 * time.Second)
			count, _ = locator.Count()
		}

		if count == 0 {
			slog.Warn("página vazia ou bloqueada",
				"page", pageNum,
				"category", category.Name,
			)
			metrics.PagesProcessed.WithLabelValues(category.Name, "empty").Inc()
			break
		}

		pageProducts, duplicates := s.extractProducts(locator, category, pageNum)
		products = append(products, pageProducts...)

		// Metrics
		duration := time.Since(startTime).Seconds()
		metrics.ScrapingDuration.WithLabelValues(category.Name).Observe(duration)
		metrics.PagesProcessed.WithLabelValues(category.Name, "success").Inc()
		metrics.ProductsScraped.WithLabelValues(category.Name).Add(float64(len(pageProducts)))
		metrics.DuplicatesSkipped.WithLabelValues(category.Name).Add(float64(duplicates))

		slog.Info("página processada",
			"category", category.Name,
			"page", pageNum,
			"new_products", len(pageProducts),
			"duplicates", duplicates,
			"total", len(products),
			"duration_seconds", fmt.Sprintf("%.2f", duration),
		)

		waitTime := s.randomWaitTime()
		slog.Debug("aguardando próxima página", "duration", waitTime)
		time.Sleep(waitTime)
	}

	return products, nil
}

func (s *PichauScraper) navigateToPage(page playwright.Page, url string) error {
	_, err := page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(30000),
	})
	return err
}

func (s *PichauScraper) detectCloudflare(page playwright.Page) bool {
	title, _ := page.Title()
	return strings.Contains(title, "Just a moment") || strings.Contains(title, "Cloudflare")
}

func (s *PichauScraper) simulateHumanBehavior(page playwright.Page) {
	scrollAmount := float64(rand.Intn(500) + 300)
	page.Mouse().Wheel(0, scrollAmount)
	time.Sleep(time.Duration(rand.Intn(2000)+1000) * time.Millisecond)
}

func (s *PichauScraper) extractProducts(
	locator playwright.Locator,
	category config.CategoryConfig,
	pageNum int,
) ([]domain.Product, int) {
	var products []domain.Product
	duplicates := 0

	items, err := locator.All()
	if err != nil {
		slog.Error("erro ao obter itens", "error", err)
		return products, duplicates
	}

	for _, item := range items {
		product, isDuplicate, err := s.extractProduct(item, category, pageNum)
		if err != nil {
			continue
		}

		if isDuplicate {
			duplicates++
			continue
		}

		products = append(products, product)
	}

	return products, duplicates
}

func (s *PichauScraper) extractProduct(
	item playwright.Locator,
	category config.CategoryConfig,
	pageNum int,
) (domain.Product, bool, error) {
	titleText, _ := item.Locator("h2").First().TextContent()
	if titleText == "" {
		titleText, _ = item.Locator(".MuiTypography-root").First().TextContent()
	}

	titleLower := strings.ToLower(titleText)

	if len(category.Targets) > 0 {
		found := false
		for _, target := range category.Targets {
			cleanTarget := strings.ToLower(strings.TrimSpace(target))
			if cleanTarget != "" && strings.Contains(titleLower, cleanTarget) {
				found = true
				break
			}
		}
		if !found {
			return domain.Product{}, false, fmt.Errorf("skip: fora dos alvos")
		}
	}

	priceText, _ := item.Locator("text=/R\\$/").First().TextContent()
	if titleText == "" || priceText == "" {
		return domain.Product{}, false, fmt.Errorf("título ou preço vazio")
	}

	if !strings.Contains(titleLower, category.Filter) {
		return domain.Product{}, false, fmt.Errorf("não corresponde ao filtro")
	}

	price := parsePrice(priceText)
	if price <= 0 {
		return domain.Product{}, false, fmt.Errorf("preço inválido")
	}

	titleClean := strings.TrimSpace(titleText)
	brand := extractBrandFromTitle(titleClean)
	key := fmt.Sprintf("%s|%.2f", titleClean, price)

	if s.seen[key] {
		return domain.Product{}, true, nil
	}

	s.seen[key] = true

	return domain.Product{
		Title:    titleClean,
		Brand:    brand,
		Price:    price,
		RawPrice: strings.TrimSpace(priceText),
		Page:     pageNum,
		Category: category.Name,
	}, false, nil
}

var commonBrands = []string{
	"ASUS", "MSI", "GIGABYTE", "ASROCK", "GALAX", "PNY",
	"INTEL", "AMD", "CORSAIR", "KINGSTON", "XPG", "LOGITECH",
	"RAZER", "REDRAGON", "SAMSUNG", "LG", "AOC", "HUSKY",
	"MANCER", "PICHAU", "NVIDIA", "ZOTAC", "COLORFUL", "GAINWARD",
	"SAPPHIRE", "POWERCOLOR", "XFX", "INNO3D",
}

func extractBrandFromTitle(title string) string {
	titleUpper := strings.ToUpper(title)
	for _, brand := range commonBrands {
		if strings.Contains(titleUpper, brand) {
			return brand
		}
	}
	return "OUTROS"
}

func (s *PichauScraper) randomWaitTime() time.Duration {
	min := s.cfg.WaitTimeMin.Seconds()
	max := s.cfg.WaitTimeMax.Seconds()
	wait := min + rand.Float64()*(max-min)
	return time.Duration(wait * float64(time.Second))
}

func parsePrice(raw string) float64 {
	clean := strings.ReplaceAll(raw, "R$", "")
	clean = strings.ReplaceAll(clean, "R$Â", "")
	clean = strings.ReplaceAll(clean, "Â", "")
	clean = strings.ReplaceAll(clean, ".", "")
	clean = strings.ReplaceAll(clean, ",", ".")
	clean = strings.TrimSpace(clean)

	fields := strings.Fields(clean)
	if len(fields) > 0 {
		clean = fields[len(fields)-1]
	}

	value, _ := strconv.ParseFloat(clean, 64)
	return value
}
