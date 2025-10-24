package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(brokers []string, topic string) *KafkaProducer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        true,
	}

	return &KafkaProducer{writer: writer}
}

func (kp *KafkaProducer) SendEvent(event GameEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error marshaling event: %v", err)
		return
	}

	msg := kafka.Message{
		Key:   []byte(event.GameID),
		Value: data,
		Time:  time.Now(),
	}

	err = kp.writer.WriteMessages(context.Background(), msg)
	if err != nil {
		log.Printf("Error sending event to Kafka: %v", err)
	}
}

func (kp *KafkaProducer) Close() error {
	return kp.writer.Close()
}

type GameEvent struct {
	EventType string    `json:"eventType"` // "game_start", "move", "game_end"
	GameID    string    `json:"gameId"`
	Timestamp time.Time `json:"timestamp"`
	Player1   string    `json:"player1"`
	Player2   string    `json:"player2"`
	Player2IsBot bool   `json:"player2IsBot"`
	Move      *MoveData `json:"move,omitempty"`
	Result    *GameResult `json:"result,omitempty"`
}

type MoveData struct {
	PlayerNum int `json:"playerNum"`
	Column    int `json:"column"`
	MoveNum   int `json:"moveNum"`
}

type GameResult struct {
	Winner        int `json:"winner"`
	TotalMoves    int `json:"totalMoves"`
	DurationSecs  int `json:"durationSecs"`
}
