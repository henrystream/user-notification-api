package services

import (
	"fmt"
	"sync"

	"github.com/gofiber/websocket/v2"
)

var (
	clients   = make(map[*websocket.Conn]string)
	clientsMu sync.Mutex
	broadcast = make(chan string)
)

func HandleWebSocket(c *websocket.Conn) {
	defer c.Close()
	userID := c.Locals("userID").(int)
	clientsMu.Lock()
	clients[c] = fmt.Sprintf("%d", userID)
	clientsMu.Unlock()

	c.WriteMessage(websocket.TextMessage, []byte("Welcome back!"))
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			clientsMu.Lock()
			delete(clients, c)
			clientsMu.Unlock()
			break
		}
		broadcast <- string(msg)
	}
}

func BroadcastMessage() {
	for msg := range broadcast {
		clientsMu.Lock()
		for client := range clients {
			client.WriteMessage(websocket.TextMessage, []byte(msg))
		}
		clientsMu.Unlock()
	}
}
