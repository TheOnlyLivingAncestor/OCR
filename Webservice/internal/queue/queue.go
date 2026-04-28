package queue

import (
	"context"
	"log/slog"

	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
)

type Queue interface {
	PublishImageReady([]byte) (*rmq.PublishResult, error)
}

type Message struct {
	Download_link string `json:"download_link"`
	Upload_link   string `json:"upload_link"`
	JobID         string `json:"jobID"`
}

type RabbitQueue struct {
	publisher *rmq.Publisher
	logger    *slog.Logger
}

func NewRabbitQueue(pb *rmq.Publisher, logger *slog.Logger) *RabbitQueue {
	return &RabbitQueue{
		publisher: pb,
		logger:    logger,
	}
}

func (rabbitmq *RabbitQueue) PublishImageReady(data []byte) (*rmq.PublishResult, error) {
	message := rmq.NewMessage(data)
	publish_result, err := rabbitmq.publisher.Publish(context.Background(), message)
	if err != nil {
		rabbitmq.logger.Error("There was an error during message publishing", "error", err)
	}
	rabbitmq.logger.Info("Message publishing succeeded", "data", data)
	return publish_result, nil
}
