package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"ocr/ocr_service/watcher/internal/heartbeat"
	"ocr/packages/queue"
	"os"
	"os/signal"
	"syscall"
)

var RabbitAddr = "hello-world.rabbitmq-cluster.svc.cluster.local"
var RabbitPort = "5672"
var RabbitConsumerQueue = "recognition-request"
var RabbitUser = "guest"
var RabbitPassword = "guest"

var logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

func read_rabbit_environment() {
	//Read minio variables from environment variables
	if os.Getenv("RABBIT_CONSUMER_QUEUE") != "" {
		logger.Info("RABBIT_QUEUE environment variable is set, using that", "value", os.Getenv("RABBIT_CONSUMER_QUEUE"))
		RabbitConsumerQueue = os.Getenv("RABBIT_CONSUMER_QUEUE")
	} else {
		logger.Info("RABBIT_CONSUMER_QUEUE environment variable is not set, using default", "value", RabbitConsumerQueue)
	}

	if os.Getenv("RABBIT_ADDR") != "" {
		logger.Info("RABBIT_ADDR environment variable is set, using that", "value", os.Getenv("RABBIT_ADDR"))
		RabbitAddr = os.Getenv("RABBIT_ADDR")
	} else {
		logger.Info("RABBIT_ADDR environment variable is not set, using default", "value", RabbitAddr)
	}

	if os.Getenv("RABBIT_USER") != "" {
		logger.Info("RABBIT_USER environment variable is set, using that", "value", os.Getenv("RABBIT_USER"))
		RabbitUser = os.Getenv("RABBIT_USER")
	} else {
		logger.Info("RABBIT_USER environment variable is not set, using default", "value", RabbitUser)
	}

	if os.Getenv("RABBIT_PASSWORD") != "" {
		logger.Info("RABBIT_PASSWORD environment variable is set, using that", "value", os.Getenv("RABBIT_PASSWORD"))
		RabbitPassword = os.Getenv("RABBIT_PASSWORD")
	} else {
		logger.Info("RABBIT_PASSWORD environment variable is not set, using default", "value", RabbitPassword)
	}
	if os.Getenv("RABBIT_PORT") != "" {
		logger.Info("RABBIT_PORT environment variable is set, using that", "value", os.Getenv("RABBIT_PORT"))
		RabbitPort = os.Getenv("RABBIT_PORT")
	} else {
		logger.Info("RABBIT_PORT environment variable is not set, using default", "value", RabbitPort)
	}
}

func main() {
	logger.Info("Setting up RabbitMQ")
	read_rabbit_environment()
	rmq, err := queue.NewRabbitMQ("amqp://" + RabbitUser + ":" + RabbitPassword + "@" + RabbitAddr + ":" + RabbitPort + "/")
	if err != nil {
		logger.Error("Failed to create RabbitMQ object, exiting", "error", err)
		os.Exit(1)
	}
	err = rmq.CreateConnection()
	if err != nil {
		logger.Error("Failed to create RabbitMQ connection, exiting", "error", err)
		os.Exit(1)
	}
	err = rmq.CreateConsumer(RabbitConsumerQueue)
	if err != nil {
		logger.Error("Failed to create RabbitMQ Consumer, exiting", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer func() {
		if err := rmq.CloseAll(); err != nil {
			logger.Error("Failed to close RabbitMQ components", "error", err)
			cancel()
			os.Exit(1)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("Starting graceful shutdown goroutine")
	go func() {
		<-sigChan
		logger.Info("Shutdown signal received, graceful shutdown starts")
		cancel()
	}()

	logger.Info("Starting heartbeat goroutine")
	go heartbeat.StartHeartbeat(ctx, cancel)

worker_loop:
	for {
		select {
		case <-ctx.Done():
			logger.Info("Shutting down worker loop")
			break worker_loop
		default:
			msg, err := rmq.GetMessage()
			if err != nil {
				logger.Error("Failed to receive message from RabbitMQ")
				continue
			}
			logger.Info("Received message", "data", msg)
			var msg_json queue.RmqMessage
			if err = json.Unmarshal(msg, &msg_json); err != nil {
				logger.Error("Failed to unmarshal message", "error", err)
				//Itt vissza kéne am utasítani az üzenetet?
			}
			logger.Info("Unmarshaled message from RabbitMQ is the following beautiful message :)", "msg", msg_json)
		}
	}

}
