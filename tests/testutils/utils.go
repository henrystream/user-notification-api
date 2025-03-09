package testutils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"user-notification-api/handlers"
	"user-notification-api/services"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	//"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
)

type DBInterface interface {
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Close()
}

// MockDB mocks the pgxpool.Pool methods used in handlers
type MockDB struct {
	ExecFunc     func(ctx context.Context, sql string, args ...interface{}) (int64, error)
	QueryRowFunc func(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

func (m *MockDB) Exec(ctx context.Context, sql string, args ...interface{}) (int64, error) {
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, sql, args...)
	}
	return 1, nil // Default: simulate success
}

func (m *MockDB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	if m.QueryRowFunc != nil {
		return m.QueryRowFunc(ctx, sql, args...)
	}
	return &MockRow{} // Default: empty row
}

func (m *MockDB) Close() {}

// MockRow mocks a pgx.Row for QueryRow
type MockRow struct {
	email, password, role, totp string
}

func (r *MockRow) Scan(dest ...interface{}) error {
	if len(dest) == 4 {
		dest[0] = r.email
		dest[1] = r.password
		dest[2] = r.role
		dest[3] = r.totp
	}
	return nil
}

func SetupTestApp() *fiber.App {
	app := fiber.New()

	// Mock DB
	mockDB := &MockDB{
		ExecFunc: func(ctx context.Context, sql string, args ...interface{}) (int64, error) {
			return 1, nil // Simulate success
		},
		QueryRowFunc: func(ctx context.Context, sql string, args ...interface{}) pgx.Row {
			return &MockRow{}
		},
	}
	services.DBFunc = func() func() services.DBInterface {
		return func() services.DBInterface {
			return mockDB
		}
	}

	// Register routes
	app.Post("/register", handlers.Register)
	app.Post("/login", handlers.Login)
	app.Post("/verify-2fa", handlers.Verify2FA)
	app.Get("/admin", handlers.AdminRoute)
	app.Get("/ws/chat", handlers.WebSocketChat)

	return app
}

/*func SetupTestApp(t *testing.T) *fiber.App {
	services.InitDBTest()
	_, err := services.DB().Exec(context.Background(), "TRUNCATE TABLE users RESTART IDENTITY")
	if err != nil {
		t.Fatalf("Failed to truncate users table: %v", err)
	}

	app := fiber.New()
	if app == nil {
		t.Fatal("Failed to create Fiber app")
	}
	app.Use(middleware.RateLimit(100, time.Minute))
	handlers.Setuproutes(app)

	protected := app.Group("", middleware.JWTAuth())
	handlers.SetupUserRoutes(protected)
	handlers.SetupWebSocketRoutes(protected)
	return app
}*/

func GetValidToken(t *testing.T, app *fiber.App, role string) string {
	if app == nil {
		t.Fatal("App is nil")
	}
	uniqueEmail := fmt.Sprintf("test+%d@example.com", time.Now().UnixNano())
	password := "password123" // Store password for reuse
	payload := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}{
		Email:    uniqueEmail,
		Password: password,
		Role:     role,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal register payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("Register request failed: %v", err)
	}
	if resp == nil {
		t.Fatal("Register response is nil")
	}
	t.Logf("Register status: %d", resp.StatusCode)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	var regResult map[string]string
	err = json.NewDecoder(resp.Body).Decode(&regResult)
	if err != nil {
		t.Fatalf("Failed <nil>Failed to decode register response: %v", err)
	}
	totpSecret := regResult["totp_secret"]

	// Ensure DB commit by waiting briefly
	time.Sleep(100 * time.Millisecond)

	// Login with the same password
	loginPayload := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{
		Email:    uniqueEmail,
		Password: password, // Reuse the same password
	}
	body, err = json.Marshal(loginPayload)
	if err != nil {
		t.Fatalf("Failed to marshal login payload: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, 5000)
	if err != nil {
		t.Fatalf("Login request failed: %v", err)
	}
	if resp == nil {
		t.Fatal("Login response is nil")
	}
	t.Logf("Login status: %d", resp.StatusCode)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var partialResult map[string]string
	err = json.NewDecoder(resp.Body).Decode(&partialResult)
	if err != nil {
		t.Fatalf("Failed to decode login response: %v", err)
	}
	partialToken := partialResult["token"]

	// Complete 2FA
	totpCode, err := totp.GenerateCode(totpSecret, time.Now())
	if err != nil {
		t.Errorf("Error generating totpcode %v", err)
	}
	tfaPayload := map[string]string{
		"token":     partialToken,
		"totp_code": totpCode,
	}
	body, err = json.Marshal(tfaPayload)
	if err != nil {
		t.Fatalf("Failed to marshal 2FA payload: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "/2fa", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, 5000)
	if err != nil {
		t.Fatalf("2FA request failed: %v", err)
	}
	if resp == nil {
		t.Fatal("2FA response is nil")
	}
	t.Logf("2FA status: %d", resp.StatusCode)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]string
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		t.Fatalf("Failed to decode 2FA response: %v", err)
	}
	token := result["token"]
	t.Logf("Generated token: %s", token)
	assert.NotEmpty(t, token, "Token should not be empty")
	return token
}
