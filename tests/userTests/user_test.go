package usertests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"user-notification-api/tests/testutils"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestAdminRoute(t *testing.T) {
	app := testutils.SetupTestApp(t)
	token := testutils.GetValidToken(t, app, "admin")
	t.Logf("Admin token: %s", token) // Debug
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	t.Logf("Admin response status: %d", resp.StatusCode) // Debug
	assert.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode, "Expected admin access")
}
