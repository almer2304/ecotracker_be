-- ============================================================
-- EcoTracker Database Schema
-- Run this in Supabase SQL Editor to create all tables
-- ============================================================

-- Enable UUID extension (usually already enabled in Supabase)
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ─── profiles ────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS profiles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    email           TEXT UNIQUE NOT NULL,
    phone           TEXT,
    role            TEXT NOT NULL CHECK (role IN ('user', 'collector')),
    total_points    INT4 NOT NULL DEFAULT 0,
    address_default TEXT,
    avatar_url      TEXT,
    password_hash   TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Auto-update updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_profiles_updated_at
    BEFORE UPDATE ON profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ─── waste_categories ─────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS waste_categories (
    id            INT4 PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name          TEXT NOT NULL,
    points_per_kg INT4 NOT NULL DEFAULT 10,
    unit          TEXT NOT NULL DEFAULT 'kg',
    icon_url      TEXT
);

-- ─── pickups ──────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS pickups (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    collector_id UUID REFERENCES profiles(id) ON DELETE SET NULL,
    status       TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'taken', 'completed', 'cancelled')),
    address      TEXT NOT NULL,
    latitude     FLOAT8 NOT NULL,
    longitude    FLOAT8 NOT NULL,
    photo_url    TEXT,
    notes        TEXT,
    completed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_pickups_user_id      ON pickups(user_id);
CREATE INDEX IF NOT EXISTS idx_pickups_collector_id ON pickups(collector_id);
CREATE INDEX IF NOT EXISTS idx_pickups_status       ON pickups(status);

-- ─── pickup_items ─────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS pickup_items (
    id              INT4 PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    pickup_id       UUID NOT NULL REFERENCES pickups(id) ON DELETE CASCADE,
    category_id     INT4 NOT NULL REFERENCES waste_categories(id),
    weight          FLOAT8 NOT NULL CHECK (weight > 0),
    subtotal_points INT4 NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_pickup_items_pickup_id ON pickup_items(pickup_id);

-- ─── point_logs ───────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS point_logs (
    id               INT4 PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id          UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    amount           INT4 NOT NULL,
    transaction_type TEXT NOT NULL CHECK (transaction_type IN ('earn', 'spend')),
    reference_id     UUID,
    description      TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_point_logs_user_id ON point_logs(user_id);

-- ─── vouchers ─────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS vouchers (
    id          INT4 PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    title       TEXT NOT NULL,
    description TEXT,
    point_cost  INT4 NOT NULL,
    stock       INT4 NOT NULL DEFAULT 0,
    image_url   TEXT,
    is_active   BOOL NOT NULL DEFAULT true
);

-- ─── user_vouchers ────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS user_vouchers (
    id         INT4 PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id    UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    voucher_id INT4 NOT NULL REFERENCES vouchers(id),
    claim_code TEXT NOT NULL UNIQUE,
    is_used    BOOL NOT NULL DEFAULT false,
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_vouchers_user_id ON user_vouchers(user_id);

-- ─── Seed Data ─────────────────────────────────────────────────────────────────
INSERT INTO waste_categories (name, points_per_kg, unit) VALUES
    ('Kertas / Kardus',    10, 'kg'),
    ('Plastik',            15, 'kg'),
    ('Logam / Besi',       20, 'kg'),
    ('Elektronik (E-Waste)', 50, 'kg'),
    ('Kaca / Botol',       8,  'kg'),
    ('Organik',            5,  'kg')
ON CONFLICT DO NOTHING;

INSERT INTO vouchers (title, description, point_cost, stock, is_active) VALUES
    ('Voucher Belanja Rp 10.000',  'Diskon belanja Rp 10.000 di mitra kami', 100,  50, true),
    ('Voucher Belanja Rp 25.000',  'Diskon belanja Rp 25.000 di mitra kami', 250,  30, true),
    ('Pulsa Rp 10.000',            'Isi pulsa Rp 10.000 semua operator',     200,  100, true),
    ('Voucher Kopi Gratis',        'Kopi gratis di kafe pilihan',             150,  20, true)
ON CONFLICT DO NOTHING;
