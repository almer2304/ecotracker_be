-- ============================================================
-- EcoTracker PostGIS Migration
-- Adds geospatial indexing for distance-based queries
-- ============================================================

-- Enable PostGIS extension (biasanya sudah ada di Supabase)
CREATE EXTENSION IF NOT EXISTS postgis;

-- ─── Add location column to pickups ───────────────────────────────────────────
ALTER TABLE pickups 
ADD COLUMN IF NOT EXISTS location GEOGRAPHY(POINT, 4326);

-- Populate location from existing lat/lon data
UPDATE pickups 
SET location = ST_SetSRID(ST_MakePoint(longitude, latitude), 4326)
WHERE location IS NULL AND latitude IS NOT NULL AND longitude IS NOT NULL;

-- Create spatial index for fast proximity queries
CREATE INDEX IF NOT EXISTS idx_pickups_location ON pickups USING GIST(location);

-- ─── Add current_location to profiles (for collectors) ────────────────────────
ALTER TABLE profiles
ADD COLUMN IF NOT EXISTS current_location GEOGRAPHY(POINT, 4326),
ADD COLUMN IF NOT EXISTS last_location_update TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_profiles_current_location ON profiles USING GIST(current_location);

-- ─── Function: Update location from lat/lon ───────────────────────────────────
-- Trigger otomatis untuk sync location saat insert/update latitude/longitude
CREATE OR REPLACE FUNCTION sync_pickup_location()
RETURNS TRIGGER AS $$
BEGIN
    NEW.location := ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_sync_pickup_location
    BEFORE INSERT OR UPDATE OF latitude, longitude ON pickups
    FOR EACH ROW
    WHEN (NEW.latitude IS NOT NULL AND NEW.longitude IS NOT NULL)
    EXECUTE FUNCTION sync_pickup_location();

-- ─── View: Pickups with calculated distances ──────────────────────────────────
-- Contoh: Buat view untuk query cepat (opsional)
-- CREATE OR REPLACE VIEW pending_pickups_with_distance AS
-- SELECT 
--     p.*,
--     ST_Distance(p.location, collector.current_location) / 1000 AS distance_km
-- FROM pickups p
-- CROSS JOIN profiles collector
-- WHERE p.status = 'pending' 
--   AND collector.role = 'collector'
--   AND collector.current_location IS NOT NULL;

-- ─── Example Query: Find nearest pending pickups ──────────────────────────────
-- SELECT 
--     id, 
--     address,
--     ST_Distance(location::geography, ST_SetSRID(ST_MakePoint(106.8456, -6.2088), 4326)::geography) / 1000 AS distance_km
-- FROM pickups
-- WHERE status = 'pending'
-- ORDER BY location <-> ST_SetSRID(ST_MakePoint(106.8456, -6.2088), 4326)::geography
-- LIMIT 10;
