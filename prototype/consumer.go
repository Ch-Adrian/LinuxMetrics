package main

import (
    "context"
    "fmt"
    "log"
    "github.com/segmentio/kafka-go"
)

func main() {
    // Set up a Kafka reader
    reader := kafka.NewReader(kafka.ReaderConfig{
        Brokers:  []string{"localhost:29092"},
        Topic:    "mytopic",
        GroupID:  "mygroup",
        MinBytes: 10e3, // 10KB
        MaxBytes: 10e6, // 10MB
    })

    // Process Kafka messages
	log.Print("Processing kafka messages: ")
    for {
        m, err := reader.ReadMessage(context.Background())
        if err != nil {
            log.Printf("Error reading message: %v", err)
            continue
        }

		fmt.Printf("message at offset %d: %s = %s\n", m.Offset, string(m.Key), string(m.Value))

    }

	if err := reader.Close(); err != nil {
		log.Fatal("failed to close reader:", err)
	}

}