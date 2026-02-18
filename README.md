# 🌱 EcoTracker API

Platform Pengelolaan Sampah Berbasis Poin — Backend API dibangun dengan **Golang**, **Gin Gonic**, dan **Supabase (PostgreSQL)**.

---

## 🏗️ Arsitektur & Struktur Folder

```
ecotracker/
├── main.go                          # Entry point
├── .env                             # Environment variables
├── go.mod                           # Go module dependencies
├── migrations/
│   └── 001_init.sql                 # Database schema & seed data
├── cmd/
│   └── server/
│       └── server.go                # Router & dependency injection
└── internal/
    ├── config/
    │   ├── config.go                # Load env variables
    │   └── database.go              # pgxpool connection
    ├── domain/
    │   ├── models.go                # All structs/entities
    │   └── errors.go                # Sentinel errors
    ├── repository/
    │   ├── auth_repository.go       # Profile DB operations
    │   ├── pickup_repository.go     # Pickup + atomic TX
    │   ├── waste_category_repository.go
    │   ├── point_log_repository.go
    │   └── voucher_repository.go    # Voucher + claim TX
    ├── service/
    │   ├── auth_service.go          # Business logic: register/login
    │   ├── pickup_service.go        # Business logic: pickup lifecycle
    │   ├── voucher_service.go       # Business logic: voucher claim
    │   └── misc_services.go         # PointLog & WasteCategory services
    ├── handler/
    │   ├── auth_handler.go          # HTTP handlers: auth
    │   ├── pickup_handler.go        # HTTP handlers: pickup
    │   ├── voucher_handler.go       # HTTP handlers: voucher
    │   └── misc_handler.go          # HTTP handlers: categories & points
    ├── middleware/
    │   └── auth.go                  # JWT auth + role guard middleware
    └── utils/
        ├── jwt.go                   # Token generate/validate
        ├── response.go              # Consistent JSON response
        ├── image.go                 # Image processing (WebP/JPG/PNG)
        └── storage.go               # Supabase Storage upload client
```

---

## ⚙️ Setup & Menjalankan

### 1. Prasyarat
- Go 1.22+
- Akun Supabase (sudah dibuat)

### 2. Setup Database
Buka **Supabase SQL Editor** dan jalankan isi file `migrations/001_init.sql`. File ini akan membuat semua tabel dan mengisi data awal (kategori sampah & voucher).

### 3. Konfigurasi Environment
File `.env` sudah tersedia. Pastikan nilainya sesuai:
```env
PORT=8080
DB_URL=postgres://...
SUPABASE_URL=https://...supabase.co
SUPABASE_SERVICE_ROLE_KEY=eyJ...
JWT_SECRET=ecotracker-super-secret-jwt-key-2026
STORAGE_BUCKET=pickups
```

### 4. Setup Supabase Storage
Di dashboard Supabase, buat bucket bernama **`pickups`** dan set sebagai **Public bucket**.

### 5. Install Dependencies & Run
```bash
go mod tidy
go run main.go
```

Server akan berjalan di `http://localhost:8080`

---

## 🔐 Autentikasi

Semua endpoint yang dilindungi membutuhkan header:
```
Authorization: Bearer <token_jwt>
```

Token diperoleh dari endpoint `/auth/login` atau `/auth/register`.

### Peran (Role)
| Role        | Aksi yang Diizinkan                                 |
|-------------|-----------------------------------------------------|
| `user`      | Buat pickup, lihat pickup sendiri, klaim voucher    |
| `collector` | Lihat semua pending pickup, ambil task, selesaikan  |

---

## 📋 Dokumentasi Endpoint API

### Base URL
```
http://localhost:8080/api/v1
```

---

### 🏥 Health Check

#### `GET /health`
Cek status server.

**Response (200):**
```json
{
  "status": "ok",
  "service": "ecotracker"
}
```

---

### 🔑 Auth

#### `POST /auth/register`
Daftarkan pengguna baru.

**Headers:**
```
Content-Type: application/json
```

**Body:**
```json
{
  "name": "Budi Santoso",
  "email": "budi@example.com",
  "phone": "08123456789",
  "password": "password123",
  "role": "user"
}
```
> `role` harus `"user"` atau `"collector"`

**Response (201):**
```json
{
  "success": true,
  "message": "Registration successful",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "profile": {
      "id": "uuid-string",
      "name": "Budi Santoso",
      "email": "budi@example.com",
      "role": "user",
      "total_points": 0,
      "created_at": "2026-02-17T13:00:00Z"
    }
  }
}
```

---

#### `POST /auth/login`
Login dan dapatkan token JWT.

**Headers:**
```
Content-Type: application/json
```

**Body:**
```json
{
  "email": "budi@example.com",
  "password": "password123"
}
```

**Response (200):**
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "profile": {
      "id": "uuid-string",
      "name": "Budi Santoso",
      "email": "budi@example.com",
      "role": "user",
      "total_points": 250
    }
  }
}
```

---

#### `GET /auth/profile`
Dapatkan profil pengguna yang sedang login.

**Headers:**
```
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "success": true,
  "message": "Profile retrieved",
  "data": {
    "id": "uuid-string",
    "name": "Budi Santoso",
    "email": "budi@example.com",
    "phone": "08123456789",
    "role": "user",
    "total_points": 250,
    "address_default": "Jl. Merdeka No. 1, Jakarta",
    "created_at": "2026-02-17T13:00:00Z"
  }
}
```

---

### ♻️ Kategori Sampah

#### `GET /categories`
Dapatkan semua kategori sampah beserta nilai poin per kg.

**Headers:**
```
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "success": true,
  "message": "Waste categories retrieved",
  "data": [
    { "id": 1, "name": "Kertas / Kardus", "points_per_kg": 10, "unit": "kg" },
    { "id": 2, "name": "Plastik",         "points_per_kg": 15, "unit": "kg" },
    { "id": 3, "name": "Logam / Besi",    "points_per_kg": 20, "unit": "kg" },
    { "id": 4, "name": "Elektronik (E-Waste)", "points_per_kg": 50, "unit": "kg" }
  ]
}
```

---

### 📦 Pickup (User)

#### `POST /pickups`
Buat permintaan jemput sampah baru. Mendukung upload foto.

**Headers:**
```
Authorization: Bearer <token>
Content-Type: multipart/form-data
```

**Form Data:**
| Field       | Type    | Keterangan                                   |
|-------------|---------|----------------------------------------------|
| `address`   | string  | **Wajib.** Alamat lengkap                    |
| `latitude`  | float   | **Wajib.** Koordinat GPS latitude             |
| `longitude` | float   | **Wajib.** Koordinat GPS longitude            |
| `notes`     | string  | Opsional. Catatan tambahan                   |
| `photo`     | file    | Opsional. Foto sampah (JPG/PNG/WebP, max 10MB) |

**Response (201):**
```json
{
  "success": true,
  "message": "Pickup request created successfully",
  "data": {
    "id": "uuid-pickup",
    "user_id": "uuid-user",
    "collector_id": "",
    "status": "pending",
    "address": "Jl. Sudirman No. 5, Jakarta",
    "latitude": -6.2088,
    "longitude": 106.8456,
    "photo_url": "https://xxx.supabase.co/storage/v1/object/public/pickups/...",
    "notes": "Sampah di depan pagar",
    "created_at": "2026-02-17T13:00:00Z"
  }
}
```

---

#### `GET /pickups/my`
Lihat semua pickup milik user yang login.

**Headers:**
```
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "success": true,
  "message": "My pickups retrieved",
  "data": [
    {
      "id": "uuid-pickup",
      "status": "completed",
      "address": "Jl. Sudirman No. 5",
      "created_at": "2026-02-17T13:00:00Z"
    }
  ]
}
```

---

#### `GET /pickups/:id`
Lihat detail pickup tertentu (user hanya bisa lihat miliknya, collector bisa lihat semua).

**Headers:**
```
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "success": true,
  "message": "Pickup detail retrieved",
  "data": {
    "id": "uuid-pickup",
    "status": "completed",
    "address": "Jl. Sudirman No. 5",
    "latitude": -6.2088,
    "longitude": 106.8456,
    "photo_url": "https://...",
    "completed_at": "2026-02-17T14:30:00Z",
    "items": [
      { "category_id": 2, "weight": 3.5, "subtotal_points": 52 },
      { "category_id": 1, "weight": 5.0, "subtotal_points": 50 }
    ]
  }
}
```

---

### 🚚 Collector Workflow

#### `GET /collector/pickups/pending`
Dashboard collector — lihat semua pickup berstatus `pending`.

**Headers:**
```
Authorization: Bearer <token>  (role: collector)
```

**Response (200):**
```json
{
  "success": true,
  "message": "Pending pickups retrieved",
  "data": [
    {
      "id": "uuid-1",
      "user_id": "uuid-user",
      "status": "pending",
      "address": "Jl. Kebon Jeruk, Jakarta",
      "latitude": -6.1944,
      "longitude": 106.7893,
      "photo_url": "https://...",
      "created_at": "2026-02-17T10:00:00Z"
    }
  ]
}
```

---

#### `GET /collector/pickups/my-tasks`
Lihat semua task yang sudah diambil oleh collector ini.

**Headers:**
```
Authorization: Bearer <token>  (role: collector)
```

**Response (200):** _(format sama seperti di atas)_

---

#### `POST /collector/pickups/:id/take`
Ambil task pickup dari status `pending` menjadi `taken`.

**Headers:**
```
Authorization: Bearer <token>  (role: collector)
```

**Response (200):**
```json
{
  "success": true,
  "message": "Task taken successfully",
  "data": {
    "id": "uuid-pickup",
    "collector_id": "uuid-collector",
    "status": "taken",
    "address": "Jl. Kebon Jeruk, Jakarta"
  }
}
```

---

#### `POST /collector/pickups/:id/complete`
Selesaikan task pickup (transaksi atomik: update status, simpan item, tambah poin, catat log).

**Headers:**
```
Authorization: Bearer <token>  (role: collector)
Content-Type: application/json
```

**Body:**
```json
{
  "items": [
    { "category_id": 2, "weight": 3.5 },
    { "category_id": 1, "weight": 5.0 }
  ]
}
```

> Sistem akan otomatis menghitung poin:
> - Plastik (3.5 kg × 15 poin/kg = 52 poin)
> - Kertas (5.0 kg × 10 poin/kg = 50 poin)
> - **Total: 102 poin** ditambahkan ke akun user

**Response (200):**
```json
{
  "success": true,
  "message": "Pickup completed. Points awarded successfully",
  "data": {
    "id": "uuid-pickup",
    "status": "completed",
    "completed_at": "2026-02-17T14:30:00Z",
    "items": [
      { "category_id": 2, "weight": 3.5, "subtotal_points": 52 },
      { "category_id": 1, "weight": 5.0, "subtotal_points": 50 }
    ]
  }
}
```

---

### 💰 Poin

#### `GET /points/logs`
Lihat riwayat transaksi poin (earn/spend).

**Headers:**
```
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "success": true,
  "message": "Point logs retrieved",
  "data": [
    {
      "id": 1,
      "user_id": "uuid-user",
      "amount": 102,
      "transaction_type": "earn",
      "reference_id": "uuid-pickup",
      "description": "Points earned from waste pickup",
      "created_at": "2026-02-17T14:30:00Z"
    },
    {
      "id": 2,
      "amount": 100,
      "transaction_type": "spend",
      "description": "Points spent on voucher redemption",
      "created_at": "2026-02-17T15:00:00Z"
    }
  ]
}
```

---

### 🎁 Voucher

#### `GET /vouchers`
Lihat semua voucher yang tersedia (aktif & stok > 0).

**Headers:**
```
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "success": true,
  "message": "Available vouchers retrieved",
  "data": [
    {
      "id": 1,
      "title": "Voucher Belanja Rp 10.000",
      "description": "Diskon belanja Rp 10.000 di mitra kami",
      "point_cost": 100,
      "stock": 50,
      "is_active": true
    }
  ]
}
```

---

#### `POST /vouchers/:id/claim`
Tukar poin untuk mendapatkan voucher.

**Headers:**
```
Authorization: Bearer <token>  (role: user)
```

**Response (201):**
```json
{
  "success": true,
  "message": "Voucher claimed successfully",
  "data": {
    "id": 1,
    "user_id": "uuid-user",
    "voucher_id": 1,
    "claim_code": "ECO-a1b2c3d4",
    "is_used": false,
    "claimed_at": "2026-02-17T15:00:00Z",
    "voucher": {
      "title": "Voucher Belanja Rp 10.000",
      "point_cost": 100
    }
  }
}
```

---

#### `GET /vouchers/my`
Lihat semua voucher yang sudah diklaim oleh user.

**Headers:**
```
Authorization: Bearer <token>
```

**Response (200):**
```json
{
  "success": true,
  "message": "My vouchers retrieved",
  "data": [
    {
      "id": 1,
      "claim_code": "ECO-a1b2c3d4",
      "is_used": false,
      "claimed_at": "2026-02-17T15:00:00Z",
      "voucher": {
        "title": "Voucher Belanja Rp 10.000",
        "point_cost": 100
      }
    }
  ]
}
```

---

## ❌ Error Responses

Semua error menggunakan format konsisten:

```json
{
  "success": false,
  "error": "pesan error di sini"
}
```

| HTTP Status | Kondisi                                           |
|-------------|---------------------------------------------------|
| `400`       | Request tidak valid / body salah                  |
| `401`       | Token tidak ada / expired / kredensial salah      |
| `403`       | Role tidak cukup (user coba akses endpoint collector) |
| `404`       | Resource tidak ditemukan                          |
| `409`       | Konflik (email sudah terdaftar, pickup sudah diambil) |
| `500`       | Internal server error                             |

---

## 🔄 Alur Lengkap (Happy Path)

```
1. User REGISTER (role: user)
2. User LOGIN → dapat JWT
3. User POST /pickups → upload foto sampah + GPS
   ↓ Foto diproses (resize, konversi ke JPEG) lalu upload ke Supabase Storage
4. Collector LOGIN (role: collector)
5. Collector GET /collector/pickups/pending → lihat list
6. Collector POST /collector/pickups/:id/take → ambil task
7. Collector POST /collector/pickups/:id/complete + {items: [...]}
   ↓ DB Transaction Atomik:
     a. UPDATE pickups SET status='completed'
     b. INSERT pickup_items (detail berat per kategori)
     c. UPDATE profiles SET total_points = total_points + {total}
     d. INSERT point_logs (transaction_type='earn')
8. User GET /points/logs → lihat poin bertambah
9. User GET /vouchers → lihat voucher tersedia
10. User POST /vouchers/:id/claim
    ↓ DB Transaction Atomik:
      a. UPDATE profiles SET total_points = total_points - {cost}
      b. UPDATE vouchers SET stock = stock - 1
      c. INSERT user_vouchers (dengan claim_code unik)
      d. INSERT point_logs (transaction_type='spend')
11. User GET /vouchers/my → lihat kode voucher
```

---

## 🛠️ Tech Stack

| Komponen        | Library/Service                          |
|-----------------|------------------------------------------|
| Framework       | [Gin Gonic](https://gin-gonic.com/)      |
| Database        | PostgreSQL via [pgx/v5](https://github.com/jackc/pgx) |
| Database Host   | [Supabase](https://supabase.com/)        |
| Storage         | Supabase Storage (REST API)              |
| Auth            | JWT via [golang-jwt/jwt](https://github.com/golang-jwt/jwt) |
| Password Hash   | bcrypt via [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) |
| Image Processing| [disintegration/imaging](https://github.com/disintegration/imaging) |
| UUID            | [google/uuid](https://github.com/google/uuid) |
| Config          | [joho/godotenv](https://github.com/joho/godotenv) |
