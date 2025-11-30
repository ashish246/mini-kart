-- Create order_items table
CREATE TABLE IF NOT EXISTS order_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id TEXT NOT NULL REFERENCES products(id),
    quantity INTEGER NOT NULL CHECK (quantity > 0)
);

-- Create index on order_id for order retrieval
CREATE INDEX idx_order_items_order_id ON order_items(order_id);

-- Create index on product_id for product analytics
CREATE INDEX idx_order_items_product_id ON order_items(product_id);
