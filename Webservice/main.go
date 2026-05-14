package main

import (
	"OCR/webservice/internal/endpoints"
	"OCR/webservice/internal/storage"
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"ocr/packages/queue"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/minio/minio-go/v7/pkg/credentials"
)

var MinioAddr = "minio.minio.svc.cluster.local"
var MinioPort = "9000"
var MinioBucket = "ocr-bucket"
var MinioUser = "minioadmin"
var MinioPassword = "minioadmin"

var RabbitAddr = "hello-world.rabbitmq-cluster.svc.cluster.local"
var RabbitPort = "5672"
var RabbitPublisherQueue = "recognition-request"
var RabbitConsumerQueue = "recognition-successful"
var RabbitUser = "guest"
var RabbitPassword = "guest"

func startServer(s *http.Server, logger *slog.Logger) {
	logger.Info("Server listening on http://:8080")
	err := s.ListenAndServe()
	// http.ErrServerClosed should not be logged to the user
	if err != nil && err != http.ErrServerClosed {
		log.Fatal("HTTP server error", "error", err)
	}
}

func read_minio_environment(logger *slog.Logger) {
	//Read minio variables from environment variables
	if os.Getenv("MINIO_BUCKET") != "" {
		logger.Info("MINIO_BUCKET environment variable is set, using that", "value", os.Getenv("MINIO_BUCKET"))
		MinioBucket = os.Getenv("MINIO_BUCKET")
	} else {
		logger.Info("MINIO_BUCKET environment variable is not set, using default", "value", MinioBucket)
	}

	if os.Getenv("MINIO_ADDR") != "" {
		logger.Info("MINIO_ADDR environment variable is set, using that", "value", os.Getenv("MINIO_ADDR"))
		MinioAddr = os.Getenv("MINIO_ADDR")
	} else {
		logger.Info("MINIO_ADDR environment variable is not set, using default", "value", MinioAddr)
	}

	if os.Getenv("MINIO_USER") != "" {
		logger.Info("MINIO_USER environment variable is set, using that", "value", os.Getenv("MINIO_USER"))
		MinioUser = os.Getenv("MINIO_USER")
	} else {
		logger.Info("MINIO_USER environment variable is not set, using default", "value", MinioUser)
	}

	if os.Getenv("MINIO_PASSWORD") != "" {
		logger.Info("MINIO_PASSWORD environment variable is set, using that", "value", os.Getenv("MINIO_PASSWORD"))
		MinioPassword = os.Getenv("MINIO_PASSWORD")
	} else {
		logger.Info("MINIO_PASSWORD environment variable is not set, using default", "value", MinioPassword)
	}
	if os.Getenv("MINIO_PORT") != "" {
		logger.Info("MINIO_PORT environment variable is set, using that", "value", os.Getenv("MINIO_PORT"))
		MinioPort = os.Getenv("MINIO_PORT")
	} else {
		logger.Info("MINIO_PORT environment variable is not set, using default", "value", MinioPort)
	}
}

func read_rabbit_environment(logger *slog.Logger) {
	//Read rabbitmq variables from environment variables
	if os.Getenv("RABBIT_PUBLISHER_QUEUE") != "" {
		logger.Info("RABBIT_PUBLISHER_QUEUE environment variable is set, using that", "value", os.Getenv("RABBIT_PUBLISHER_QUEUE"))
		RabbitPublisherQueue = os.Getenv("RABBIT_PUBLISHER_QUEUE")
	} else {
		logger.Info("RABBIT_QUEUE environment variable is not set, using default", "value", RabbitPublisherQueue)
	}

	if os.Getenv("RABBIT_CONSUMER_QUEUE") != "" {
		logger.Info("RABBIT_CONSUMER_QUEUE environment variable is set, using that", "value", os.Getenv("RABBIT_CONSUMER_QUEUE"))
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

func rmq_consumer_loop(ctx context.Context, logger *slog.Logger, rmq queue.Queue, minio *storage.MinioStorage) {
consumer_loop:
	for {
		select {
		case <-ctx.Done():
			logger.Info("Shutting down consumer loop")
			break consumer_loop
		default:
			deliveryctx, err := rmq.GetMessage()
			if err != nil {
				logger.Error("Failed to receive message from RabbitMQ", "error", err)
				continue
			}
			msg := deliveryctx.Message().GetData()
			logger.Info("Received message", "data", msg)
			var msg_json queue.RmqSuccess
			if err = json.Unmarshal(msg, &msg_json); err != nil {
				logger.Error("Failed to unmarshal message", "error", err)
				//Nem tudjuk processzálni az üzenetet -> nagy valséggel hibás, discard?
				err = deliveryctx.Discard(context.Background(), nil)
				if err != nil {
					logger.Error("Failed to discard message", "error", err)
				}
			}
			logger.Info("Unmarshaled message from RabbitMQ", "msg", msg_json)
			//Megkaptuk az üzenetet, RMQ részéről kész vagyunk -> Accept
			err = deliveryctx.Accept(context.Background())
			if err != nil {
				logger.Error("Error during accepting the message", "error", err)
			}
			//Le kell hívni az adott JobID-vel az eredményeket, és a websocketnek továbbítani
			ocr_result, err := minio.Download(msg_json.JobID + "_processed.json")
			if err != nil {
				logger.Error("Failed to download OCR result", "error", err)
				//Vissza kell küldeni vmi error code-t a websocketen?
			}
			logger.Info("Successfully downloaded OCR result", "image", ocr_result.Image, "jobID", ocr_result.JobID, "results", ocr_result.Results)
			//Websocketre továbbítunk
		}
	}
}

func main() {
	// Set the default logger to a fancier log format.
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	read_minio_environment(logger)
	read_rabbit_environment(logger)

	//init MinIO storage
	cred := credentials.NewStaticV4(MinioUser, MinioPassword, "")
	minio_storage := storage.NewMinioStorage(MinioAddr+":"+MinioPort, cred, false, MinioBucket, logger)
	ctx := context.Background()
	err := minio_storage.EnsureBucket(ctx)
	if err != nil {
		logger.Error("Failed to ensure storage bucket, exiting", "error", err)
		os.Exit(1)
	}

	rmq, err := queue.NewRabbitMQ("amqp://" + RabbitUser + ":" + RabbitPassword + "@" + RabbitAddr + ":" + RabbitPort + "/")
	if err != nil {
		logger.Error("Could not create RabbitMQ environment, exiting", "error", err)
		os.Exit(1)
	}

	err = rmq.CreateAll(RabbitConsumerQueue, RabbitPublisherQueue)
	if err != nil {
		logger.Error("Creating RabbitMQ componenst failed, exiting", "error", err)
		os.Exit(1)
	}
	logger.Info("Creating RabbitMQ components succeeded")

	defer func() {
		if err := rmq.CloseAll(); err != nil {
			logger.Error("Failed to close RabbitMQ components", "error", err)
		}
	}()

	// Static HTTP handler to serve files from the static folder.
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", endpoints.NewUIHandler(logger))
	http.HandleFunc("/process", endpoints.NewOCRRequestHandler(logger, minio_storage, rmq))
	http.HandleFunc("/healthz", endpoints.NewHealthzHandler(logger))

	//Start the rabbitMQ consumer in a goroutine
	rmq_ctx, rmq_cancel := context.WithCancel(context.Background())
	defer rmq_cancel()

	go rmq_consumer_loop(rmq_ctx, logger, rmq, minio_storage)

	//HTTP server starts in a goroutine to handle graceful shutdown
	s := &http.Server{Addr: ":8080"}
	// Start the server in the goroutine
	go startServer(s, logger)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutdown signal received")
	rmq_cancel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	err = s.Shutdown(ctx)
	if err != nil {
		logger.Info("Graceful server shutdown failed with", "error", err)
	} else {
		logger.Info("Graceful server sutdown succeeded")
	}

}
