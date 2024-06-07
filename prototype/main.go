package main

import (
	"fmt"
	"main/kafka"
)

const KAFKA_IP = "localhost:29092"

func getTopics() {

	kafkaEntry, err := kafka.NewKafkaClient(KAFKA_IP)

	if err != nil {
		fmt.Println(err)
	}

	defer kafkaEntry.Client.Close()

	topics, err := kafkaEntry.ListTopics()
	fmt.Println(topics)

	// writer, err := kafkaEntry.NewKafkaWriter("test-topic")
	
	// if err != nil{
	// 	fmt.Println(err)
	// }

	// writer.WriteToTopic("test-topic", "Hello World")

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{KAFKA_IP},
		Topic:    "test-topic",
		GroupID:  "mygroup",
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	m, err := reader.ReadMessage(context.Background())
	if err != nil {
		log.Printf("Error reading message: %v", err)
		continue
	}

	fmt.Printf("message: ", m)


}

// func ReadHandler(topic string) {

// 	kafkaEntry, err := kafka.NewKafkaClient(KAFKA_IP)

// 	if err != nil {
// 		fmt.Println(err)
// 	}

// 	defer kafkaEntry.Client.Close()

// 	var reader = kafka.ReaderState{BatchSize: 1, ContinuationToken: ""}

// 	reader, err := kafkaEntry.NewKafkaReader(topic, reader)
// 	defer reader.Client.Close()

// 	msg, err := reader.ReadTopic()

// 	fmt.Println(msg)
// }

func main(){
	
	getTopics()

	// ReadHandler("test-topic")
	// getTopics()
}