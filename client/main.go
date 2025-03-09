package main

import (
	"context"
	"log"
	pb "user-notification-api/proto"

	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewNotificationServiceClient(conn)
	req := &pb.NotificationRequest{
		Email:   "testuser@example.com",
		Subject: "Test gRPC Notification",
		Message: "Hello from gRPC!",
	}

	resp, err := client.SendNotification(context.Background(), req)
	if err != nil {
		log.Fatalf("Failed to send notification: %v", err)
	}
	if resp.Success {
		log.Println("Notification sent successfully")
	} else {
		log.Printf("Notification failed: %s", resp.Error)
	}
}
