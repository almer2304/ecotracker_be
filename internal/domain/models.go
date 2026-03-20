package domain

import (
	"time"
)

// ============================================================
// ENUMS
// ============================================================

type UserRole string

const (
	RoleUser      UserRole = "user"
	RoleCollector UserRole = "collector"
	RoleAdmin     UserRole = "admin"
)

type PickupStatus string

const (
	StatusPending    PickupStatus = "pending"
	StatusAssigned   PickupStatus = "assigned"
	StatusReassigned PickupStatus = "reassigned"
	StatusAccepted   PickupStatus = "accepted"
	StatusInProgress PickupStatus = "in_progress"
	StatusArrived    PickupStatus = "arrived"
	StatusCompleted  PickupStatus = "completed"
	StatusCancelled  PickupStatus = "cancelled"
)

type ReportSeverity string

const (
	SeverityLow    ReportSeverity = "low"
	SeverityMedium ReportSeverity = "medium"
	SeverityHigh   ReportSeverity = "high"
)

type ReportStatus string

const (
	ReportStatusNew          ReportStatus = "new"
	ReportStatusInvestigating ReportStatus = "investigating"
	ReportStatusAssigned     ReportStatus = "assigned"
	ReportStatusInProgress   ReportStatus = "in_progress"
	ReportStatusResolved     ReportStatus = "resolved"
)

type FeedbackType string

const (
	FeedbackApp       FeedbackType = "app"
	FeedbackCollector FeedbackType = "collector"
	FeedbackGeneral   FeedbackType = "general"
)

type PointLogType string

const (
	PointEarned     PointLogType = "earned"
	PointSpent      PointLogType = "spent"
	PointAdjustment PointLogType = "adjustment"
)

// ============================================================
// CORE MODELS
// ============================================================

// Profile mewakili semua pengguna (user, collector, admin)
type Profile struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Email string   `json:"email"`
	Phone *string  `json:"phone,omitempty"`
	Role  UserRole `json:"role"`

	AvatarURL *string `json:"avatar_url,omitempty"`

	// User stats
	TotalPoints           int `json:"total_points"`
	TotalPickupsCompleted int `json:"total_pickups_completed"`
	TotalReportsSubmitted int `json:"total_reports_submitted"`

	// Collector stats
	IsOnline             bool    `json:"is_online"`
	IsBusy               bool    `json:"is_busy"`
	AverageRating        float64 `json:"average_rating"`
	TotalRatings         int     `json:"total_ratings"`
	TotalWeightCollected float64 `json:"total_weight_collected"`

	// Geolocation
	LastLat               *float64   `json:"last_lat,omitempty"`
	LastLon               *float64   `json:"last_lon,omitempty"`
	LastLocationUpdatedAt *time.Time `json:"last_location_updated_at,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

// WasteCategory kategori jenis sampah
type WasteCategory struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	PointsPerKg int     `json:"points_per_kg"`
	IconURL     *string `json:"icon_url,omitempty"`
	ColorHex    *string `json:"color_hex,omitempty"`
	IsActive    bool    `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// Pickup permintaan pengambilan sampah
type Pickup struct {
	ID          string       `json:"id"`
	UserID      string       `json:"user_id"`
	CollectorID *string      `json:"collector_id,omitempty"`
	Address     string       `json:"address"`
	Lat         float64      `json:"lat"`
	Lon         float64      `json:"lon"`
	PhotoURL    *string      `json:"photo_url,omitempty"`
	Notes       *string      `json:"notes,omitempty"`
	Status      PickupStatus `json:"status"`

	AssignedAt        *time.Time `json:"assigned_at,omitempty"`
	AssignmentTimeout *time.Time `json:"assignment_timeout,omitempty"`
	ReassignmentCount int        `json:"reassignment_count"`

	AcceptedAt  *time.Time `json:"accepted_at,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	ArrivedAt   *time.Time `json:"arrived_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	TotalWeight       *float64 `json:"total_weight,omitempty"`
	TotalPointsAwarded *int    `json:"total_points_awarded,omitempty"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`

	// Joined fields
	User      *Profile      `json:"user,omitempty"`
	Collector *Profile      `json:"collector,omitempty"`
	Items     []PickupItem  `json:"items,omitempty"`
}

// PickupItem detail sampah per kategori
type PickupItem struct {
	ID            string        `json:"id"`
	PickupID      string        `json:"pickup_id"`
	CategoryID    string        `json:"category_id"`
	WeightKg      float64       `json:"weight_kg"`
	PointsAwarded int           `json:"points_awarded"`
	CreatedAt     time.Time     `json:"created_at"`
	Category      *WasteCategory `json:"category,omitempty"`
}

// AssignmentHistory riwayat penugasan collector
type AssignmentHistory struct {
	ID            string     `json:"id"`
	PickupID      string     `json:"pickup_id"`
	CollectorID   string     `json:"collector_id"`
	AssignedAt    time.Time  `json:"assigned_at"`
	TimeoutAt     *time.Time `json:"timeout_at,omitempty"`
	ReleasedAt    *time.Time `json:"released_at,omitempty"`
	ReleaseReason *string    `json:"release_reason,omitempty"`
	DistanceKm    *float64   `json:"distance_km,omitempty"`
}

// Badge definisi pencapaian
type Badge struct {
	ID            string `json:"id"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	Description   *string `json:"description,omitempty"`
	IconURL       *string `json:"icon_url,omitempty"`
	ColorHex      *string `json:"color_hex,omitempty"`
	CriteriaType  string `json:"criteria_type"`
	CriteriaValue int    `json:"criteria_value"`
	DisplayOrder  int    `json:"display_order"`
	CreatedAt     time.Time `json:"created_at"`

	// Joined (saat get user badges)
	IsUnlocked *bool      `json:"is_unlocked,omitempty"`
	AwardedAt  *time.Time `json:"awarded_at,omitempty"`
	Progress   *int       `json:"progress,omitempty"`
}

// UserBadge badge yang telah diraih user
type UserBadge struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	BadgeID   string    `json:"badge_id"`
	AwardedAt time.Time `json:"awarded_at"`
	Badge     *Badge    `json:"badge,omitempty"`
}

// PointLog log transaksi poin
type PointLog struct {
	ID           string       `json:"id"`
	UserID       string       `json:"user_id"`
	PickupID     *string      `json:"pickup_id,omitempty"`
	LogType      PointLogType `json:"log_type"`
	Points       int          `json:"points"`
	Description  *string      `json:"description,omitempty"`
	BalanceAfter int          `json:"balance_after"`
	CreatedAt    time.Time    `json:"created_at"`
}

// AreaReport laporan area kotor
type AreaReport struct {
	ID          string         `json:"id"`
	ReporterID  string         `json:"reporter_id"`
	AssignedTo  *string        `json:"assigned_to,omitempty"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Address     string         `json:"address"`
	Lat         float64        `json:"lat"`
	Lon         float64        `json:"lon"`
	Severity    ReportSeverity `json:"severity"`
	Status      ReportStatus   `json:"status"`
	PhotoURLs   []string       `json:"photo_urls"`
	AdminNotes  *string        `json:"admin_notes,omitempty"`
	ResolvedAt  *time.Time     `json:"resolved_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Reporter    *Profile       `json:"reporter,omitempty"`
}

// Feedback ulasan dan penilaian
type Feedback struct {
	ID            string       `json:"id"`
	UserID        string       `json:"user_id"`
	PickupID      *string      `json:"pickup_id,omitempty"`
	CollectorID   *string      `json:"collector_id,omitempty"`
	FeedbackType  FeedbackType `json:"feedback_type"`
	Rating        *int         `json:"rating,omitempty"`
	Title         *string      `json:"title,omitempty"`
	Comment       *string      `json:"comment,omitempty"`
	Tags          []string     `json:"tags"`
	AdminResponse *string      `json:"admin_response,omitempty"`
	RespondedAt   *time.Time   `json:"responded_at,omitempty"`
	RespondedBy   *string      `json:"responded_by,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
	User          *Profile     `json:"user,omitempty"`
	Collector     *Profile     `json:"collector,omitempty"`
}

// ============================================================
// REQUEST / RESPONSE DTOs
// ============================================================

// Auth
type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Phone    string `json:"phone" binding:"omitempty,min=9,max=20"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type AuthResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int64    `json:"expires_in"` // seconds
	User         *Profile `json:"user"`
}

// Pickup
type CreatePickupRequest struct {
	Address string  `form:"address" binding:"required"`
	Lat     float64 `form:"lat" binding:"required"`
	Lon     float64 `form:"lon" binding:"required"`
	Notes   string  `form:"notes"`
}

// CollectorStatus
type UpdateStatusRequest struct {
	IsOnline bool `json:"is_online"`
}

type UpdateLocationRequest struct {
	Lat float64 `json:"lat" binding:"required"`
	Lon float64 `json:"lon" binding:"required"`
}

type CompletePickupRequest struct {
	Items []PickupItemInput `json:"items" binding:"required,min=1"`
}

type PickupItemInput struct {
	CategoryID string  `json:"category_id" binding:"required,uuid"`
	WeightKg   float64 `json:"weight_kg" binding:"required,gt=0"`
}

// Report
type CreateReportRequest struct {
	Title       string `form:"title" binding:"required,min=5,max=200"`
	Description string `form:"description" binding:"required,min=10"`
	Address     string `form:"address" binding:"required"`
	Lat         float64 `form:"lat" binding:"required"`
	Lon         float64 `form:"lon" binding:"required"`
	Severity    string `form:"severity" binding:"required,oneof=low medium high"`
}

type UpdateReportRequest struct {
	Status     string `json:"status" binding:"required,oneof=new investigating assigned in_progress resolved"`
	AdminNotes string `json:"admin_notes"`
	AssignedTo string `json:"assigned_to"`
}

// Feedback
type CreateFeedbackRequest struct {
	FeedbackType string   `json:"feedback_type" binding:"required,oneof=app collector general"`
	PickupID     string   `json:"pickup_id"`
	Rating       *int     `json:"rating" binding:"omitempty,min=1,max=5"`
	Title        string   `json:"title" binding:"omitempty,max=200"`
	Comment      string   `json:"comment"`
	Tags         []string `json:"tags"`
}

type AdminFeedbackResponse struct {
	Response string `json:"response" binding:"required"`
}

// Admin - Create collector
type CreateCollectorRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Phone    string `json:"phone" binding:"omitempty"`
}

// Pagination
type PaginationQuery struct {
	Page  int `form:"page,default=1" binding:"min=1"`
	Limit int `form:"limit,default=20" binding:"min=1,max=100"`
}

func (p *PaginationQuery) Offset() int {
	return (p.Page - 1) * p.Limit
}

// Generic list response
type ListResponse struct {
	Data       interface{} `json:"data"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	TotalPages int         `json:"total_pages"`
}

// Dashboard stats
type DashboardStats struct {
	TotalUsers        int     `json:"total_users"`
	TotalCollectors   int     `json:"total_collectors"`
	OnlineCollectors  int     `json:"online_collectors"`
	TotalPickups      int     `json:"total_pickups"`
	PendingPickups    int     `json:"pending_pickups"`
	CompletedPickups  int     `json:"completed_pickups"`
	TotalWeightKg     float64 `json:"total_weight_kg"`
	TotalReports      int     `json:"total_reports"`
	NewReports        int     `json:"new_reports"`
}
