package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"halogold-api/internal/domain"
	"halogold-api/internal/service"
)

// GoldHandler menerima request HTTP, memvalidasi input, memanggil service,
// lalu memformat response. Handler tidak berisi logic bisnis.
type GoldHandler struct {
	svc           *service.GoldService
	defaultUserID int64
}

func NewGoldHandler(svc *service.GoldService, defaultUserID int64) *GoldHandler {
	return &GoldHandler{svc: svc, defaultUserID: defaultUserID}
}

// GET /price -> {"price": 1945200}
func (h *GoldHandler) GetPrice(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"price": h.svc.CurrentPrice()})
}

// GET /transactions -> daftar transaksi user default.
func (h *GoldHandler) ListTransactions(c *gin.Context) {
	txs, err := h.svc.ListTransactions(c.Request.Context(), h.defaultUserID, 50)
	if err != nil {
		respondError(c, err)
		return
	}
	// Selalu kembalikan array (bukan null) agar konsumen frontend aman.
	c.JSON(http.StatusOK, gin.H{"data": txs})
}

// buyRequest: {"amount": 500000}
type buyRequest struct {
	Amount int64 `json:"amount" binding:"required"`
}

// POST /buy -> {"gram": 0.2571, "price": 1945200}
func (h *GoldHandler) Buy(c *gin.Context) {
	var req buyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, domain.NewAppError(domain.CodeValidation, "Field 'amount' wajib diisi dan berupa angka", err))
		return
	}

	tx, err := h.svc.Buy(c.Request.Context(), h.defaultUserID, req.Amount)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"gram":  tx.Gram,
		"price": tx.Price,
	})
}

// sellRequest: {"gram": 1}
// Gram diterima sebagai decimal (string atau number) untuk menjaga presisi.
type sellRequest struct {
	Gram decimal.Decimal `json:"gram" binding:"required"`
}

// POST /sell -> {"amount": 1945200}
func (h *GoldHandler) Sell(c *gin.Context) {
	var req sellRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, domain.NewAppError(domain.CodeValidation, "Field 'gram' wajib diisi dan berupa angka", err))
		return
	}

	tx, err := h.svc.Sell(c.Request.Context(), h.defaultUserID, req.Gram)
	if err != nil {
		respondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"amount": tx.Amount})
}

// GET /balance -> saldo emas user (endpoint tambahan yang berguna untuk verifikasi).
func (h *GoldHandler) GetBalance(c *gin.Context) {
	bal, err := h.svc.Balance(c.Request.Context(), h.defaultUserID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"gram": bal})
}
