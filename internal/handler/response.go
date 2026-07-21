package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"halogold-api/internal/domain"
)

// errorBody adalah format error yang konsisten untuk seluruh endpoint.
// Response sukses sengaja TIDAK dibungkus envelope agar sesuai persis dengan
// contoh di BRD (mis. {"price":1945200}).
type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// respondError memetakan error domain -> HTTP status yang tepat.
// Logic bisnis tidak tahu soal HTTP; pemetaan terpusat di sini.
func respondError(c *gin.Context, err error) {
	appErr, ok := domain.AsAppError(err)
	if !ok {
		// Error tak terduga: jangan bocorkan detail internal ke klien.
		c.JSON(http.StatusInternalServerError, errorBody{
			Error: errorDetail{Code: domain.CodeInternal, Message: "Terjadi kesalahan pada server"},
		})
		return
	}

	status := statusForCode(appErr.Code)
	c.JSON(status, errorBody{
		Error: errorDetail{Code: appErr.Code, Message: appErr.Message},
	})
}

func statusForCode(code string) int {
	switch code {
	case domain.CodeValidation, domain.CodeInvalidAmount, domain.CodeInvalidGram:
		return http.StatusBadRequest
	case domain.CodeInsufficientGold:
		return http.StatusUnprocessableEntity
	case domain.CodeUserNotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
