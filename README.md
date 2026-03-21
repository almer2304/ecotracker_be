# 🌿 EcoTracker V2.0 — Backend API

> Platform manajemen pengambilan sampah daur ulang yang menghubungkan user dengan collector terdekat secara otomatis.

**Stack:** Go 1.21 · Gin · PostgreSQL (Supabase) · Redis (opsional) · JWT Auth

---

## 📋 Daftar Isi

- [Overview](#overview)
- [Cara Setup](#cara-setup)
- [Environment Variables](#environment-variables)
- [Struktur Project](#struktur-project)
- [API Endpoints](#api-endpoints)
  - [Auth](#auth)
  - [Categories](#categories)
  - [Pickups (User)](#pickups-user)
  - [Collector](#collector)
  - [Badges](#badges)
  - [Reports](#reports)
  - [Feedback](#feedback)
  - [Admin](#admin)
- [Sistem Poin & Badge](#sistem-poin--badge)
- [Alur Auto-Assignment](#alur-auto-assignment)
- [Error Codes](#error-codes)

---

## Overview

### Aktor & Role

| Role | Deskripsi |
|------|-----------|
| `user` | Membuat pickup request, kumpulkan poin & badge, lapor area kotor |
| `collector` | Terima assignment pickup, update lokasi GPS, selesaikan pickup |
| `admin` | Kelola collector, pantau semua aktivitas, balas feedback |

### Autentikasi

Semua endpoint (kecuali **Public**) membutuhkan JWT Bearer Token:

```
Authorization: Bearer <access_token>
```

| Token | Durasi | Keterangan |
|-------|--------|------------|
| Access Token | 15 menit | Untuk akses endpoint |
| Refresh Token | 7 hari | Untuk generate access token baru |

---

## Cara Setup

### 1. Clone & Install Dependencies
```bash
git clone <repo-url>
cd ecotracker
go mod tidy
```

### 2. Konfigurasi Environment
```bash
cp .env.example .env
# Edit .env dan isi semua nilai yang diperlukan
```

### 3. Setup Database
Buka **Supabase Dashboard → SQL Editor**, lalu jalankan berurutan:
```
migrations/001_schema.sql   ← buat semua tabel, index, view, seed data
```

### 4. Buat Bucket Supabase Storage
Buat 3 bucket di **Supabase Dashboard → Storage** (set sebagai Public):
- `waste-photos`
- `report-photos`
- `avatars`

### 5. Jalankan Server
```bash
go run cmd/server/main.go
```

### 6. Verifikasi
```bash
curl http://localhost:8080/health
# Response: {"status":"ok","service":"EcoTracker V2.0"}
```

---

## Environment Variables

```env
# Server
APP_ENV=development
APP_PORT=8080

# Database (Supabase)
DB_HOST=db.xxx.supabase.co
DB_PORT=6543
DB_USER=postgres.xxx
DB_PASSWORD=your_password
DB_NAME=postgres
DB_SSL_MODE=require
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5

# JWT
JWT_SECRET=min-32-karakter-secret-key-disini
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h

# Supabase Storage
SUPABASE_URL=https://xxx.supabase.co
SUPABASE_KEY=eyJhbGci...service_role_key
SUPABASE_BUCKET_PICKUPS=waste-photos
SUPABASE_BUCKET_REPORTS=report-photos
SUPABASE_BUCKET_AVATARS=avatars

# Admin Secret (untuk buat akun admin/collector via endpoint)
ADMIN_SECRET=ecotracker-admin-secret-2026

# Worker
ASSIGNMENT_TIMEOUT=15m
TIMEOUT_CHECK_INTERVAL=60s

# Bcrypt
BCRYPT_COST=12

# Redis (opsional, untuk rate limiting)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
```

---

## Struktur Project

```
ecotracker/
├── cmd/server/
│   └── main.go                  ← Entry point, routing, DI
├── internal/
│   ├── config/
│   │   ├── config.go            ← Load env vars
│   │   ├── database.go          ← Connection pool
│   │   └── redis.go             ← Redis client
│   ├── domain/
│   │   ├── models.go            ← Semua struct & DTOs
│   │   └── errors.go            ← Domain errors
│   ├── handler/                 ← HTTP layer (Gin handlers)
│   ├── service/                 ← Business logic
│   ├── repository/              ← Database access
│   ├── middleware/              ← Auth, RateLimit, CORS, Logger
│   ├── worker/                  ← Background jobs
│   └── utils/                   ← JWT, bcrypt, Haversine, storage
├── migrations/
│   └── 001_schema.sql           ← Schema lengkap PostgreSQL
├── .env.example
└── go.mod
```

---

## API Endpoints

**Base URL:** `http://localhost:8080/api/v1`

### Ringkasan

| Method | Endpoint | Akses |
|--------|----------|-------|
| POST | `/auth/register` | Public |
| POST | `/auth/login` | Public |
| POST | `/auth/refresh` | Public |
| POST | `/auth/register-admin` | Secret Key |
| POST | `/auth/register-collector` | Secret Key |
| GET | `/auth/profile` | All Auth |
| GET | `/categories` | Public |
| POST | `/pickups` | User |
| GET | `/pickups/my` | User |
| GET | `/pickups/:id` | User/Collector |
| PUT | `/collector/status` | Collector |
| PUT | `/collector/location` | Collector |
| GET | `/collector/assigned` | Collector |
| POST | `/collector/pickups/:id/accept` | Collector |
| POST | `/collector/pickups/:id/start` | Collector |
| POST | `/collector/pickups/:id/arrive` | Collector |
| POST | `/collector/pickups/:id/complete` | Collector |
| GET | `/badges` | All Auth |
| GET | `/badges/my` | User |
| POST | `/reports` | User |
| GET | `/reports/my` | User |
| GET | `/reports/:id` | User/Admin |
| POST | `/feedback` | User |
| GET | `/feedback/my` | User |
| GET | `/admin/dashboard` | Admin |
| GET | `/admin/collectors` | Admin |
| POST | `/admin/collectors` | Admin |
| DELETE | `/admin/collectors/:id` | Admin |
| GET | `/admin/pickups` | Admin |
| GET | `/admin/reports` | Admin |
| PUT | `/admin/reports/:id` | Admin |
| GET | `/admin/feedback` | Admin |
| PUT | `/admin/feedback/:id/respond` | Admin |

---

### Auth

#### `POST /auth/register`
Daftar akun user baru. Role otomatis `user`.

**Request:**
```json
{
  "name": "Budi Santoso",
  "email": "budi@example.com",
  "password": "password123",
  "phone": "08123456789"
}
```

**Response `201`:**
```json
{
  "success": true,
  "message": "Registrasi berhasil",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 900,
    "user": {
      "id": "uuid",
      "name": "Budi Santoso",
      "email": "budi@example.com",
      "role": "user",
      "total_points": 0,
      "total_pickups_completed": 0
    }
  }
}
```

---

#### `POST /auth/login`
Login untuk semua role.

**Request:**
```json
{
  "email": "budi@example.com",
  "password": "password123"
}
```

**Response `200`:** _(sama seperti register)_

---

#### `POST /auth/refresh`
Generate access token baru menggunakan refresh token.

**Request:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response `200`:** _(sama seperti login)_

---

#### `POST /auth/register-admin`
Buat akun admin. Membutuhkan header secret key.

**Header:**
```
X-Admin-Secret: ecotracker-admin-secret-2026
```

**Request:**
```json
{
  "name": "Super Admin",
  "email": "admin@ecotracker.com",
  "password": "Admin@2026",
  "phone": "08100000000"
}
```

**Response `201`:** _(sama seperti register, role = `admin`)_

---

#### `POST /auth/register-collector`
Buat akun collector. Membutuhkan header secret key.

**Header:**
```
X-Admin-Secret: ecotracker-admin-secret-2026
```

**Request:**
```json
{
  "name": "Collector Test",
  "email": "collector@ecotracker.com",
  "password": "Collector@2026",
  "phone": "08111111111"
}
```

**Response `201`:** _(sama seperti register, role = `collector`)_

---

#### `GET /auth/profile`
Ambil profil user yang sedang login.

**Response `200`:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "Budi Santoso",
    "email": "budi@example.com",
    "role": "user",
    "total_points": 150,
    "total_pickups_completed": 5,
    "total_reports_submitted": 2,
    "is_online": false,
    "average_rating": 0
  }
}
```

---

### Categories

#### `GET /categories`
Ambil semua kategori sampah. **Tidak perlu token.**

**Response `200`:**
```json
{
  "success": true,
  "data": [
    { "id": "uuid", "name": "Plastik", "points_per_kg": 15, "color_hex": "#3498DB" },
    { "id": "uuid", "name": "Kertas",  "points_per_kg": 10, "color_hex": "#F39C12" },
    { "id": "uuid", "name": "Logam",   "points_per_kg": 20, "color_hex": "#95A5A6" },
    { "id": "uuid", "name": "Kaca",    "points_per_kg": 12, "color_hex": "#1ABC9C" },
    { "id": "uuid", "name": "Organik", "points_per_kg": 5,  "color_hex": "#27AE60" }
  ]
}
```

---

### Pickups (User)

#### `POST /pickups`
Buat pickup request baru. Setelah dibuat, sistem otomatis mencari collector terdekat.

> ⚠️ Gunakan `Content-Type: multipart/form-data`

**Form Data:**

| Field | Type | Wajib | Keterangan |
|-------|------|-------|------------|
| `address` | Text | ✅ | Alamat lengkap |
| `lat` | Text | ✅ | Latitude (contoh: `-6.2088`) |
| `lon` | Text | ✅ | Longitude (contoh: `106.8456`) |
| `notes` | Text | ❌ | Catatan untuk collector |
| `photo` | File | ❌ | Foto sampah (jpg/png/webp, max 5MB) |

**Response `201`:**
```json
{
  "success": true,
  "message": "Pickup berhasil dibuat, mencari collector terdekat...",
  "data": {
    "id": "a481bd18-d6a1-4a2b-8e7a-99b6ef7ec4bc",
    "user_id": "user-uuid",
    "address": "Jl. Sudirman No. 5, Jakarta Pusat",
    "lat": -6.2088,
    "lon": 106.8456,
    "notes": "Sampah di depan pagar biru",
    "photo_url": "https://xxx.supabase.co/storage/v1/object/public/waste-photos/...",
    "status": "pending",
    "reassignment_count": 0,
    "created_at": "2026-03-20T09:18:11Z"
  }
}
```

---

#### `GET /pickups/my`
Riwayat pickup milik user.

**Query Params:** `?page=1&limit=20`

**Response `200`:**
```json
{
  "success": true,
  "data": {
    "data": [
      {
        "id": "uuid",
        "status": "completed",
        "address": "Jl. Sudirman No. 5",
        "total_weight": 3.5,
        "total_points_awarded": 47,
        "collector": {
          "name": "Budi Collector",
          "average_rating": 4.8
        },
        "created_at": "2026-03-20T09:18:11Z",
        "completed_at": "2026-03-20T09:45:00Z"
      }
    ],
    "total": 5,
    "page": 1,
    "limit": 20,
    "total_pages": 1
  }
}
```

---

#### `GET /pickups/:id`
Detail satu pickup. User hanya bisa akses pickup miliknya.

**Response `200`:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "status": "completed",
    "address": "Jl. Sudirman No. 5, Jakarta Pusat",
    "lat": -6.2088,
    "lon": 106.8456,
    "photo_url": "https://...",
    "notes": "Sampah di depan pagar",
    "total_weight": 3.5,
    "total_points_awarded": 47,
    "user": { "name": "Budi Santoso", "phone": "08123456789" },
    "collector": { "name": "Collector Budi", "average_rating": 4.8 },
    "items": [
      { "category": { "name": "Plastik" }, "weight_kg": 2.5, "points_awarded": 37 },
      { "category": { "name": "Kertas" },  "weight_kg": 1.0, "points_awarded": 10 }
    ],
    "assigned_at": "2026-03-20T09:19:00Z",
    "accepted_at": "2026-03-20T09:20:00Z",
    "completed_at": "2026-03-20T09:45:00Z"
  }
}
```

**Status Pickup:**

| Status | Keterangan |
|--------|------------|
| `pending` | Menunggu collector tersedia |
| `assigned` | Ditugaskan ke collector, menunggu konfirmasi (15 menit) |
| `accepted` | Collector konfirmasi, segera berangkat |
| `in_progress` | Collector dalam perjalanan |
| `arrived` | Collector tiba di lokasi |
| `completed` | Selesai, poin sudah diberikan |
| `cancelled` | Dibatalkan |

---

### Collector

> Semua endpoint collector membutuhkan token dengan role `collector`.

#### `PUT /collector/status`
Toggle status online/offline. Hanya collector **online** yang bisa menerima assignment.

**Request:**
```json
{ "is_online": true }
```

**Response `200`:**
```json
{
  "success": true,
  "message": "Status berhasil diubah menjadi online",
  "data": { "is_online": true }
}
```

---

#### `PUT /collector/location`
Update koordinat GPS collector. Wajib diupdate secara berkala.

> ⚠️ Lokasi tidak diupdate lebih dari **30 menit** → dianggap tidak tersedia untuk assignment.

**Request:**
```json
{
  "lat": -6.2100,
  "lon": 106.8400
}
```

**Response `200`:**
```json
{
  "success": true,
  "message": "Lokasi diperbarui",
  "data": { "lat": -6.21, "lon": 106.84 }
}
```

---

#### `GET /collector/assigned`
Lihat pickup yang sedang aktif ditugaskan ke collector.

**Response `200` (ada pickup):**
```json
{
  "success": true,
  "data": {
    "id": "pickup-uuid",
    "status": "assigned",
    "address": "Jl. Sudirman No. 5, Jakarta Pusat",
    "lat": -6.2088,
    "lon": 106.8456,
    "photo_url": "https://...",
    "assignment_timeout": "2026-03-20T09:33:00Z",
    "user": {
      "name": "Budi Santoso",
      "phone": "08123456789"
    }
  }
}
```

**Response `200` (tidak ada):**
```json
{ "success": true, "message": "Tidak ada pickup aktif", "data": null }
```

---

#### `POST /collector/pickups/:id/accept`
Terima pickup yang ditugaskan. Harus dilakukan sebelum timeout **15 menit**.

**Response `200`:**
```json
{
  "success": true,
  "message": "Pickup berhasil diterima",
  "data": { "id": "uuid", "status": "accepted", "accepted_at": "2026-03-20T09:20:00Z" }
}
```

---

#### `POST /collector/pickups/:id/start`
Mulai berangkat menuju lokasi user.

**Response `200`:**
```json
{
  "success": true,
  "message": "Pickup dimulai, menuju ke lokasi user",
  "data": { "status": "in_progress", "started_at": "2026-03-20T09:22:00Z" }
}
```

---

#### `POST /collector/pickups/:id/arrive`
Tandai bahwa collector sudah tiba di lokasi user.

**Response `200`:**
```json
{
  "success": true,
  "message": "Berhasil dicatat, collector telah tiba di lokasi",
  "data": { "status": "arrived", "arrived_at": "2026-03-20T09:35:00Z" }
}
```

---

#### `POST /collector/pickups/:id/complete`
Selesaikan pickup. Input detail sampah yang dikumpulkan. Poin otomatis dihitung dan diberikan ke user.

> Formula: **Poin = weight_kg × points_per_kg** (sesuai kategori)

**Request:**
```json
{
  "items": [
    {
      "category_id": "uuid-kategori-plastik",
      "weight_kg": 2.5
    },
    {
      "category_id": "uuid-kategori-kertas",
      "weight_kg": 1.0
    }
  ]
}
```

**Response `200`:**
```json
{
  "success": true,
  "message": "Pickup selesai! Poin telah diberikan ke user",
  "data": {
    "id": "pickup-uuid",
    "status": "completed",
    "total_weight": 3.5,
    "total_points_awarded": 47,
    "completed_at": "2026-03-20T09:45:00Z"
  }
}
```

> Setelah complete, sistem otomatis mengecek dan memberikan badge yang memenuhi kriteria.

---

### Badges

#### `GET /badges`
Semua definisi badge yang ada di sistem.

**Response `200`:**
```json
{
  "success": true,
  "data": [
    {
      "id": "uuid",
      "code": "first_pickup",
      "name": "Pickup Pertama",
      "description": "Selesaikan pickup pertamamu",
      "criteria_type": "pickups",
      "criteria_value": 1,
      "display_order": 1
    }
  ]
}
```

---

#### `GET /badges/my`
Badge milik user dengan status locked/unlocked.

**Response `200`:**
```json
{
  "success": true,
  "data": [
    {
      "code": "first_pickup",
      "name": "Pickup Pertama",
      "criteria_value": 1,
      "is_unlocked": true,
      "awarded_at": "2026-03-20T09:45:00Z"
    },
    {
      "code": "eco_warrior",
      "name": "Eco Warrior",
      "criteria_value": 10,
      "is_unlocked": false,
      "awarded_at": null
    }
  ]
}
```

---

### Reports

#### `POST /reports`
Laporkan area kotor dengan foto.

> ⚠️ Gunakan `Content-Type: multipart/form-data`

**Form Data:**

| Field | Type | Wajib | Keterangan |
|-------|------|-------|------------|
| `title` | Text | ✅ | Judul laporan (min 5 karakter) |
| `description` | Text | ✅ | Deskripsi detail (min 10 karakter) |
| `address` | Text | ✅ | Alamat area yang dilaporkan |
| `lat` | Text | ✅ | Latitude |
| `lon` | Text | ✅ | Longitude |
| `severity` | Text | ✅ | `low` / `medium` / `high` |
| `photos` | File[] | ❌ | Foto area (bisa multiple, max 5) |

**Response `201`:**
```json
{
  "success": true,
  "message": "Laporan berhasil dikirim",
  "data": {
    "id": "uuid",
    "title": "Tumpukan Sampah di Pinggir Jalan",
    "severity": "high",
    "status": "new",
    "photo_urls": ["https://..."],
    "created_at": "2026-03-20T10:00:00Z"
  }
}
```

**Status Laporan:**

| Status | Keterangan |
|--------|------------|
| `new` | Laporan baru masuk |
| `investigating` | Admin sedang menginvestigasi |
| `assigned` | Tim kebersihan sudah ditugaskan |
| `in_progress` | Proses pembersihan berlangsung |
| `resolved` | Selesai dibersihkan |

---

#### `GET /reports/my`
Riwayat laporan milik user. `?page=1&limit=20`

---

#### `GET /reports/:id`
Detail laporan tertentu.

---

### Feedback

#### `POST /feedback`
Kirim feedback atau rating untuk collector.

**Request:**
```json
{
  "feedback_type": "collector",
  "pickup_id": "uuid-pickup",
  "rating": 5,
  "title": "Pelayanan sangat baik",
  "comment": "Collector sangat ramah dan tepat waktu",
  "tags": ["Fast Service", "Friendly", "Professional"]
}
```

> `feedback_type`: `app` | `collector` | `general`
> Rating 1-5 wajib diisi jika `feedback_type` = `collector`

**Response `201`:**
```json
{
  "success": true,
  "message": "Feedback berhasil dikirim, terima kasih!",
  "data": { "id": "uuid", "rating": 5, "created_at": "2026-03-20T10:00:00Z" }
}
```

---

#### `GET /feedback/my`
Riwayat feedback yang sudah dikirim user. `?page=1&limit=20`

---

### Admin

> Semua endpoint admin membutuhkan token dengan role `admin`.

#### `GET /admin/dashboard`
Statistik keseluruhan sistem.

**Response `200`:**
```json
{
  "success": true,
  "data": {
    "total_users": 150,
    "total_collectors": 25,
    "online_collectors": 8,
    "total_pickups": 320,
    "pending_pickups": 5,
    "completed_pickups": 290,
    "total_weight_kg": 1250.75,
    "total_reports": 45,
    "new_reports": 3
  }
}
```

---

#### `GET /admin/collectors`
Daftar semua collector. `?page=1&limit=20`

---

#### `POST /admin/collectors`
Buat akun collector baru (tidak perlu secret key, sudah pakai token admin).

**Request:**
```json
{
  "name": "Collector Baru",
  "email": "collector.baru@ecotracker.com",
  "password": "Collector@2026",
  "phone": "08199999999"
}
```

---

#### `DELETE /admin/collectors/:id`
Soft delete akun collector.

---

#### `GET /admin/pickups`
Semua pickup dengan filter opsional.

**Query Params:** `?status=pending&page=1&limit=20`

---

#### `GET /admin/reports`
Semua laporan area kotor.

**Query Params:** `?status=new&severity=high&page=1&limit=20`

---

#### `PUT /admin/reports/:id`
Update status laporan.

**Request:**
```json
{
  "status": "investigating",
  "admin_notes": "Tim sedang menuju lokasi",
  "assigned_to": "uuid-collector"
}
```

---

#### `GET /admin/feedback`
Semua feedback. `?type=collector&page=1&limit=20`

---

#### `PUT /admin/feedback/:id/respond`
Admin membalas feedback user.

**Request:**
```json
{
  "response": "Terima kasih atas feedbacknya. Kami akan terus meningkatkan layanan."
}
```

---

## Sistem Poin & Badge

### Poin per Kategori Sampah

| Kategori | Poin per Kg |
|----------|-------------|
| 🔵 Plastik | 15 poin |
| 🟡 Kertas | 10 poin |
| ⚫ Logam | 20 poin |
| 🟢 Kaca | 12 poin |
| 🌱 Organik | 5 poin |

### Daftar Badge

| Badge | Kriteria | Target |
|-------|----------|--------|
| 🥇 Pickup Pertama | Jumlah pickup | 1 |
| 🌿 Eco Warrior | Jumlah pickup | 10 |
| 🌳 Eco Champion | Jumlah pickup | 50 |
| 🏆 Eco Legend | Jumlah pickup | 100 |
| 💎 Point Master | Total poin | 1.000 |
| 👑 Point Legend | Total poin | 5.000 |
| 📢 Reporter Hero | Jumlah laporan | 5 |
| 🛡️ Community Guardian | Jumlah laporan | 20 |

Badge diberikan **otomatis** setelah pickup atau laporan berhasil diselesaikan.

---

## Alur Auto-Assignment

```
User buat pickup
      ↓
Sistem cari collector: is_online=true, is_busy=false, lokasi < 30 menit lalu
      ↓
Hitung jarak Haversine ke setiap collector
      ↓
Assign ke collector TERDEKAT (atomik)
      ↓
Collector punya 15 menit untuk Accept
      ↓
Timeout? → Release collector lama → Cari collector berikutnya
      ↓
Maksimum 5x reassignment → Status kembali ke pending
```

**Background Worker** berjalan setiap **60 detik** mengecek assignment yang expired.

---

## Error Codes

| Code | Keterangan |
|------|------------|
| `200` | OK |
| `201` | Created |
| `400` | Bad Request — validasi input gagal |
| `401` | Unauthorized — token tidak ada / expired |
| `403` | Forbidden — role tidak punya izin |
| `404` | Not Found — data tidak ditemukan |
| `409` | Conflict — email sudah terdaftar / status konflik |
| `429` | Too Many Requests — rate limit (100 req/menit per IP) |
| `500` | Internal Server Error |
| `503` | Service Unavailable — tidak ada collector tersedia |

**Format Error Response:**
```json
{
  "success": false,
  "error": "Pesan error yang menjelaskan masalah"
}
```

---

## Cara Test di Postman

### 1. Buat Akun Admin
```
POST /api/v1/auth/register-admin
Header: X-Admin-Secret: ecotracker-admin-secret-2026
```

### 2. Buat Akun Collector
```
POST /api/v1/auth/register-collector
Header: X-Admin-Secret: ecotracker-admin-secret-2026
```

### 3. Test Alur Pickup Lengkap
```
1. Login sebagai collector → simpan token
2. PUT /collector/location → update GPS collector
3. PUT /collector/status → { "is_online": true }
4. Login sebagai user → simpan token
5. POST /pickups → buat pickup (form-data)
6. GET /pickups/my → cek status (harusnya "assigned")
7. POST /collector/pickups/:id/accept → collector terima
8. POST /collector/pickups/:id/start → mulai berangkat
9. POST /collector/pickups/:id/arrive → tiba di lokasi
10. POST /collector/pickups/:id/complete → selesaikan + input sampah
11. GET /auth/profile (user) → cek poin bertambah
12. GET /badges/my (user) → cek badge baru
```

---

*EcoTracker V2.0 — Go + PostgreSQL + Supabase*
