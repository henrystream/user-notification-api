package main

import (
	"log"
	"net"
	"time"
	"user-notification-api/handlers"
	"user-notification-api/middleware"
	"user-notification-api/services"

	pb "user-notification-api/proto"

	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc"
)

func main() {
	app := fiber.New()
	app.Use(middleware.Logging())
	app.Use(middleware.RateLimit(100, time.Minute))

	prometheus := fiberprometheus.New("user_notification_api")
	prometheus.RegisterAt(app, "/metrics")
	app.Use(prometheus.Middleware)

	app.Post("/register", handlers.Register)
	app.Post("/login", handlers.Login)
	app.Post("/2fa", handlers.Verify2FA)
	app.Get("/admin", handlers.AdminRoute)
	app.Get("/ws/chat", handlers.WebSocketChat)

	//handlers.Setuproutes(app) // Public
	protected := app.Group("", middleware.JWTAuth())
	//handlers.SetupUserRoutes(protected)
	handlers.SetupWebSocketRoutes(protected)

	// Initialize services
	dbFunc := services.InitDB()
	if dbFunc() == nil {
		log.Println("Warning: Database not initialized; proceeding without DB")
	}

	services.InitRedis()

	// Start Kafka consumer in background
	go services.StartEmailConsumer()

	// Start Fiber server
	go func() {
		// Start gRPC server
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Printf("Failed to listen for gRPC: %v; gRPC server not starting", err)
		} else {
			grpcServer := grpc.NewServer()
			pb.RegisterNotificationServiceServer(grpcServer, &services.NotificationServer{})
			log.Println("gRPC server starting on :50051")
			if err := grpcServer.Serve(lis); err != nil {
				log.Printf("gRPC server failed: %v", err)
			}
		}
	}()

	log.Println("Starting Fiber server on :3000")
	if err := app.Listen(":3000"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	/*app := fiber.New()
	app.Use(middleware.Logging())
	app.Use(middleware.RateLimit(100, time.Minute))

	prometheus := fiberprometheus.New("user_notification_api")
	prometheus.RegisterAt(app, "/metrics")
	app.Use(prometheus.Middleware)

	handlers.Setuproutes(app) // Public
	protected := app.Group("", middleware.JWTAuth())
	handlers.SetupUserRoutes(protected)
	handlers.SetupWebSocketRoutes(protected)

	app.Listen(":3000")*/
}
