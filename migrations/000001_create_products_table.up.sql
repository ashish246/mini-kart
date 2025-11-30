-- Create products table
CREATE TABLE IF NOT EXISTS products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    price DECIMAL(10,2) NOT NULL CHECK (price >= 0),
    category TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index on category for category-based queries
CREATE INDEX idx_products_category ON products(category);

-- Create index on created_at for listing queries
CREATE INDEX idx_products_created_at ON products(created_at DESC);
