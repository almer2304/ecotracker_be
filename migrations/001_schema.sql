-- ============================================================
-- EcoTracker V2.0 - Complete Database Schema
-- PostgreSQL with PostGIS extension
-- ============================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "postgis";

-- ============================================================
-- ENUM TYPES
-- ============================================================
CREATE TYPE user_role AS ENUM ('user', 'collector', 'admin');
CREATE TYPE pickup_status AS ENUM (
    'pending', 'assigned', 'reassigned', 'accepted',
    'in_progress', 'arrived', 'completed', 'cancelled'
);
CREATE TYPE report_severity AS ENUM ('low', 'medium', 'high');
CREATE TYPE report_status AS ENUM (
    'new', 'investigating', 'assigned', 'in_progress', 'resolved'
);
CREATE TYPE feedback_type AS ENUM ('app', 'collector', 'general');
CREATE TYPE point_log_type AS ENUM ('earned', 'spent', 'adjustment');

-- ============================================================
-- TABLE: profiles
-- ============================================================
CREATE TABLE profiles (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(100) NOT NULL,
    email           VARCHAR(150) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    phone           VARCHAR(20),
    role            user_role NOT NULL DEFAULT 'user',
    avatar_url      TEXT,

    -- Points (for users)
    total_points            INTEGER NOT NULL DEFAULT 0,
    total_pickups_completed INTEGER NOT NULL DEFAULT 0,
    total_reports_submitted INTEGER NOT NULL DEFAULT 0,

    -- Collector-specific fields
    is_online       BOOLEAN NOT NULL DEFAULT false,
    is_busy         BOOLEAN NOT NULL DEFAULT false,
    average_rating  NUMERIC(3,2) NOT NULL DEFAULT 0.00,
    total_ratings   INTEGER NOT NULL DEFAULT 0,
    total_weight_collected NUMERIC(10,2) NOT NULL DEFAULT 0.00,

    -- Geolocation (collector's last known position)
    last_lat        DOUBLE PRECISION,
    last_lon        DOUBLE PRECISION,
    last_location_updated_at TIMESTAMPTZ,

    -- Refresh token
    refresh_token   TEXT,
    refresh_token_expires_at TIMESTAMPTZ,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- ============================================================
-- TABLE: waste_categories
-- ============================================================
CREATE TABLE waste_categories (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name            VARCHAR(50) NOT NULL UNIQUE,
    description     TEXT,
    points_per_kg   INTEGER NOT NULL DEFAULT 10,
    icon_url        TEXT,
    color_hex       VARCHAR(7),
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- TABLE: pickups
-- ============================================================
CREATE TABLE pickups (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id             UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    collector_id        UUID REFERENCES profiles(id) ON DELETE SET NULL,

    -- Location
    address             TEXT NOT NULL,
    lat                 DOUBLE PRECISION NOT NULL,
    lon                 DOUBLE PRECISION NOT NULL,

    -- Content
    photo_url           TEXT,
    notes               TEXT,
    status              pickup_status NOT NULL DEFAULT 'pending',

    -- Assignment tracking
    assigned_at         TIMESTAMPTZ,
    assignment_timeout  TIMESTAMPTZ,
    reassignment_count  INTEGER NOT NULL DEFAULT 0,

    -- Progress timestamps
    accepted_at         TIMESTAMPTZ,
    started_at          TIMESTAMPTZ,
    arrived_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,

    -- Results
    total_weight        NUMERIC(10,2),
    total_points_awarded INTEGER,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

-- ============================================================
-- TABLE: pickup_items
-- ============================================================
CREATE TABLE pickup_items (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pickup_id       UUID NOT NULL REFERENCES pickups(id) ON DELETE CASCADE,
    category_id     UUID NOT NULL REFERENCES waste_categories(id),
    weight_kg       NUMERIC(8,2) NOT NULL,
    points_awarded  INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- TABLE: assignment_history
-- ============================================================
CREATE TABLE assignment_history (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pickup_id       UUID NOT NULL REFERENCES pickups(id) ON DELETE CASCADE,
    collector_id    UUID NOT NULL REFERENCES profiles(id),
    assigned_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    timeout_at      TIMESTAMPTZ,
    released_at     TIMESTAMPTZ,
    release_reason  VARCHAR(50), -- 'timeout', 'cancelled', 'completed'
    distance_km     NUMERIC(8,2)
);

-- ============================================================
-- TABLE: badges
-- ============================================================
CREATE TABLE badges (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code            VARCHAR(50) NOT NULL UNIQUE,
    name            VARCHAR(100) NOT NULL,
    description     TEXT,
    icon_url        TEXT,
    color_hex       VARCHAR(7),

    -- Criteria
    criteria_type   VARCHAR(50) NOT NULL, -- 'pickups', 'points', 'reports'
    criteria_value  INTEGER NOT NULL,

    display_order   INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- TABLE: user_badges
-- ============================================================
CREATE TABLE user_badges (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    badge_id    UUID NOT NULL REFERENCES badges(id),
    awarded_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, badge_id)
);

-- ============================================================
-- TABLE: point_logs
-- ============================================================
CREATE TABLE point_logs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    pickup_id       UUID REFERENCES pickups(id) ON DELETE SET NULL,
    log_type        point_log_type NOT NULL DEFAULT 'earned',
    points          INTEGER NOT NULL,
    description     TEXT,
    balance_after   INTEGER NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- TABLE: area_reports
-- ============================================================
CREATE TABLE area_reports (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    reporter_id     UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    assigned_to     UUID REFERENCES profiles(id) ON DELETE SET NULL,

    title           VARCHAR(200) NOT NULL,
    description     TEXT NOT NULL,
    address         TEXT NOT NULL,
    lat             DOUBLE PRECISION NOT NULL,
    lon             DOUBLE PRECISION NOT NULL,
    severity        report_severity NOT NULL DEFAULT 'medium',
    status          report_status NOT NULL DEFAULT 'new',

    photo_urls      TEXT[], -- Array of photo URLs

    admin_notes     TEXT,
    resolved_at     TIMESTAMPTZ,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- ============================================================
-- TABLE: feedback
-- ============================================================
CREATE TABLE feedback (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    pickup_id       UUID REFERENCES pickups(id) ON DELETE SET NULL,
    collector_id    UUID REFERENCES profiles(id) ON DELETE SET NULL,

    feedback_type   feedback_type NOT NULL DEFAULT 'general',
    rating          SMALLINT CHECK (rating >= 1 AND rating <= 5),
    title           VARCHAR(200),
    comment         TEXT,
    tags            TEXT[],

    admin_response  TEXT,
    responded_at    TIMESTAMPTZ,
    responded_by    UUID REFERENCES profiles(id) ON DELETE SET NULL,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- INDEXES
-- ============================================================

-- profiles
CREATE INDEX idx_profiles_email ON profiles(email);
CREATE INDEX idx_profiles_role ON profiles(role);
CREATE INDEX idx_profiles_is_online ON profiles(is_online) WHERE deleted_at IS NULL;
CREATE INDEX idx_profiles_collector_available ON profiles(is_online, is_busy, role)
    WHERE deleted_at IS NULL AND role = 'collector';
CREATE INDEX idx_profiles_location ON profiles(last_lat, last_lon)
    WHERE last_lat IS NOT NULL AND last_lon IS NOT NULL;

-- pickups
CREATE INDEX idx_pickups_user_id ON pickups(user_id);
CREATE INDEX idx_pickups_collector_id ON pickups(collector_id);
CREATE INDEX idx_pickups_status ON pickups(status);
CREATE INDEX idx_pickups_timeout ON pickups(assignment_timeout)
    WHERE status = 'assigned';
CREATE INDEX idx_pickups_created_at ON pickups(created_at DESC);

-- pickup_items
CREATE INDEX idx_pickup_items_pickup_id ON pickup_items(pickup_id);
CREATE INDEX idx_pickup_items_category_id ON pickup_items(category_id);

-- assignment_history
CREATE INDEX idx_assignment_history_pickup_id ON assignment_history(pickup_id);
CREATE INDEX idx_assignment_history_collector_id ON assignment_history(collector_id);

-- user_badges
CREATE INDEX idx_user_badges_user_id ON user_badges(user_id);

-- point_logs
CREATE INDEX idx_point_logs_user_id ON point_logs(user_id);
CREATE INDEX idx_point_logs_pickup_id ON point_logs(pickup_id);
CREATE INDEX idx_point_logs_created_at ON point_logs(created_at DESC);

-- area_reports
CREATE INDEX idx_area_reports_reporter_id ON area_reports(reporter_id);
CREATE INDEX idx_area_reports_status ON area_reports(status);
CREATE INDEX idx_area_reports_severity ON area_reports(severity);

-- feedback
CREATE INDEX idx_feedback_user_id ON feedback(user_id);
CREATE INDEX idx_feedback_collector_id ON feedback(collector_id);
CREATE INDEX idx_feedback_pickup_id ON feedback(pickup_id);

-- ============================================================
-- TRIGGERS
-- ============================================================

-- Auto-update updated_at
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_profiles_updated_at
    BEFORE UPDATE ON profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_pickups_updated_at
    BEFORE UPDATE ON pickups
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_area_reports_updated_at
    BEFORE UPDATE ON area_reports
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_feedback_updated_at
    BEFORE UPDATE ON feedback
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Auto-update collector average rating after feedback insert
CREATE OR REPLACE FUNCTION update_collector_rating()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.collector_id IS NOT NULL AND NEW.rating IS NOT NULL THEN
        UPDATE profiles
        SET
            average_rating = (
                SELECT ROUND(AVG(rating)::NUMERIC, 2)
                FROM feedback
                WHERE collector_id = NEW.collector_id AND rating IS NOT NULL
            ),
            total_ratings = (
                SELECT COUNT(*) FROM feedback
                WHERE collector_id = NEW.collector_id AND rating IS NOT NULL
            )
        WHERE id = NEW.collector_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_collector_rating
    AFTER INSERT ON feedback
    FOR EACH ROW EXECUTE FUNCTION update_collector_rating();

-- ============================================================
-- VIEWS
-- ============================================================

-- Available collectors (online, not busy, location known)
CREATE VIEW v_available_collectors AS
SELECT
    id, name, email, phone, avatar_url,
    last_lat, last_lon, last_location_updated_at,
    average_rating, total_ratings
FROM profiles
WHERE
    role = 'collector'
    AND is_online = true
    AND is_busy = false
    AND deleted_at IS NULL
    AND last_lat IS NOT NULL
    AND last_lon IS NOT NULL
    AND last_location_updated_at > NOW() - INTERVAL '30 minutes';

-- Pending pickups (unassigned)
CREATE VIEW v_pending_pickups AS
SELECT p.*, u.name AS user_name, u.phone AS user_phone
FROM pickups p
JOIN profiles u ON u.id = p.user_id
WHERE p.status = 'pending' AND p.deleted_at IS NULL;

-- Expired assignments (timeout passed)
CREATE VIEW v_expired_assignments AS
SELECT
    p.id,
    p.collector_id,
    p.lat,
    p.lon,
    p.assignment_timeout,
    p.reassignment_count,
    c.name AS collector_name
FROM pickups p
LEFT JOIN profiles c ON c.id = p.collector_id
WHERE
    p.status = 'assigned'
    AND p.assignment_timeout < NOW()
    AND p.deleted_at IS NULL;

-- Admin dashboard stats
CREATE VIEW v_dashboard_stats AS
SELECT
    (SELECT COUNT(*) FROM profiles WHERE role = 'user' AND deleted_at IS NULL) AS total_users,
    (SELECT COUNT(*) FROM profiles WHERE role = 'collector' AND deleted_at IS NULL) AS total_collectors,
    (SELECT COUNT(*) FROM profiles WHERE role = 'collector' AND is_online = true AND deleted_at IS NULL) AS online_collectors,
    (SELECT COUNT(*) FROM pickups WHERE deleted_at IS NULL) AS total_pickups,
    (SELECT COUNT(*) FROM pickups WHERE status = 'pending' AND deleted_at IS NULL) AS pending_pickups,
    (SELECT COUNT(*) FROM pickups WHERE status = 'completed' AND deleted_at IS NULL) AS completed_pickups,
    (SELECT COALESCE(SUM(total_weight), 0) FROM pickups WHERE status = 'completed') AS total_weight_kg,
    (SELECT COUNT(*) FROM area_reports WHERE deleted_at IS NULL) AS total_reports,
    (SELECT COUNT(*) FROM area_reports WHERE status = 'new' AND deleted_at IS NULL) AS new_reports;

-- ============================================================
-- SEED DATA: waste_categories
-- ============================================================
INSERT INTO waste_categories (name, description, points_per_kg, color_hex) VALUES
    ('Plastik',  'Botol plastik, kantong, kemasan plastik', 15, '#3498DB'),
    ('Kertas',   'Koran, kardus, buku bekas, kertas HVS',   10, '#F39C12'),
    ('Logam',    'Kaleng, besi tua, aluminium foil',         20, '#95A5A6'),
    ('Kaca',     'Botol kaca, cermin pecah, guci',           12, '#1ABC9C'),
    ('Organik',  'Sisa makanan, daun kering, sayuran busuk',  5, '#27AE60');

-- ============================================================
-- SEED DATA: badges
-- ============================================================
INSERT INTO badges (code, name, description, criteria_type, criteria_value, display_order, color_hex) VALUES
    ('first_pickup',       'Pickup Pertama',       'Selesaikan pickup pertamamu',                     'pickups', 1,   1, '#F1C40F'),
    ('eco_warrior',        'Eco Warrior',          'Selesaikan 10 pickup',                            'pickups', 10,  2, '#2ECC71'),
    ('eco_champion',       'Eco Champion',         'Selesaikan 50 pickup',                            'pickups', 50,  3, '#27AE60'),
    ('eco_legend',         'Eco Legend',           'Selesaikan 100 pickup',                           'pickups', 100, 4, '#1E8449'),
    ('point_master',       'Point Master',         'Kumpulkan 1.000 poin',                            'points',  1000, 5, '#9B59B6'),
    ('point_legend',       'Point Legend',         'Kumpulkan 5.000 poin',                            'points',  5000, 6, '#6C3483'),
    ('reporter_hero',      'Reporter Hero',        'Laporkan 5 area kotor',                           'reports', 5,   7, '#E74C3C'),
    ('community_guardian', 'Community Guardian',   'Laporkan 20 area kotor demi lingkungan bersih',   'reports', 20,  8, '#C0392B');
