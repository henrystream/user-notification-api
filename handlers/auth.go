package handlers

import (
	"context"
	"log"
	"user-notification-api/services"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
)

func Setuproutes(app *fiber.App) {
	app.Post("/register", Register)
	app.Post("/login", Login)
	app.Post("/2fa", Verify2FAHandler)
	app.Get("/auth/google", GoogleAuthRedirect)
	app.Get("/auth/google/callback", GoogleAuthCallback)
}

func Register(c *fiber.Ctx) error {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to hash password"})
	}

	totpSecret, err := totp.Generate(totp.GenerateOpts{Issuer: "YourApp", AccountName: input.Email})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to generate TOTP"})
	}

	var userID int
	err = services.DB().QueryRow(c.Context(), `
		INSERT INTO users (email, password, role, totp_secret)
		VALUES ($1, $2, $3, $4) RETURNING id`,
		input.Email, hashedPassword, input.Role, totpSecret.Secret()).Scan(&userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to register user"})
	}

	log.Printf("Registered user: Email=%s", input.Email)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"totp_secret": totpSecret.Secret()})
}

func Login(c *fiber.Ctx) error {
	var creds struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&creds); err != nil {
		log.Printf("Parse error: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input"})
	}
	log.Printf("Login attempt: Email=%s", creds.Email)
	token, err := services.Login(creds.Email, creds.Password)
	if err != nil {
		log.Printf("Login failed for %s: %v", creds.Email, err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Login failed"})
	}
	log.Printf("Login successful for %s, partial token: %s", creds.Email, token)
	return c.JSON(fiber.Map{"token": token})
}

func GoogleAuthRedirect(c *fiber.Ctx) error {
	url := services.GoogleOauthConfig.AuthCodeURL("state", oauth2.AccessTypeOffline)
	return c.Redirect(url)
}

func GoogleAuthCallback(c *fiber.Ctx) error {
	code := c.Query("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Missing code"})
	}
	token, err := services.GoogleLogin(code)
	if err != nil {
		log.Printf("Google login error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Google login failed"})
	}
	var userID int
	tokenObj, _ := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return services.JWTSecret(), nil
	})
	claims := tokenObj.Claims.(jwt.MapClaims)
	userID = int(claims["id"].(float64))
	var totpSecret string
	err = services.DB().QueryRow(context.Background(), `
	SELECT totp_secret FROM users WHERE id=$1
	`, userID).Scan(&totpSecret)
	if err != nil {
		log.Printf("Failed to fetch TOTP secret: %v", err)
	}
	return c.JSON(fiber.Map{"token": token, "totp_secret": totpSecret})
}

func Verify2FAHandler(c *fiber.Ctx) error {
	var input struct {
		Token    string `json:"token"`
		TOTPCode string `json:"totp_code"`
	}

	if err := c.BodyParser(&input); err != nil {
		log.Printf("Parser error: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input"})
	}
	token, err := services.Verify2FA(input.Token, input.TOTPCode)
	if err != nil {
		log.Printf("2FA verificationn failed:  %v", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid 2FA code"})
	}
	return c.JSON(fiber.Map{"token": token})
}
