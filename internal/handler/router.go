package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// NewRouter merakit seluruh route dan middleware.
func NewRouter(gold *GoldHandler) *gin.Engine {
	r := gin.New()

	// Middleware bawaan: recovery (cegah panic menjatuhkan server) + logger.
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// Health check untuk liveness/readiness probe.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Endpoint sesuai BRD.
	r.GET("/price", gold.GetPrice)
	r.GET("/transactions", gold.ListTransactions)
	r.POST("/buy", gold.Buy)
	r.POST("/sell", gold.Sell)

	// Endpoint tambahan (bonus) untuk verifikasi saldo.
	r.GET("/balance", gold.GetBalance)

	return r
}
