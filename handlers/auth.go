package handlers

import (
	"context"

	"encoding/json"
	"log"
	"time"
	"user-notification-api/services"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp/totp"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/segmentio/kafka-go"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
)

var jwtSecret = []byte("your-secret-key") // Replace with env var in production

// Custom metrics
var (
	loginCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "login_attempts_total",
			Help: "Total number of login attempts",
		},
		[]string{"status"}, // Success or failure
	)
)

func init() {
	prometheus.MustRegister(loginCounter)
}

type User struct {
	Email      string
	Password   string
	Role       string
	TOTPSecret string
}

func Setuproutes(app *fiber.App) {
	app.Post("/register", Register)
	app.Post("/login", Login)
	app.Get("/admin", AdminRoute)
	app.Post("/2fa", Verify2FA)
	app.Get("/auth/google", GoogleAuthRedirect)
	app.Get("/auth/google/callback", GoogleAuthCallback)
	app.Get("/ws", WebSocketChat)
}

// Register handles user registration
func Register(c *fiber.Ctx) error {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid input"})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to hash password"})
	}

	key, err := totp.Generate(totp.GenerateOpts{Issuer: "UserNotificationAPI", AccountName: input.Email})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate TOTP"})
	}

	db := services.DB()
	if db == nil {
		return c.Status(500).JSON(fiber.Map{"error": "Database not available"})
	}
	//defer db.Close()
	_, err = db.Exec(context.Background(),
		"INSERT INTO users (email, password, role, totp_secret) VALUES ($1, $2, $3, $4)",
		input.Email, string(hashedPassword), input.Role, key.Secret())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to save user: " + err.Error()})
	}

	msg := services.RegistrationMessage{
		Email:   input.Email,
		Subject: "Welcome to User Notification API",
		Message: "Thanks for registering! Enjoy our services.",
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal Kafka message: %v", err)
	} else {
		err = services.KafkaWriter().WriteMessages(context.Background(), kafka.Message{Value: msgBytes})
		if err != nil {
			log.Printf("Failed to send Kafka message: %v", err)
		} else {
			log.Printf("Sent Kafka message for %s", input.Email)
		}
	}

	return c.Status(201).JSON(fiber.Map{"totp_secret": key.Secret()})
}

// Login handles user login, issuing a partial JWT
func Login(c *fiber.Ctx) error {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&input); err != nil {
		log.Printf("Login: Invalid input - %v", err)
		return c.Status(400).JSON(fiber.Map{"error": "Invalid input"})
	}

	log.Printf("Login attempt: Email=%s", input.Email)

	redisClient := services.InitRedis()
	if redisClient != nil {
		ctx := context.Background()
		attempts, err := redisClient.Incr(ctx, "login_attempts:"+input.Email).Result()
		if err != nil {
			log.Printf("Redis incr error: %v", err)
		} else if attempts > 5 {
			return c.Status(429).JSON(fiber.Map{"error": "Too many login attempts"})
		}
	}

	tokenString, err := services.Login(input.Email, input.Password)
	if redisClient != nil {
		err = redisClient.Set(context.Background(), tokenString, input.Email, 5*time.Minute).Err()
		if err != nil {
			log.Printf("Redis set error: %v", err)
		}
	}

	log.Printf("Generated partial token: %s", tokenString)
	return c.JSON(fiber.Map{"token": tokenString})
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

// Verify2FA completes 2FA and issues a full JWT
func Verify2FA(c *fiber.Ctx) error {
	var input struct {
		Token    string `json:"token"`
		TOTPCode string `json:"totp_code"`
	}
	if err := c.BodyParser(&input); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid input"})
	}

	log.Println("token:", input.Token)
	log.Println("code:", input.TOTPCode)
	fullTokenString, err := services.Verify2FA(input.Token, input.TOTPCode)
	if err != nil {
		log.Println(err)
	}
	return c.JSON(fiber.Map{"token": fullTokenString})
}

// AdminRoute restricts access to admins
func AdminRoute(c *fiber.Ctx) error {
	tokenString := c.Get("Authorization")
	if tokenString == "" || len(tokenString) < 8 || tokenString[:7] != "Bearer " {
		return c.Status(401).JSON(fiber.Map{"error": "Missing or invalid Authorization header"})
	}
	tokenString = tokenString[7:]

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return c.Status(401).JSON(fiber.Map{"error": "Invalid token"})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !claims["2fa"].(bool) || claims["role"] != "admin" {
		return c.Status(403).JSON(fiber.Map{"error": "Unauthorized"})
	}

	return c.JSON(fiber.Map{"message": "Welcome, admin!"})
}

// WebSocketChat handles chat connections
func WebSocketChat(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		tokenString := c.Get("Authorization")
		if tokenString == "" || len(tokenString) < 8 || tokenString[:7] != "Bearer " {
			return c.Status(401).JSON(fiber.Map{"error": "Missing or invalid Authorization header"})
		}
		tokenString = tokenString[7:]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			return c.Status(401).JSON(fiber.Map{"error": "Invalid token"})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !claims["2fa"].(bool) {
			return c.Status(403).JSON(fiber.Map{"error": "Unauthorized"})
		}

		return websocket.New(func(conn *websocket.Conn) {
			defer conn.Close()
			for {
				msgType, msg, err := conn.ReadMessage()
				if err != nil {
					log.Printf("WebSocket read error: %v", err)
					return
				}
				log.Printf("Received: %s", msg)
				if err := conn.WriteMessage(msgType, msg); err != nil {
					log.Printf("WebSocket write error: %v", err)
					return
				}
			}
		})(c)
	}
	return c.Status(400).JSON(fiber.Map{"error": "Not a WebSocket request"})
}
