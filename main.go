package main

import (
	"time"
	"user-notification-api/handlers"
	"user-notification-api/middleware"
	"user-notification-api/services"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/fiber/v2"
)

func main() {
	services.InitDB()
	go services.ProcessJobs()

	app := fiber.New()
	app.Use(middleware.Logging())
	app.Use(middleware.RateLimit(100, time.Minute))

	prometheus := fiberprometheus.New("user_notification_api")
	prometheus.RegisterAt(app, "/metrics")
	app.Use(prometheus.Middleware)

	handlers.Setuproutes(app) // Public
	protected := app.Group("", middleware.JWTAuth())
	handlers.SetupUserRoutes(protected)
	handlers.SetupWebSocketRoutes(protected)

	app.Listen(":3000")
}
