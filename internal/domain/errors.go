package domain

import (
	"errors"
	"fmt"
)

// AppError adalah error domain yang membawa kode & pesan yang aman untuk
// ditampilkan ke klien. Handler HTTP memetakan Code -> HTTP status,
// sehingga logic bisnis tidak perlu tahu soal HTTP sama sekali.
type AppError struct {
	Code    string // kode stabil, mis. "INVALID_AMOUNT", "INSUFFICIENT_BALANCE"
	Message string // pesan ramah untuk klien
	Err     error  // error asli (opsional) untuk logging internal
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Err }

// NewAppError membuat AppError baru.
func NewAppError(code, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

// Kode error yang dipakai lintas layer.
const (
	CodeValidation       = "VALIDATION_ERROR"
	CodeInvalidAmount    = "INVALID_AMOUNT"
	CodeInvalidGram      = "INVALID_GRAM"
	CodeInsufficientGold = "INSUFFICIENT_BALANCE"
	CodeUserNotFound     = "USER_NOT_FOUND"
	CodeInternal         = "INTERNAL_ERROR"
)

// AsAppError mengekstrak *AppError dari rantai error, bila ada.
func AsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
