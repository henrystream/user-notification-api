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
	/*for {
	msg, err := reader.ReadMessage(context.Background())
			if err != nil {
				log.Printf("Failed to read Kafka message: %v", err)
				continue
			}*/

	/*var user struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.Unmarshal(msg.Value, &user); err != nil {
		log.Printf("Failed to unmarshal Kafka message: %v", err)
		continue
	}*/

	// Send email (using Gmail SMTP as an example)
	/*err = sendEmail(user.Email, "Welcome to User Notification API",
		"Hi,\n\nYou’ve registered with role: "+user.Role+".\n\nThanks,\nTeam")
	if err != nil {
		log.Printf("Failed to send email to %s: %v", user.Email, err)
	} else {
		log.Printf("Sent email to %s", user.Email)
	}*/
	//}
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
