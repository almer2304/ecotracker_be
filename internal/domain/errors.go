package domain

import "errors"

// Sentinel errors untuk service layer
var (
	ErrNotFound          = errors.New("resource not found")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden: insufficient permissions")
	ErrConflict          = errors.New("resource already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInsufficientPoints = errors.New("insufficient points balance")
	ErrVoucherOutOfStock  = errors.New("voucher is out of stock")
	ErrVoucherInactive    = errors.New("voucher is not active")
	ErrPickupNotPending   = errors.New("pickup is not in pending status")
	ErrPickupNotTaken     = errors.New("pickup has not been taken yet")
	ErrPickupAlreadyTaken = errors.New("pickup has already been taken by a collector")
	ErrInvalidRole        = errors.New("invalid role for this action")
	ErrEmailAlreadyExists = errors.New("email is already registered")
)
