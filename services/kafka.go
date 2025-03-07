package services

import (
	"log"

	"github.com/IBM/sarama"
)

const (
	KafkaTopic = "user-registration"
	Brokers    = "localhost:9092"
)

// kafka producer
func produceMessage(message string) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer([]string{Brokers}, config)
	if err != nil {
		log.Fatalf("Failed to start Kafka producer: %v", err)
	}
	defer producer.Close()

	msg := &sarama.ProducerMessage{
		Topic: KafkaTopic,
		Value: sarama.StringEncoder(message),
	}

	_, _, err = producer.SendMessage(msg)
	if err != nil {
		log.Fatalf("Failed to send message: %v", err)
	}
	log.Println("Message sent successfully to Kafka:", message)
}
