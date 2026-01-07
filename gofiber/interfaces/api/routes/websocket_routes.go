package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"gofiber-template/interfaces/api/middleware"
	websocketHandler "gofiber-template/interfaces/api/websocket"
)

func SetupWebSocketRoutes(app *fiber.App) {
	wsHandler := websocketHandler.NewWebSocketHandler()

	// WebSocket with optional authentication (supports query token for WS connections)
	app.Use("/ws", middleware.OptionalWithQueryToken(), wsHandler.WebSocketUpgrade)
	app.Get("/ws", websocket.New(wsHandler.HandleWebSocket))
}