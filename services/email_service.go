package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/segmentio/kafka-go"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type RegistrationMessage struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

func StartEmailConsumer() {
	kafkaBroker := os.Getenv("KAFKA_BROKER")
	if kafkaBroker == "" {
		kafkaBroker = "localhost:9092"
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{kafkaBroker},
		Topic:    "user-registration",
		GroupID:  "email-consumer-group",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	log.Println("Starting Kafka email consumer")
	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Failed to read Kafka message: %v", err)
			continue
		}

		var regMsg RegistrationMessage
		if err := json.Unmarshal(msg.Value, &regMsg); err != nil {
			log.Printf("Failed to unmarshal message: %v", err)
			continue
		}

		if err := sendEmail(regMsg.Email); err != nil {
			log.Printf("Failed to send email to %s: %v", regMsg.Email, err)
		} else {
			log.Printf("Sent welcome email to %s", regMsg.Email)
		}
	}
}

func sendEmail(toEmail string) error {
	apiKey := os.Getenv("SENDGRID_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("SENDGRID_API_KEY not set")
	}

	from := mail.NewEmail("User Notification API", "no-reply@example.com")
	subject := "Welcome to User Notification API"
	to := mail.NewEmail("", toEmail)
	content := mail.NewContent("text/plain", "Thanks for registering! Enjoy our services.")
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
