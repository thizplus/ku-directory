package websocket

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	websocketManager "gofiber-template/infrastructure/websocket"
	"gofiber-template/pkg/logger"
	"gofiber-template/pkg/utils"
)

type WebSocketHandler struct{}

func NewWebSocketHandler() *WebSocketHandler {
	return &WebSocketHandler{}
}

func (h *WebSocketHandler) WebSocketUpgrade(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

func (h *WebSocketHandler) HandleWebSocket(c *websocket.Conn) {
	var userID uuid.UUID
	var roomID string

	// Try to get user from context (set by Optional middleware)
	if userContext := c.Locals("user"); userContext != nil {
		if user, ok := userContext.(*utils.UserContext); ok {
			userID = user.ID
		}
	}

	// If no user context, generate anonymous user ID
	if userID == uuid.Nil {
		userID = uuid.New()
		logger.WebSocket("anonymous_connected", "Anonymous user connected", map[string]interface{}{"user_id": userID.String()})
	} else {
		logger.WebSocket("authenticated_connected", "Authenticated user connected", map[string]interface{}{"user_id": userID.String()})
	}

	roomID = c.Query("room", "")

	websocketManager.Manager.RegisterClient(c, userID, roomID)

	defer func() {
		websocketManager.Manager.UnregisterClient(c)
	}()

	for {
		messageType, message, err := c.ReadMessage()
		if err != nil {
			logger.WebSocketError("read_message", "WebSocket read error", err, map[string]interface{}{"user_id": userID.String()})
			break
		}

		websocketManager.HandleWebSocketMessage(c, messageType, message)
	}
}