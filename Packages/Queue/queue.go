package queue

import (
	"context"
	"log/slog"
	"os"

	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
)

var logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

type Queue interface {
	CreateConnection() error
	CloseConnection() error
	CreatePublisher(string) error
	ClosePublisher() error
	CreateConsumer(string) error
	CloseConsumer() error
	CreateAll(string, string) error
	CloseAll() error
	GetMessage() ([]byte, error)
}

type RmqMessage struct {
	Download_link string `json:"download_link"`
	Upload_link   string `json:"upload_link"`
	JobID         string `json:"jobID"`
}

type RabbitMQ struct {
	environment *rmq.Environment
	connection  *rmq.AmqpConnection
	publisher   *rmq.Publisher
	consumer    *rmq.Consumer
}

func NewRabbitMQ(address string) (Queue, error) {
	rmq_env := rmq.NewEnvironment(address, nil)
	logger.Info("RabbitMQ environment created")
	return &RabbitMQ{environment: rmq_env, connection: nil, publisher: nil, consumer: nil}, nil

}

func (q *RabbitMQ) CreateConnection() error {
	connection, err := q.environment.NewConnection(context.Background())
	if err != nil {
		logger.Error("Failed to create RabbitMQ connection", "error", err)
		return err
	}
	logger.Info("RabbitMQ connection successfully created")
	q.connection = connection
	return nil
}

func (q *RabbitMQ) CloseConnection() error {
	if err := q.connection.Close(context.Background()); err != nil {
		logger.Error("Failed to close connection", "error", err)
		return err
	}
	return nil
}

func (q *RabbitMQ) CreatePublisher(queue string) error {
	publisher, err := q.connection.NewPublisher(context.Background(),
		&rmq.QueueAddress{
			Queue: queue,
		},
		nil,
	)
	if err != nil {
		logger.Error("RabbitMQ publisher creation failed", "error", err)
		return err
	}
	logger.Info("RabbitMQ publisher creation succeeded")
	q.publisher = publisher
	return nil
}

func (q *RabbitMQ) ClosePublisher() error {
	if q.publisher == nil {
		logger.Info("No publisher to close.")
		return nil
	}
	if err := q.publisher.Close(context.Background()); err != nil {
		logger.Error("Failed to close RabbitMQ publisher", "error", err)
		return err
	}
	return nil
}

func (q *RabbitMQ) CreateConsumer(queue string) error {
	consumer, err := q.connection.NewConsumer(context.Background(), queue, nil)
	if err != nil {
		logger.Error("Failed to create RabbitMQ consumer", "error", err)
	}
	q.consumer = consumer
	return nil
}

func (q *RabbitMQ) CloseConsumer() error {
	if q.consumer == nil {
		logger.Info("No consumer to close.")
		return nil
	}
	if err := q.consumer.Close(context.Background()); err != nil {
		logger.Error("Failed to close connection", "error", err)
		return err
	}
	return nil
}

// Creates Connection, Consumer and Publisher.
func (q *RabbitMQ) CreateAll(c_queue string, p_queue string) error {
	if err := q.CreateConnection(); err != nil {
		return err
	}
	if err := q.CreateConsumer(c_queue); err != nil {
		return err
	}
	if err := q.CreatePublisher(p_queue); err != nil {
		return err
	}
	return nil
}

// Closes Publisher, Consumer and Connection.
func (q *RabbitMQ) CloseAll() error {
	if err := q.ClosePublisher(); err != nil {
		return err
	}
	if err := q.CloseConsumer(); err != nil {
		return err
	}
	if err := q.CloseConnection(); err != nil {
		return err
	}
	return nil
}

func (q *RabbitMQ) GetMessage() ([]byte, error) {
	deliveryctx, err := q.consumer.Receive(context.Background())
	if err != nil {
		logger.Error("Getting message from RabbitMQ failed", "error", err)
		return nil, err
	}
	return deliveryctx.Message().GetData(), err
}
