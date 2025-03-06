package handlers

import (
	"user-notification-api/services"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// handlers/websocket.go
func SetupWebSocketRoutes(app fiber.Router) { // Change *fiber.App to fiber.Router
	go services.BroadcastMessage()
	app.Get("/ws", websocket.New(services.HandleWebSocket))
}
