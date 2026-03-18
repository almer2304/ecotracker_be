package domain

import "time"

// ─── Auth ───────────────────────────────────────────────────────────────────

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Phone    string `json:"phone"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=user collector"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token   string   `json:"token"`
	Profile *Profile `json:"profile"`
}

// ─── Profile ─────────────────────────────────────────────────────────────────

// Profile maps to the `profiles` table (UUID primary key)
type Profile struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Email          string     `json:"email"`
	Phone          string     `json:"phone,omitempty"`
	Role           string     `json:"role"`
	TotalPoints    int        `json:"total_points"`
	AddressDefault string     `json:"address_default,omitempty"`
	AvatarURL      string     `json:"avatar_url,omitempty"`
	PasswordHash   string     `json:"-"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ─── Waste Categories ─────────────────────────────────────────────────────────

type WasteCategory struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	PointsPerKg int    `json:"points_per_kg"`
	Unit        string `json:"unit"`
	IconURL     string `json:"icon_url,omitempty"`
}

// ─── Pickup ───────────────────────────────────────────────────────────────────

// Pickup maps to the `pickups` table (UUID primary key)
type Pickup struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	CollectorID string     `json:"collector_id,omitempty"`
	Status      string     `json:"status"` // pending | taken | completed | cancelled
	Address     string     `json:"address"`
	Latitude    float64    `json:"latitude"`
	Longitude   float64    `json:"longitude"`
	PhotoURL    string     `json:"photo_url,omitempty"`
	Notes       string     `json:"notes,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type CreatePickupRequest struct {
	Address   string  `form:"address" binding:"required"`
	Latitude  float64 `form:"latitude" binding:"required"`
	Longitude float64 `form:"longitude" binding:"required"`
	Notes     string  `form:"notes"`
}

type PickupDetail struct {
	Pickup
	Items []PickupItem `json:"items,omitempty"`
	User  *Profile     `json:"user,omitempty"`
}

type PickupWithDistance struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	CollectorID string     `json:"collector_id,omitempty"`
	Status      string     `json:"status"`
	Address     string     `json:"address"`
	Latitude    float64    `json:"latitude"`
	Longitude   float64    `json:"longitude"`
	PhotoURL    string     `json:"photo_url,omitempty"`
	Notes       string     `json:"notes,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	DistanceKm  float64    `json:"distance_km"`
}

// ─── Pickup Items ─────────────────────────────────────────────────────────────

// PickupItem maps to the `pickup_items` table (int4 PK)
type PickupItem struct {
	ID            int     `json:"id"`
	PickupID      string  `json:"pickup_id"`
	CategoryID    int     `json:"category_id"`
	Weight        float64 `json:"weight"` // in kg
	SubtotalPoints int    `json:"subtotal_points"`
}

type PickupItemInput struct {
	CategoryID int     `json:"category_id" binding:"required"`
	Weight     float64 `json:"weight" binding:"required,gt=0"`
}

type CompletePickupRequest struct {
	Items []PickupItemInput `json:"items" binding:"required,min=1,dive"`
}

// ─── Point Logs ──────────────────────────────────────────────────────────────

// PointLog maps to the `point_logs` table (int4 PK)
type PointLog struct {
	ID              int       `json:"id"`
	UserID          string    `json:"user_id"`
	Amount          int       `json:"amount"`
	TransactionType string    `json:"transaction_type"` // earn | spend
	ReferenceID     string    `json:"reference_id,omitempty"`
	Description     string    `json:"description"`
	CreatedAt       time.Time `json:"created_at"`
}

// ─── Vouchers ─────────────────────────────────────────────────────────────────

// Voucher maps to the `vouchers` table (int4 PK)
type Voucher struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	PointCost   int    `json:"point_cost"`
	Stock       int    `json:"stock"`
	ImageURL    string `json:"image_url,omitempty"`
	IsActive    bool   `json:"is_active"`
}

// UserVoucher maps to the `user_vouchers` table (int4 PK)
type UserVoucher struct {
	ID        int        `json:"id"`
	UserID    string     `json:"user_id"`
	VoucherID int        `json:"voucher_id"`
	ClaimCode string     `json:"claim_code"`
	IsUsed    bool       `json:"is_used"`
	ClaimedAt time.Time  `json:"claimed_at"`
	Voucher   *Voucher   `json:"voucher,omitempty"`
}

// ─── JWT Claims ───────────────────────────────────────────────────────────────

type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

// ─── Admin ───────────────────────────────────────────────────────────────────

// AdminStats for dashboard
type AdminStats struct {
	TotalUsers         int `json:"total_users"`
	TotalCollectors    int `json:"total_collectors"`
	PendingPickups     int `json:"pending_pickups"`
	CompletedPickups   int `json:"completed_pickups"`
	TotalPointsAwarded int `json:"total_points_awarded"`
}
