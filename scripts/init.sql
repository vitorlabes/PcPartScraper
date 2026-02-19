CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    brand VARCHAR(50),
    price DECIMAL(10, 2) NOT NULL,
    raw_price VARCHAR(50),
    page_number INTEGER,
    category VARCHAR(50) NOT NULL,
    scraped_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_products_category ON products(category);
CREATE INDEX idx_products_price ON products(price);
CREATE INDEX idx_products_scraped_at ON products(scraped_at);
CREATE INDEX idx_products_title ON products(title);

CREATE TABLE IF NOT EXISTS price_history (
    id SERIAL PRIMARY KEY,
    product_title VARCHAR(500) NOT NULL,
    category VARCHAR(50) NOT NULL,
    old_price DECIMAL(10, 2),
    new_price DECIMAL(10, 2) NOT NULL,
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_price_history_title ON price_history(product_title);
CREATE INDEX idx_price_history_changed_at ON price_history(changed_at);

CREATE OR REPLACE VIEW v_best_prices AS
SELECT DISTINCT ON (title, category)
    title,
    category,
    price,
    raw_price,
    scraped_at
FROM products
ORDER BY title, category, price ASC, scraped_at DESC;

COMMENT ON TABLE products IS 'Produtos scrapeados da Pichau';
COMMENT ON TABLE price_history IS 'Histórico de mudanças de preço';
COMMENT ON VIEW v_best_prices IS 'Melhores preços por produto';