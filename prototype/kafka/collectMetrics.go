package kafka

import (
	"context"
	"fmt"
	"github.com/segmentio/kafka-go"
	"log"
	"slices"
	"strconv"
	"time"
)

type KafkaMessage struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ReadResponse struct {
	Messages          []KafkaMessage `json:"messages"`
	ContinuationToken string         `json:"continuationToken"`
}

type ReaderState struct {
	BatchSize         int    `json:"batchSize"`
	ContinuationToken string `json:"continuationToken"`
}

type KafkaClient struct {
	Client  *kafka.Conn
	Address string
}

type KafkaReader struct {
	Client *kafka.Reader
	State  *ReaderState
}

type KafkaWriter struct {
	Client *kafka.Writer
}

func NewKafkaClient(address string) (*KafkaClient, error) {
	conn, err := kafka.Dial("tcp", address)
	if err != nil {
		log.Print("failed to dial leader:", err)
	}
	return &KafkaClient{Client: conn, Address: address}, err
}

func (k *KafkaClient) NewKafkaWriter(topic string) (*KafkaWriter, error) {
	w := &kafka.Writer{
		Addr:                   kafka.TCP(k.Address),
		Topic:                  topic,
		AllowAutoTopicCreation: true,
	}
	return &KafkaWriter{Client: w}, nil
}

func (k *KafkaClient) NewKafkaReader(topic string, state ReaderState) (*KafkaReader, error) {
	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
	}

	if state.ContinuationToken == "" {
		state.ContinuationToken = strconv.FormatInt(time.Now().Unix(), 10)
	}

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{k.Address},
		GroupID: fmt.Sprintf("loader-group-%s", state.ContinuationToken),
		Topic:   topic,
		Dialer:  dialer,
	})
	return &KafkaReader{Client: r, State: &state}, nil
}

func (k *KafkaClient) ListTopics() ([]string, error) {
	partitions, err := k.Client.ReadPartitions()
	if err != nil {
		log.Printf(err.Error())
		return nil, err
	}
	var result []string

	for _, p := range partitions {
		if !slices.Contains(result, p.Topic) {
			result = append(result, p.Topic)
		}
	}
	return result, nil
}

func (k *KafkaWriter) WriteRecord(key, value string) error {
	msg := kafka.Message{
		Key:   []byte(key),
		Value: []byte(value),
		Time:  time.Now(),
	}
	//log.Printf("%s", value)
	err := k.Client.WriteMessages(context.Background(), msg)
	return err
}

func (k *KafkaReader) ReadTopic() ([]KafkaMessage, error) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
	var msg []KafkaMessage
	for i := 0; i < k.State.BatchSize; i++ {
		m, err := k.Client.ReadMessage(ctx)
		if err != nil {
			return msg, err
		}
		msg = append(msg, KafkaMessage{Key: string(m.Key), Value: string(m.Value)})
	}
	return msg, nil
}
