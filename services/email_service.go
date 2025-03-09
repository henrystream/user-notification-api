package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	pb "user-notification-api/proto"

	"github.com/segmentio/kafka-go"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// RegistrationMessage defines the structure of Kafka messages
type RegistrationMessage struct {
	Email   string `json:"email"`
	Subject string `json:"subject,omitempty"` // Optional, with default if missing
	Message string `json:"message,omitempty"` // Optional, with default if missing
}

var kafkaReader *kafka.Reader

// NotificationServer implements the gRPC NotificationService
type NotificationServer struct {
	pb.UnimplementedNotificationServiceServer
}

func (s *NotificationServer) SendNotification(ctx context.Context, req *pb.NotificationRequest) (*pb.NotificationResponse, error) {
	err := sendEmail(req.Email, req.Subject, req.Message)
	if err != nil {
		log.Printf("gRPC SendNotification failed for %s: %v", req.Email, err)
		return &pb.NotificationResponse{Success: false, Error: err.Error()}, nil
	}
	log.Printf("gRPC Sent notification to %s", req.Email)
	return &pb.NotificationResponse{Success: true, Error: ""}, nil
}

// KafkaWriter returns the writer instance
func KafkaWriter() *kafka.Writer {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		broker = "localhost:9092"
		log.Println("KAFKA_BROKER not set, defaulting to kafka:9092")
	}
	writer := &kafka.Writer{
		Addr:     kafka.TCP(broker),
		Topic:    "user-registration",
		Balancer: &kafka.LeastBytes{},
	}
	conn, err := kafka.Dial("tcp", broker)
	if err != nil {
		log.Printf("Failed to connect to Kafka: %v", err)
	} else {
		log.Println("Connected to Kafka successfully")
		conn.Close()
	}
	return writer
}

func StartEmailConsumer() {
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		broker = "localhost:9092"
		log.Println("KAFKA_BROKER not set, defaulting to localhost:9092")
	}
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{broker},
		Topic:    "user-registration",
		GroupID:  "email-consumer-group",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	// Ensure reader is not closed prematurely
	defer func() {
		if err := r.Close(); err != nil {
			log.Printf("Failed to close Kafka reader: %v", err)
		}
	}()

	for i := 0; i < 5; i++ {
		_, err := r.FetchMessage(context.Background())
		if err != nil {
			log.Printf("Failed to connect to Kafka for consumer (attempt %d): %v", i, err)
			time.Sleep(5 * time.Second)
			continue
		}
		log.Println("Connected to Kafka for consumer successfully")
		break
		/*conn, err := kafka.Dial("tcp", broker)
		if err == nil {
			log.Println("Connected to Kafka for consumer successfully")
			conn.Close()
			break
		}
		log.Printf("Failed to connect to Kafka for consumer (attempt %d): %v", i+1, err)
		if i == 4 {
			log.Println("Giving up on Kafka consumer after 5 attempts")
			return
		}
		time.Sleep(5 * time.Second)*/
	}

	/*kafkaReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{broker},
		Topic:    "user-registration",
		GroupID:  "email-consumer-group",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})*/

	defer r.Close()
	log.Println("Starting Kafka email consumer")
	for {
		msg, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Failed to read Kafka message: %v", err)
			continue
		}

		var regMsg RegistrationMessage
		if err := json.Unmarshal(msg.Value, &regMsg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		// Provide defaults if subject or message are missing
		subject := regMsg.Subject
		if subject == "" {
			subject = "Welcome to User Notification API"
		}
		message := regMsg.Message
		if message == "" {
			message = "Thanks for registering! Enjoy our services."
		}

		if err := sendEmail(regMsg.Email, subject, message); err != nil {
			log.Printf("Failed to send email to %s: %v", regMsg.Email, err)
		} else {
			log.Printf("Sent welcome email to %s", regMsg.Email)
		}
	}
}

func sendEmail(toEmail, subject, message string) error {
	apiKey := os.Getenv("SENDGRID_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("SENDGRID_API_KEY not set")
	}

	from := mail.NewEmail("User Notification API", "no-reply@example.com")
	to := mail.NewEmail("", toEmail)
	content := mail.NewContent("text/plain", message)
	m := mail.NewV3MailInit(from, subject, to, content)

	client := sendgrid.NewSendClient(apiKey)
	response, err := client.Send(m)
	if err != nil {
		return err
	}
	if response.StatusCode >= 400 {
		return fmt.Errorf("email send failed with status: %d", response.StatusCode)
	}
	return nil
}
