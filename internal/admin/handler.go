package admin

import (
	"github.com/gofiber/fiber/v2"
	"mangosteen/pkg/worker"
)

type Handler struct {
	uploadWorker *worker.UploadWorker
}

func NewHandler(uploadWorker *worker.UploadWorker) *Handler {
	return &Handler{uploadWorker: uploadWorker}
}

func (h *Handler) RegisterRoutes(app fiber.Router) {
	admin := app.Group("/admin")

	admin.Get("/logs/stats", h.GetStats)
	admin.Post("/logs/retry", h.RetryUploads)
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