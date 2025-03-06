package middleware

import (
	"log"
	"strings"
	"user-notification-api/services"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
)

func JWTAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			log.Println("No Bearer token provided")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "No token"})
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			return services.JWTSecret(), nil
		})
		if err != nil || !token.Valid {
			log.Printf("JWT parse error: %v: %v", err)
			return c.Status(fiber.StatusUnauthorized).JSON(
				fiber.Map{"error": "Invalid token"})
		}
		claims := token.Claims.(jwt.MapClaims)
		if !claims["2fa"].(bool) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "2FA required"})
		}
		c.Locals("userID", int(claims["id"].(float64)))
		c.Locals("role", claims["role"].(string))
		return c.Next()
	}
}

func Role(role string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := c.Locals("role").(string)

		if userRole != role {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Access denied"})
		}
		return c.Next()

	}
}
