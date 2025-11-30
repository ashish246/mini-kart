-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create orders table
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    coupon_code TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index on created_at for order listing and queries
CREATE INDEX idx_orders_created_at ON orders(created_at DESC);

-- Create index on coupon_code for analytics queries
CREATE INDEX idx_orders_coupon_code ON orders(coupon_code) WHERE coupon_code IS NOT NULL;
