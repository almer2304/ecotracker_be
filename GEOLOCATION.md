# 🗺️ Geolocation Features - EcoTracker API

## Overview

Sistem EcoTracker mendukung **2 pendekatan** untuk sorting pickup berdasarkan jarak:

| Pendekatan | Cara Kerja | Cocok Untuk | Setup |
|------------|------------|-------------|-------|
| **Haversine (Go)** | Hitung jarak di application layer | MVP, < 10k pickups | Tidak perlu setup DB |
| **PostGIS (DB)** | Native geospatial query di DB | Production, skala besar | Perlu enable PostGIS |

---

## 📱 User Experience (Kedua Case Sama)

### Cara 1: Share Lokasi GPS (Mobile App)
```javascript
// Frontend: Ambil koordinat dari device
navigator.geolocation.getCurrentPosition((position) => {
    const { latitude, longitude } = position.coords;
    
    // Kirim ke API
    fetch('/api/v1/pickups', {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` },
        body: JSON.stringify({
            address: "Jl. Sudirman No. 5, Jakarta",
            latitude: latitude,
            longitude: longitude,
            notes: "Sampah di depan pagar"
        })
    });
});
```

### Cara 2: Ketik Alamat (Web/Mobile Fallback)
User hanya ketik alamat, backend bisa pakai geocoding API (Google Maps, OpenCage, dll):

```javascript
// Frontend: User ketik alamat
const address = "Jl. Sudirman No. 5, Jakarta";

// Backend nanti geocode ke koordinat via API external
// (implementasi geocoding tidak termasuk dalam tutorial ini)
```

---

## 🔧 Implementasi 1: Haversine (Pure Go)

### Setup (Tidak Perlu Apa-apa!)
Langsung bisa dipakai tanpa setup tambahan.

### Endpoint Collector

**GET** `/api/v1/collector/pickups/pending?lat=-6.2088&lon=106.8456`

**Headers:**
```
Authorization: Bearer <collector_token>
```

**Query Parameters:**
| Param | Type | Required | Keterangan |
|-------|------|----------|------------|
| `lat` | float | Yes | Latitude lokasi collector saat ini |
| `lon` | float | Yes | Longitude lokasi collector saat ini |

**Response (200):**
```json
{
  "success": true,
  "message": "Nearby pending pickups retrieved (sorted by distance)",
  "data": [
    {
      "id": "uuid-1",
      "user_id": "uuid-user-1",
      "address": "Jl. Sudirman No. 5, Jakarta",
      "latitude": -6.2088,
      "longitude": 106.8456,
      "photo_url": "https://...",
      "status": "pending",
      "distance_km": 1.2,
      "created_at": "2026-02-17T10:00:00Z"
    },
    {
      "id": "uuid-2",
      "address": "Jl. Thamrin No. 10",
      "latitude": -6.1944,
      "longitude": 106.8229,
      "distance_km": 3.5,
      "created_at": "2026-02-17T09:30:00Z"
    }
  ]
}
```

### Cara Kerja (Haversine)
```
1. Frontend collector kirim lokasi GPS saat ini via query param
2. Backend fetch semua pending pickups dari DB
3. Loop semua pickup, hitung jarak pakai Haversine formula
4. Sort array berdasarkan distance_km (ascending)
5. Return ke frontend
```

**Kelebihan:**
- ✅ Simple, tidak perlu setup DB
- ✅ Akurat untuk jarak < 100km
- ✅ Mudah di-debug

**Kekurangan:**
- ❌ Semua pickup di-load ke memory dulu
- ❌ Sorting di application layer (lambat jika > 10k pickups)

---

## 🗄️ Implementasi 2: PostGIS (Production)

### Setup PostGIS

#### 1. Jalankan Migration di Supabase
```sql
-- File: migrations/002_postgis.sql
CREATE EXTENSION IF NOT EXISTS postgis;

ALTER TABLE pickups 
ADD COLUMN IF NOT EXISTS location GEOGRAPHY(POINT, 4326);

UPDATE pickups 
SET location = ST_SetSRID(ST_MakePoint(longitude, latitude), 4326);

CREATE INDEX idx_pickups_location ON pickups USING GIST(location);
```

#### 2. Trigger Auto-Sync Location
Setiap kali insert/update `latitude`/`longitude`, kolom `location` auto-update:
```sql
CREATE OR REPLACE FUNCTION sync_pickup_location()
RETURNS TRIGGER AS $$
BEGIN
    NEW.location := ST_SetSRID(ST_MakePoint(NEW.longitude, NEW.latitude), 4326);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_sync_pickup_location
    BEFORE INSERT OR UPDATE OF latitude, longitude ON pickups
    FOR EACH ROW EXECUTE FUNCTION sync_pickup_location();
```

### Endpoint Collector (PostGIS)

**GET** `/api/v1/collector/pickups/pending?lat=-6.2088&lon=106.8456&use_postgis=true`

**Query Parameters:**
| Param | Type | Required | Keterangan |
|-------|------|----------|------------|
| `lat` | float | Yes | Latitude lokasi collector |
| `lon` | float | Yes | Longitude lokasi collector |
| `use_postgis` | bool | Optional | Set `true` untuk pakai PostGIS |
| `limit` | int | Optional | Max hasil (default 50) |

**Response:** _(sama seperti Haversine)_

### Cara Kerja (PostGIS)
```
1. Frontend collector kirim lokasi GPS via query param + use_postgis=true
2. Backend eksekusi native PostGIS query:
   SELECT *, ST_Distance(location, collector_point) AS distance_km
   FROM pickups
   WHERE status = 'pending'
   ORDER BY location <-> collector_point
   LIMIT 50
3. Database langsung return hasil ter-sort
4. Backend forward ke frontend
```

**Kelebihan:**
- ✅ Query di database (super cepat)
- ✅ Index GIST untuk performa tinggi
- ✅ Skala untuk jutaan koordinat
- ✅ Fitur advanced: radius search, polygon, routing

**Kekurangan:**
- ❌ Perlu setup PostGIS extension
- ❌ Lebih kompleks untuk di-debug

---

## 🎯 Handler Update untuk Switch Mode

Anda bisa modifikasi `pickup_handler.go` untuk support keduanya:

```go
func (h *PickupHandler) GetPendingPickups(c *gin.Context) {
	latStr := c.Query("lat")
	lonStr := c.Query("lon")
	usePostGIS := c.Query("use_postgis") == "true"
	limitStr := c.DefaultQuery("limit", "50")

	if latStr == "" || lonStr == "" {
		// No location: return all without sorting
		pickups, err := h.pickupService.ListPendingPickups(c.Request.Context())
		if err != nil {
			utils.RespondWithDomainError(c, err)
			return
		}
		utils.RespondSuccess(c, http.StatusOK, "Pending pickups", pickups)
		return
	}

	lat, _ := strconv.ParseFloat(latStr, 64)
	lon, _ := strconv.ParseFloat(lonStr, 64)
	limit, _ := strconv.Atoi(limitStr)

	// Choose implementation
	var pickups []domain.PickupWithDistance
	var err error

	if usePostGIS {
		pickups, err = h.pickupService.ListPendingPickupsNearbyPostGIS(
			c.Request.Context(), lat, lon, limit)
	} else {
		pickups, err = h.pickupService.ListPendingPickupsNearby(
			c.Request.Context(), lat, lon)
	}

	if err != nil {
		utils.RespondWithDomainError(c, err)
		return
	}
	utils.RespondSuccess(c, http.StatusOK, "Nearby pickups (sorted)", pickups)
}
```

---

## 📊 Perbandingan Performa

| Metric | Haversine (Go) | PostGIS (DB) |
|--------|----------------|--------------|
| Setup Time | 0 menit | 5 menit (enable extension) |
| Query 100 pickups | ~10ms | ~2ms |
| Query 10,000 pickups | ~500ms | ~15ms |
| Query 1,000,000 pickups | ⚠️ OOM | ~50ms |
| Scalability | Limited | Excellent |

---

## 🧪 Testing

### Test Haversine
```bash
# Collector di Jakarta Pusat (-6.2088, 106.8456)
curl -H "Authorization: Bearer <token>" \
  "http://localhost:8080/api/v1/collector/pickups/pending?lat=-6.2088&lon=106.8456"
```

### Test PostGIS
```bash
# Setelah jalankan migrations/002_postgis.sql
curl -H "Authorization: Bearer <token>" \
  "http://localhost:8080/api/v1/collector/pickups/pending?lat=-6.2088&lon=106.8456&use_postgis=true&limit=10"
```

---

## 🔄 Migration Path

**Start:** Haversine (MVP)  
↓ (Saat traffic meningkat)  
**Upgrade:** PostGIS (Production)

**Zero Downtime Migration:**
1. Jalankan `002_postgis.sql` di database
2. Update code untuk support `use_postgis=true`
3. Test PostGIS di staging
4. Flip environment variable di production
5. Remove Haversine code setelah stabil

---

## 💡 Tips & Best Practices

### Frontend Mobile App
```javascript
// Update lokasi collector setiap 30 detik
setInterval(() => {
    navigator.geolocation.getCurrentPosition((pos) => {
        fetchNearbyPickups(pos.coords.latitude, pos.coords.longitude);
    });
}, 30000);
```

### Backend Caching
```go
// Cache hasil query untuk 1 menit (opsional)
type CachedPickups struct {
    Data      []domain.PickupWithDistance
    Timestamp time.Time
}

var cache = make(map[string]CachedPickups)
```

### Rate Limiting
Limit request GPS-based query untuk prevent abuse:
```go
// Max 10 requests per minute per collector
middleware.RateLimit(10, time.Minute)
```

---

## ❓ FAQ

**Q: User harus input GPS manual?**  
A: Tidak. Mobile app auto-detect GPS via `navigator.geolocation`. User cuma klik "Request Pickup".

**Q: Bagaimana jika user tidak share lokasi?**  
A: Fallback ke input alamat manual, lalu backend geocode ke koordinat (perlu API key Google Maps/OpenCage).

**Q: Apakah collector harus update lokasi terus-menerus?**  
A: Tidak wajib. Collector cukup share lokasi saat buka dashboard pending pickups.

**Q: Bisakah pakai Google Maps Distance Matrix API?**  
A: Bisa, tapi berbayar dan ada quota limit. Haversine/PostGIS lebih ekonomis untuk skala besar.
