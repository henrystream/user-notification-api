package workers

import (
	"log"
	"net/smtp"

	"github.com/segmentio/kafka-go"
)

func StartEmailWorker() {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{"localhost:9092"},
		Topic:    "user-registration",
		GroupID:  "email-worker-group",
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	defer reader.Close()

	log.Println("Email worker started")

}

func sendEmail(to, subject, body string) error {
	// Replace with your SMTP credentials (e.g., Gmail or Mailtrap)
	auth := smtp.PlainAuth("", "henrystreamhenry@gmail.com", "@Tr34me7", "smtp.gmail.com")
	msg := []byte("To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body + "\r\n")

	err := smtp.SendMail("smtp.gmail.com:587", auth, "henrystreamhenry@gmail.com", []string{to}, msg)
	return err
}
