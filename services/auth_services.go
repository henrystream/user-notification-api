package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
	"user-notification-api/models"

	"github.com/dgrijalva/jwt-go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pquerna/otp/totp"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	db                *pgxpool.Pool
	redisClient       *redis.Client
	jwtSecret         = []byte("secret-key")
	GoogleOauthConfig = &oauth2.Config{
		ClientID:     "468907561667-g2pvuq6llu3l6bbd4egqit1noqih9a3i.apps.googleusercontent.com",
		ClientSecret: "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",
		RedirectURL:  "http://localhost:3000/auth/google/callback",
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile"},
		Endpoint:     google.Endpoint,
	}
)

func DB() *pgxpool.Pool {
	return db
}

func InitDB() { // Runtime initialization
	var err error
	connStr := "postgres://postgres:password123@localhost:5432/userdb?sslmode=disable"
	if db != nil {
		db.Close()
	}
	db, err = pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Unable to connect to PostgreSQL: %v", err)
	}
	if db == nil {
		log.Fatal("Database pool is nil")
	}

	_, err = db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			password TEXT,
			role TEXT NOT NULL,
			google_id TEXT UNIQUE,
			totp_secret TEXT
		)
	`)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42P07" {
			// Ignore "relation already exists"
		} else {
			log.Fatalf("Failed to create table: %v", err)
		}
	}

	if redisClient != nil {
		redisClient.Close()
	}
	redisClient = redis.NewClient(&redis.Options{Addr: "redis:6379"})

	kafkaBroker := os.Getenv("KAFKA_BROKER")
	if kafkaBroker == "" {
		kafkaBroker = "localhost:9092" // Default for Docker
	}
	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    "user-registration",
		Balancer: &kafka.LeastBytes{},
	}
	conn, err := kafka.Dial("leader", kafkaBroker)
	if err != nil {
		log.Printf("Failed to connect to Kafka: %v, using mock", err)
		kafkaWriter = &mockKafkaWriter{}
	} else {
		defer conn.Close()
		log.Println("Connected to Kafka successfully")
	}

}

// Declare kafkaWriter at package level
var kafkaWriter KafkaWriterInterface

// KafkaWriterInterface defines the methods we need
type KafkaWriterInterface interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

// mockKafkaWriter for fallback
type mockKafkaWriter struct{}

func (m *mockKafkaWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	log.Println("Mock Kafka: Skipping message write")
	return nil
}
func (m *mockKafkaWriter) Close() error { return nil }

// KafkaWriter returns the writer instance
func KafkaWriter() KafkaWriterInterface {
	return kafkaWriter
}

func InitDBTest() {
	var err error
	connStr := "postgres://postgres:password123@localhost:5432/userdb?sslmode=disable"
	if db != nil {
		db.Close()
	}
	db, err = pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Unable to connect to PostgreSQL: %v", err)
	}
	if db == nil {
		log.Fatal("Database pool is nil")
	}

	_, err = db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			password TEXT,
			role TEXT NOT NULL,
			google_id TEXT UNIQUE,
			totp_secret TEXT
		)
	`)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42P07" {
			// Ignore "relation already exists" error
		} else {
			log.Fatalf("Failed to create table: %v", err)
		}
	}

	_, err = db.Exec(context.Background(), "TRUNCATE TABLE users RESTART IDENTITY")
	if err != nil {
		log.Fatalf("Failed to truncate table: %v", err)
	}

	if redisClient != nil {
		redisClient.Close()
	}
	redisClient = redis.NewClient(&redis.Options{Addr: "localhost:6379"})
}

func JWTSecret() []byte {
	return jwtSecret
}

func Register(user models.User) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Hash error: %v", err)
		return "", err
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "UserNotificationAPI",
		AccountName: user.Email,
	})
	if err != nil {
		log.Printf("TOTP generation error: %v", err)
		return "", err
	}
	_, err = db.Exec(context.Background(), `
		INSERT INTO users (email, password, role, totp_secret) VALUES ($1, $2, $3, $4)
	`, user.Email, hashed, user.Role, key.Secret())
	if err != nil {
		log.Printf("DB insert error: %v", err)
		return "", fmt.Errorf("DB insert error: %v", err)
	}

	msg := fmt.Sprintf("New User Registered: Email=%s", user.Email)
	log.Println(msg)
	return key.Secret(), nil
}

func Login(email, password string) (string, error) {
	var user models.User
	var totpSecret string
	log.Printf("Querying user: %s", email)

	err := db.QueryRow(context.Background(), `
		SELECT id, email, password, role,totp_secret 
		FROM users WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.Password, &user.Role, &totpSecret)
	if err != nil {
		log.Printf("DB query error for %s: %v", email, err)
		return "", err
	}
	log.Printf("Found user: ID=%d, Email=%s, HasPassword=%v", user.ID, user.Email, user.Password != "")
	if user.Password != "" {
		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
		if err != nil {
			log.Printf("Password mismatch for %s: %v", email, err)
			return "", err
		}
		log.Printf("Password mismatch for %s", email)
	} else {
		log.Printf("No password for %s, assuming OAuth user", email)
		return "", errors.New("OAuth user, use Google login")
	}

	partialToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":   user.ID,
		"role": user.Role,
		"exp":  time.Now().Add(5 * time.Minute).Unix(),
		"2fa":  false,
	})

	tokenString, err := partialToken.SignedString(jwtSecret)
	if err != nil {
		log.Printf("Token signing error: %v", err)
		return "", err
	}
	err = redisClient.Set(context.Background(), tokenString, user.ID, 5*time.Minute).Err()
	if err != nil {
		log.Printf("Redis set error: %v", err)
	}
	log.Printf("Generated partial token: %s", tokenString)
	return tokenString, nil
}

func GoogleLogin(code string) (string, error) {
	log.Printf("Starting Google login with code: %s", code)
	token, err := GoogleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("OAuth exchane error:%v", err)
		return "", err
	}
	log.Printf("OAuth token received: %v", token)
	client := GoogleOauthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		log.Printf("Failed to get user info: %v", err)
		return "", err
	}
	defer resp.Body.Close()
	log.Printf("User info response status: %d", resp.StatusCode)
	var userInfo struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		log.Printf("Failed to decode user info: %v", err)
		return "", err
	}
	log.Printf("User info: ID=%s, Email=%s", userInfo.ID, userInfo.Email)

	//check if user exists, or create one
	var userID int
	err = db.QueryRow(context.Background(), `
	SELECT id FROM users WHERE google_id=$1
	`, userInfo.ID).Scan(&userID)

	if err != nil {
		if err == pgx.ErrNoRows {
			//New user
			log.Printf("Registering new Google user: %s", userInfo.Email)
			key, err := totp.Generate(totp.GenerateOpts{
				Issuer:      "UserNotificationAPI",
				AccountName: userInfo.Email,
			})
			if err != nil {
				log.Printf("TOTP generation error: %v", err)
				return "", err
			}
			err = db.QueryRow(context.Background(), `
INSERT INTO users (email,role,google_id,totp_secret) VALUES ($1, $2, $3, $4) RETURNING id
`, userInfo.Email, "user", userInfo.ID, key.Secret()).Scan(&userID)
			if err != nil {
				log.Printf("Failed to insert OAuth user: %v", err)
				return "", err
			}
			log.Printf("Generated TOTP secret for %s: %s", userInfo.Email, key.Secret())
		} else {
			log.Printf("DB query error for Google ID %s: %v", userInfo.ID, err)
			return "", err
		}
	}
	log.Printf("User ID: %d", userID)

	//partial token required 2FA
	partialToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":   userID,
		"role": "user",
		"exp":  time.Now().Add(5 * time.Minute).Unix(),
		"2fa":  false,
	})

	tokenString, err := partialToken.SignedString(jwtSecret)
	if err != nil {
		log.Printf("Token signing error: %v", err)
		return "", err
	}
	err = redisClient.Set(context.Background(), tokenString, userID, 5*time.Minute).Err()
	if err != nil {
		log.Printf("Redis set error: %v", err)
	}
	log.Printf("Generated partial toke: %s", tokenString)
	return tokenString, nil
}

func Verify2FA(tokenString, totpCode string) (string, error) {
	token, err := jwt.Parse(tokenString,
		func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
	if err != nil || !token.Valid {
		log.Printf("Invalid partial token: %v", err)
		return "", err
	}
	claims := token.Claims.(jwt.MapClaims)
	if claims["2fa"].(bool) {
		log.Printf("Token already 2FA verified")
		return tokenString, nil
	}
	userID := int(claims["id"].(float64))

	var totpSecret string
	err = db.QueryRow(context.Background(), `
			SELECT totp_secret FROM users WHERE id= $1
			`, userID).Scan(&totpSecret)
	log.Printf("totp from DB %v", totpSecret)
	if err != nil {
		log.Printf("DB query error for user %d: %v", userID, err)
		return "", err
	}
	valid := totp.Validate(totpCode, totpSecret)
	if !valid {
		log.Printf("Invalid TOTP code for user %d", userID)
		return "", errors.New("invalid 2FA code")
	}
	fullToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":   userID,
		"role": claims["role"].(string),
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
		"2fa":  true,
	})
	fullTokenString, err := fullToken.SignedString(jwtSecret)
	if err != nil {
		log.Printf("Token signing error: %v", err)
		return "", err
	}
	err = redisClient.Set(context.Background(), fullTokenString, userID, 24*time.Hour).Err()
	if err != nil {
		log.Printf("Redis set error: %v", err)
	}
	log.Printf("Generated full toke: %s", fullTokenString)
	return fullTokenString, nil
}

func generateToken(userID int, role string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":   userID,
		"role": role,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		log.Printf("Token signing error: %v", err)
		return "", err
	}
	err = redisClient.Set(context.Background(), tokenString, userID, time.Hour*24).Err()
	if err != nil {
		log.Printf("Redis set error: %v", err)
	}
	return tokenString, nil
}
