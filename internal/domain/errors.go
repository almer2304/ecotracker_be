package domain

import "errors"

// Application errors yang bisa di-handle dengan HTTP status code tertentu
var (
	// Auth
	ErrEmailAlreadyExists  = errors.New("email sudah terdaftar")
	ErrInvalidCredentials  = errors.New("email atau password salah")
	ErrInvalidToken        = errors.New("token tidak valid")
	ErrTokenExpired        = errors.New("token sudah kadaluarsa")
	ErrUnauthorized        = errors.New("tidak memiliki akses")
	ErrForbidden           = errors.New("akses ditolak")

	// Resource
	ErrNotFound            = errors.New("data tidak ditemukan")
	ErrPickupNotFound      = errors.New("pickup tidak ditemukan")
	ErrCollectorNotFound   = errors.New("collector tidak ditemukan")
	ErrBadgeNotFound       = errors.New("badge tidak ditemukan")
	ErrReportNotFound      = errors.New("laporan tidak ditemukan")
	ErrFeedbackNotFound    = errors.New("feedback tidak ditemukan")
	ErrCategoryNotFound    = errors.New("kategori tidak ditemukan")

	// Business logic
	ErrNoCollectorAvailable = errors.New("tidak ada collector yang tersedia saat ini")
	ErrPickupAlreadyAssigned = errors.New("pickup sudah ditugaskan ke collector lain")
	ErrCollectorOffline      = errors.New("collector sedang offline")
	ErrCollectorBusy         = errors.New("collector sedang menangani pickup lain")
	ErrInvalidPickupStatus   = errors.New("status pickup tidak valid untuk operasi ini")
	ErrBadgeAlreadyAwarded   = errors.New("badge sudah pernah diberikan")
	ErrInvalidStatus         = errors.New("status tidak valid")

	// Validation
	ErrInvalidInput        = errors.New("input tidak valid")
	ErrFileTooLarge        = errors.New("ukuran file terlalu besar")
	ErrInvalidFileType     = errors.New("tipe file tidak didukung")
)

// AppError adalah error dengan HTTP status code
type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(code int, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}
