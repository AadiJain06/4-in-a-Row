package analytics

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	if len(brokers) == 0 || topic == "" {
		return nil
	}
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		AllowAutoTopicCreation: true,
	}
	return &Producer{writer: writer}
}

func (p *Producer) Publish(ctx context.Context, event string, payload map[string]any) {
	if p == nil || p.writer == nil {
		return
	}
	body := map[string]any{
		"event":     event,
		"payload":   payload,
		"timestamp": time.Now().UTC(),
	}
	data, _ := json.Marshal(body)
	err := p.writer.WriteMessages(ctx, kafka.Message{Value: data})
	if err != nil {
		log.Printf("kafka publish failed: %v", err)
	}
}

func (p *Producer) Close() {
	if p == nil || p.writer == nil {
		return
	}
	_ = p.writer.Close()
}

