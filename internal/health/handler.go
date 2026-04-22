package health

import (
	"context"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2"
	"mangosteen/internal/db"
	"mangosteen/pkg/cache"
)

type Handler struct {
	db    *db.DB
	cache *cache.ValkeyClient
}

func NewHandler(database *db.DB, cache *cache.ValkeyClient) *Handler {
	return &Handler{db: database, cache: cache}
}

func (h *Handler) RegisterRoutes(app fiber.Router) {
	health := app.Group("/health")
	health.Get("/live", h.Live)
	health.Get("/ready", h.Ready)
	health.Get("/metrics", h.Metrics)
}

func (h *Handler) Live(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "alive",
		"time":   time.Now(),
	})
}

func (h *Handler) Ready(c *fiber.Ctx) error {
	ctx := context.Background()
	if err := h.db.PingContext(ctx); err != nil {
		return c.Status(503).JSON(fiber.Map{
			"status":   "not_ready",
			"database": "error",
			"time":     time.Now(),
		})
	}

	return c.JSON(fiber.Map{
		"status":   "ready",
		"database": "ok",
		"cache":    "ok",
		"time":     time.Now(),
	})
}

func (h *Handler) Metrics(c *fiber.Ctx) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return c.JSON(fiber.Map{
		"goroutines": runtime.NumGoroutine(),
		"memory": fiber.Map{
			"alloc":   m.Alloc,
			"total":   m.TotalAlloc,
			"sys":     m.Sys,
			"gc":      m.NumGC,
		},
		"time": time.Now(),
	})
}