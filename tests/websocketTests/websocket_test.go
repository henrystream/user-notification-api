package websocketTests

import (
	"net/http"
	"testing"
	"time"
	"user-notification-api/tests/testutils"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestWebSocketWelcome(t *testing.T) {
	app := testutils.SetupTestApp(t)
	go func() {
		if err := app.Listen(":3000"); err != nil {
			t.Logf("Server failed: %v", err)
		}
	}()
	time.Sleep(100 * time.Millisecond)

	token := testutils.GetValidToken(t, app, "user")
	wsURL := "ws://localhost:3000/ws"
	dialer := websocket.Dialer{}
	header := http.Header{"Authorization": []string{"Bearer " + token}}
	conn, _, err := dialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	_, message, err := conn.ReadMessage()
	assert.NoError(t, err, "Failed to read message")
	assert.Equal(t, "Welcome back!", string(message), "Expected welcome message")
}
