-- Migration: 001_migrate_to_uuid
-- Purpose: Migrate all tables from uint/serial IDs to UUIDv7 primary keys
-- Date: 2026-04-02
-- Note: This is a destructive migration — all data will be lost.
-- Run BEFORE GORM AutoMigrate.

-- Enable UUID extension (provides gen_random_uuid in PG13+)
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================
-- DROP TABLES in reverse FK dependency order
-- ============================================================
DROP TABLE IF EXISTS refresh_tokens CASCADE;
DROP TABLE IF EXISTS payment_links CASCADE;
DROP TABLE IF EXISTS payments CASCADE;
DROP TABLE IF EXISTS order_items CASCADE;
DROP TABLE IF EXISTS orders CASCADE;
DROP TABLE IF EXISTS inventories CASCADE;
DROP TABLE IF EXISTS product_images CASCADE;
DROP TABLE IF EXISTS product_variant_attributes CASCADE;
DROP TABLE IF EXISTS product_variants CASCADE;
DROP TABLE IF EXISTS products CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS categories CASCADE;

-- ============================================================
-- CREATE TABLES with UUID primary keys (FK dependency order)
-- ============================================================

-- 1. categories (no FK dependencies)
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    parent_id UUID,
    level INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
CREATE INDEX idx_categories_parent_id ON categories(parent_id);
CREATE INDEX idx_categories_level ON categories(level);
CREATE INDEX idx_categories_deleted_at ON categories(deleted_at);
ALTER TABLE categories ADD CONSTRAINT fk_categories_parent
    FOREIGN KEY (parent_id) REFERENCES categories(id) ON DELETE SET NULL;

-- 2. users (no FK dependencies)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE,
    password VARCHAR(255),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    phone VARCHAR(20),
    role VARCHAR(50) DEFAULT 'customer',
    active BOOLEAN DEFAULT true,
    email_verified BOOLEAN DEFAULT false,
    verification_token VARCHAR(64),
    verification_expires TIMESTAMP,
    reset_token VARCHAR(64),
    reset_expires TIMESTAMP,
    two_fa_secret VARCHAR(255),
    two_fa_enabled BOOLEAN DEFAULT false,
    two_fa_backup_codes TEXT,
    oauth_provider VARCHAR(50),
    oauth_provider_id VARCHAR(255),
    avatar_url VARCHAR(500),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
CREATE INDEX idx_users_verification_expires ON users(verification_expires);
CREATE INDEX idx_users_reset_expires ON users(reset_expires);
CREATE INDEX idx_users_deleted_at ON users(deleted_at);

-- 3. products (FK: category_id → categories)
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    brand VARCHAR(100),
    description TEXT,
    base_price DECIMAL(12,2) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_products_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT
);
CREATE INDEX idx_products_category_id ON products(category_id);

-- 4. product_variants (FK: product_id → products)
CREATE TABLE product_variants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL,
    sku VARCHAR(100) NOT NULL UNIQUE,
    price DECIMAL(12,2) NOT NULL,
    stock INTEGER DEFAULT 0,
    reserved INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_variants_product FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);
CREATE INDEX idx_product_variants_product_id ON product_variants(product_id);

-- 5. product_variant_attributes (FK: variant_id → product_variants)
CREATE TABLE product_variant_attributes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    variant_id UUID NOT NULL UNIQUE,
    color VARCHAR(50) NOT NULL,
    size VARCHAR(20) NOT NULL,
    weight VARCHAR(50) NOT NULL,
    CONSTRAINT fk_variant_attrs_variant FOREIGN KEY (variant_id) REFERENCES product_variants(id) ON DELETE CASCADE
);

-- 6. product_images (FK: product_id → products, variant_id → product_variants)
CREATE TABLE product_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL,
    variant_id UUID,
    url_image TEXT NOT NULL,
    is_main BOOLEAN DEFAULT false,
    sort_order INTEGER DEFAULT 0,
    CONSTRAINT fk_images_product FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    CONSTRAINT fk_images_variant FOREIGN KEY (variant_id) REFERENCES product_variants(id) ON DELETE SET NULL
);
CREATE INDEX idx_product_images_product_id ON product_images(product_id);
CREATE INDEX idx_product_images_variant_id ON product_images(variant_id);

-- 7. inventories (FK: product_id → products)
CREATE TABLE inventories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL UNIQUE,
    quantity INTEGER DEFAULT 0,
    reserved INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    CONSTRAINT fk_inventories_product FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);
CREATE INDEX idx_inventories_product_id ON inventories(product_id);
CREATE INDEX idx_inventories_deleted_at ON inventories(deleted_at);

-- 8. orders (FK: user_id → users)
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    total_price DECIMAL(10,2),
    shipping_address TEXT,
    notes TEXT,
    payment_transaction_id VARCHAR(255),
    payment_link_id VARCHAR(255),
    payment_status VARCHAR(50) DEFAULT 'pending',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    CONSTRAINT fk_orders_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
);
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_deleted_at ON orders(deleted_at);

-- 9. order_items (FK: order_id → orders, product_id → products, variant_id → product_variants)
CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL,
    product_id UUID NOT NULL,
    variant_id UUID,
    quantity INTEGER NOT NULL,
    unit_price DECIMAL(10,2),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_order_items_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    CONSTRAINT fk_order_items_product FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE RESTRICT,
    CONSTRAINT fk_order_items_variant FOREIGN KEY (variant_id) REFERENCES product_variants(id) ON DELETE SET NULL
);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_product_id ON order_items(product_id);
CREATE INDEX idx_order_items_variant_id ON order_items(variant_id);

-- 10. payments (FK: order_id → orders)
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL,
    wompi_transaction_id VARCHAR(255) UNIQUE,
    amount BIGINT NOT NULL,
    currency VARCHAR(3) DEFAULT 'COP',
    status VARCHAR(50) DEFAULT 'pending',
    payment_method VARCHAR(50),
    payment_token VARCHAR(255),
    redirect_url VARCHAR(500),
    reference VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    CONSTRAINT fk_payments_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE RESTRICT
);
CREATE INDEX idx_payments_order_id ON payments(order_id);
CREATE INDEX idx_payments_reference ON payments(reference);
CREATE INDEX idx_payments_deleted_at ON payments(deleted_at);

-- 11. payment_links (FK: order_id → orders)
CREATE TABLE payment_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL,
    wompi_link_id VARCHAR(255) UNIQUE,
    url VARCHAR(500) NOT NULL,
    amount BIGINT NOT NULL,
    currency VARCHAR(3) DEFAULT 'COP',
    description VARCHAR(500),
    status VARCHAR(50) DEFAULT 'active',
    single_use BOOLEAN DEFAULT false,
    expires_at TIMESTAMP,
    redirect_url VARCHAR(500),
    reference VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    CONSTRAINT fk_payment_links_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE RESTRICT
);
CREATE INDEX idx_payment_links_order_id ON payment_links(order_id);
CREATE INDEX idx_payment_links_expires_at ON payment_links(expires_at);
CREATE INDEX idx_payment_links_reference ON payment_links(reference);
CREATE INDEX idx_payment_links_deleted_at ON payment_links(deleted_at);

-- 12. refresh_tokens (FK: user_id → users)
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token VARCHAR(255) NOT NULL UNIQUE,
    user_id UUID NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    revoked BOOLEAN DEFAULT false,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_refresh_tokens_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token ON refresh_tokens(token);
