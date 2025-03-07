package authTests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"user-notification-api/models"
	"user-notification-api/tests/testutils"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestRegister(t *testing.T) {
	app := testutils.SetupTestApp(t)
	payload := models.User{Email: "test@example.com", Password: "password123", Role: "user"}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 5000)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
}

func TestRateLimit(t *testing.T) {
	app := testutils.SetupTestApp(t)
	token := testutils.GetValidToken(t, app, "user")
	// Unique test ID to isolate rate limit counters
	testID := time.Now().UnixNano()
	for i := 0; i < 101; i++ {
		req := httptest.NewRequest(http.MethodGet, "/profile", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-Test-ID", fmt.Sprintf("%d", testID))
		resp, err := app.Test(req)
		assert.NoError(t, err)
		if i < 100 {
			assert.Equal(t, fiber.StatusOK, resp.StatusCode, "Expected OK before limit")
		} else {
			assert.Equal(t, fiber.StatusTooManyRequests, resp.StatusCode, "Expected rate limit")
		}
	}
}

func TestGoogleLogin(t *testing.T) {
	app := testutils.SetupTestApp(t)
	//Mock OAuth flow (simplified; real test would need a mock server)
	req := httptest.NewRequest(http.MethodGet, "/auth/google", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusFound, resp.StatusCode) //Redirect to Google
}

func TestiLoginWith2FA(t *testing.T) {
	app := testutils.SetupTestApp(t)
	token := testutils.GetValidToken(t, app, "user")
	req := httptest.NewRequest(http.MethodGet, "/profile", nil)
	req.Header.Set("Authorization", "Bearer"+token)
	resp, err := app.Test(req, 5000)
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestVerify2FA(t *testing.T) {

}
