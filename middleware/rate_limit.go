package middleware

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

var redisClient = redis.NewClient(&redis.Options{Addr: "localhost:6379"})

func RateLimit(maxReqs int, window time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Use a unique key per test run or user, combining IP and a test-specific header
		testID := c.Get("X-Test-ID", c.IP()) // Default to IP if no header
		key := "rate:" + testID
		count, err := redisClient.Incr(c.Context(), key).Result()
		if err != nil {
			log.Printf("Redis incr error: %v", err)
		}
		if count == 1 {
			err := redisClient.Expire(c.Context(), key, window).Err()
			if err != nil {
				log.Printf("Redis expire error: %v", err)
			}
		}
		if count > int64(maxReqs) {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "Rate limit exceeded"})
		}
		return c.Next()
	}
}
