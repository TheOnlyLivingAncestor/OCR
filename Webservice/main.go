package main

import (
	"OCR/webservice/packages/endpoints"
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	//"github.com/minio/minio-go/v7"
)

var MinioAddr = "minio.minio.svc.cluster.local"
var MinioPort = "9000"
var MinioBucket = "ocr_bucket"
var MinioUser = "minioadmin"
var MinioPassword = "minioadmin"

func startServer(s *http.Server, logger *slog.Logger) {
	logger.Info("Server listening on http://:8080")
	err := s.ListenAndServe()
	// http.ErrServerClosed should not be logged to the user
	if err != nil && err != http.ErrServerClosed {
		log.Fatal("HTTP server error", "error", err)
	}
}

func read_environment(logger *slog.Logger) {
	//Read variables from environment variables
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

func main() {
	// Set the default logger to a fancier log format.
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	read_environment(logger)

	// Static HTTP handler to serve files from the static folder.
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", endpoints.UIHandler)
	http.HandleFunc("/process", endpoints.OcrRequestHandler)
	http.HandleFunc("/healthz", endpoints.HealthzHandler)

	//HTTP server starts in a goroutine to handle graceful shutdown
	s := &http.Server{Addr: ":8080"}
	// Start the server in the goroutine
	go startServer(s, logger)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutdown signal received")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	err := s.Shutdown(ctx)
	if err != nil {
		logger.Info("Graceful server shutdown failed with", "error", err)
	} else {
		logger.Info("Graceful server sutdown succeeded")
	}

}
