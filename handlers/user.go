package handlers

import (
	"user-notification-api/middleware"

	"github.com/gofiber/fiber/v2"
)

func SetupUserRoutes(app fiber.Router) {
	app.Get("/profile", Profile)
	app.Get("/admin", middleware.JWTAuth(), middleware.Role("admin"), Admin)
	app.Get("/user-data", middleware.Role("user"), UserData)
}

func Profile(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"id": c.Locals("userID"), "role": c.Locals("role")})
}
func Admin(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "Welcome, Admin"})
}

func UserData(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"message": "User data"})
}
