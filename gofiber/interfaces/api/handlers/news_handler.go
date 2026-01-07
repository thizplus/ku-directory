package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"gofiber-template/domain/services"
	"gofiber-template/pkg/utils"
)

type NewsHandler struct {
	newsService services.NewsService
}

func NewNewsHandler(newsService services.NewsService) *NewsHandler {
	return &NewsHandler{
		newsService: newsService,
	}
}

// GenerateNewsRequest is the request body for news generation
type GenerateNewsRequest struct {
	PhotoIDs []string `json:"photo_ids"` // Optional: photos for context
	Headings []string `json:"headings"`  // Optional: 4 custom headings
	Tone     string   `json:"tone"`      // formal, friendly, news
	Length   string   `json:"length"`    // short, medium, long
}

// GenerateNews handles news generation from photos
// @Summary Generate news article from photos
// @Tags News
// @Accept json
// @Produce json
// @Param request body GenerateNewsRequest true "Generate request"
// @Success 200 {object} utils.Response
// @Router /api/v1/news/generate [post]
func (h *NewsHandler) GenerateNews(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return utils.UnauthorizedResponse(c, "Not authenticated")
	}

	var req GenerateNewsRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request body", err)
	}

	// Parse photo IDs to UUIDs (optional)
	var photoIDs []uuid.UUID
	for _, idStr := range req.PhotoIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid photo ID: "+idStr, err)
		}
		photoIDs = append(photoIDs, id)
	}

	// Set defaults
	tone := req.Tone
	if tone == "" {
		tone = "formal"
	}
	length := req.Length
	if length == "" {
		length = "medium"
	}

	// Use default headings if not provided
	headings := req.Headings
	if len(headings) != 4 {
		headings = []string{
			"ความเป็นมา",
			"กิจกรรมที่จัด",
			"ผู้เข้าร่วม",
			"สรุป",
		}
	}

	// Create service request
	serviceReq := &services.NewsGenerateRequest{
		PhotoIDs: photoIDs,
		Headings: headings,
		Tone:     tone,
		Length:   length,
	}

	// Generate news
	article, err := h.newsService.GenerateNews(c.Context(), userCtx.ID, serviceReq)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to generate news", err)
	}

	return utils.SuccessResponse(c, "News generated successfully", article)
}
