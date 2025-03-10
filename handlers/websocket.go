package handlers

import (
	"log"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	clients   = make(map[*websocket.Conn]int) // Map of WebSocket connections to user IDs
	clientsMu sync.Mutex                      // Mutex for thread-safe client management

	wsConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "websocket_connections_active",
			Help: "Number of active WebSocket connections",
		},
	)
)

func init() {
	prometheus.MustRegister(wsConnections)
}

func WebSocketHandler(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

func SetupWebSocketRoutes(app fiber.Router) {
	app.Get("/ws", WebSocketHandler, websocket.New(func(c *websocket.Conn) {
		// Extract user ID from JWT (set by middleware.JWTAuth)
		userID := c.Locals("user_id").(int)

		// Register client
		clientsMu.Lock()
		clients[c] = userID
		wsConnections.Inc() // Increment active connections
		clientsMu.Unlock()
		log.Printf("User %d connected to WebSocket", userID)

		// Send welcome message
		err := c.WriteJSON(map[string]string{"message": "Welcome to global chat!"})
		if err != nil {
			log.Printf("Failed to send welcome message to user %d: %v", userID, err)
			return
		}

		// Handle incoming messages
		defer func() {
			clientsMu.Lock()
			delete(clients, c)
			wsConnections.Dec() // Decrement active connections
			clientsMu.Unlock()
			c.Close()
			log.Printf("User %d disconnected from WebSocket", userID)
		}()

		for {
			var msg map[string]string
			err := c.ReadJSON(&msg)
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					break
				}
				log.Printf("Error reading message from user %d: %v", userID, err)
				break
			}

			// Broadcast message to all connected clients
			clientsMu.Lock()
			for client, id := range clients {
				err = client.WriteJSON(map[string]interface{}{
					"user_id": userID,
					"message": msg["message"],
				})
				if err != nil {
					log.Printf("Failed to send message to user %d: %v", id, err)
					client.Close()
					delete(clients, client)
				}
			}
			clientsMu.Unlock()
		}
	}))
}
