package main

import (
	"OCR/webservice/endpoints"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	//"github.com/minio/minio-go/v7"
)

var MinioAddr = "minio.minio.svc.cluster.local"
var MinioPort = 9000
var MinioBucket = "ocrbucket"
var MinioUser = "admin"
var MinioPassword = "admin"

func read_environment() {
	//Read variables from environment variables
	if os.Getenv("MINIO_BUCKET") != "" {
		log.Printf("MINIO_BUCKET environment variable is set, using its value %v", os.Getenv("MINIO_BUCKET"))
		MinioBucket = os.Getenv("MINIO_BUCKET")
	} else {
		log.Printf("MINIO_BUCKET environment variable is not set, using default value %v", MinioBucket)
	}

	if os.Getenv("MINIO_ADDR") != "" {
		log.Printf("MINIO_ADDR environment variable is set, using its value %v", os.Getenv("MINIO_ADDR"))
		MinioAddr = os.Getenv("MINIO_ADDR")
	} else {
		log.Printf("MINIO_ADDR environment variable is not set, using default value %v", MinioAddr)
	}

	if os.Getenv("MINIO_USER") != "" {
		log.Printf("MINIO_USER environment variable is set, using its value %v", os.Getenv("MINIO_USER"))
		MinioUser = os.Getenv("MINIO_USER")
	} else {
		log.Printf("MINIO_USER environment variable is not set, using default value %v", MinioUser)
	}

	if os.Getenv("MINIO_PASSWORD") != "" {
		log.Printf("MINIO_PASSWORD environment variable is set, using its value %v", os.Getenv("MINIO_PASSWORD"))
		MinioPassword = os.Getenv("MINIO_PASSWORD")
	} else {
		log.Printf("MINIO_PASSWORD environment variable is not set, using default value %v", MinioPassword)
	}
}

func startServer(s *http.Server) {
	log.Println("Server listening on http://:8080")
	err := s.ListenAndServe()
	// http.ErrServerClosed should not be logged to the user
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server error %v", err)
	}
}

func main() {
	// Set the default logger to a fancier log format.
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	read_environment()

	// Static HTTP handler to serve files from the static folder.
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/", endpoints.UIHandler)
	http.HandleFunc("/process", endpoints.OcrRequestHandler)
	http.HandleFunc("/healthz", endpoints.HealthzHandler)

	//HTTP server starts in a goroutine to handle graceful shutdown
	s := &http.Server{Addr: ":8080"}
	// Start the server in the goroutine
	go startServer(s)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutdown signal received")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	err := s.Shutdown(ctx)
	if err != nil {
		log.Printf("Graceful server shutdown failed with error %v", err)
	} else {
		log.Println("Graceful server sutdown succeeded")
	}

}
