package websocketTests

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
	"user-notification-api/tests/testutils"

	"github.com/stretchr/testify/assert"
	"nhooyr.io/websocket"
)

func TestWebSocketChat(t *testing.T) {
	app := testutils.SetupTestApp(t)

	// Start the server in a goroutine
	go func() {
		err := app.Listen(":3000")
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Failed to start server: %v", err)
		}
	}()

	// Wait for the server to be ready
	time.Sleep(500 * time.Millisecond)

	// Get tokens for two users
	token1 := testutils.GetValidToken(t, app, "user")
	token2 := testutils.GetValidToken(t, app, "user")

	url := "ws://localhost:3000/ws"

	// Prepare headers for client 1
	headers1 := http.Header{}
	headers1.Set("Authorization", "Bearer "+token1)

	// Prepare headers for client 2
	headers2 := http.Header{}
	headers2.Set("Authorization", "Bearer "+token2)

	// Connect first client
	c1, _, err := websocket.Dial(context.Background(), url, &websocket.DialOptions{
		HTTPHeader: headers1,
	})
	assert.NoError(t, err)
	defer c1.Close(websocket.StatusNormalClosure, "")

	// Connect second client
	c2, _, err := websocket.Dial(context.Background(), url, &websocket.DialOptions{
		HTTPHeader: headers2,
	})
	assert.NoError(t, err)
	defer c2.Close(websocket.StatusNormalClosure, "")

	// Consume welcome message on c1
	_, data, err := c1.Read(context.Background())
	assert.NoError(t, err)
	var welcome map[string]string
	err = json.Unmarshal(data, &welcome)
	assert.NoError(t, err)
	assert.Equal(t, "Welcome to global chat!", welcome["message"], "Expected welcome message on c1")

	// Consume welcome message on c2
	_, data, err = c2.Read(context.Background())
	assert.NoError(t, err)
	err = json.Unmarshal(data, &welcome)
	assert.NoError(t, err)
	assert.Equal(t, "Welcome to global chat!", welcome["message"], "Expected welcome message on c2")

	// Send message from c1
	err = c1.Write(context.Background(), websocket.MessageText, []byte(`{"message":"Hello from user 1!"}`))
	assert.NoError(t, err)

	// Receive message on c2
	_, data, err = c2.Read(context.Background())
	assert.NoError(t, err)
	var received map[string]interface{}
	err = json.Unmarshal(data, &received)
	assert.NoError(t, err)
	assert.Equal(t, "Hello from user 1!", received["message"], "Expected broadcast message")
	assert.NotEmpty(t, received["user_id"], "Expected user_id in message")

	// Receive echo on c1
	_, data, err = c1.Read(context.Background())
	assert.NoError(t, err)
	err = json.Unmarshal(data, &received)
	assert.NoError(t, err)
	assert.Equal(t, "Hello from user 1!", received["message"], "Expected echo message")

	// Clean up
	assert.NoError(t, app.Shutdown())
}
