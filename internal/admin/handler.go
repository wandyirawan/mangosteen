package admin

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"mangosteen/internal/db"
	"mangosteen/pkg/worker"
)

type Handler struct {
	uploadWorker *worker.UploadWorker
	db           *db.DB
}

func NewHandler(uploadWorker *worker.UploadWorker, database *db.DB) *Handler {
	return &Handler{uploadWorker: uploadWorker, db: database}
}

func (h *Handler) RegisterRoutes(app fiber.Router, requireAuth fiber.Handler, requireAdmin fiber.Handler) {
	admin := app.Group("/admin", requireAuth, requireAdmin)

	admin.Get("/logs/stats", h.GetStats)
	admin.Post("/logs/retry", h.RetryUploads)
	admin.Get("/info", h.Info)
}

func (h *Handler) GetStats(c *fiber.Ctx) error {
	stats, err := h.uploadWorker.GetStats()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"stats": stats})
}

func (h *Handler) RetryUploads(c *fiber.Ctx) error {
	if err := h.uploadWorker.RetryFailedPermanent(); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Retry triggered for failed uploads"})
}

func (h *Handler) Info(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"service": "mangosteen",
		"version": "1.0.0",
		"time":    time.Now(),
	})
}