package endpoints

import (
	"OCR/webservice/internal/queue"
	"OCR/webservice/internal/storage"
	"encoding/json"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/google/uuid"
	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
)

func NewOCRRequestHandler(logger *slog.Logger, minio_client *storage.MinioStorage, rabbitmq *queue.RabbitQueue) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		image, handler, err := r.FormFile("image")
		if err != nil {
			logger.Info("Error during image parsing", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer func() {
			if err := image.Close(); err != nil {
				logger.Info("Error occurred while closing image", "error", err)
			}
		}()
		logger.Info("Image from request read successfully", "name", handler.Filename, "size in bytes", handler.Size)

		description := r.FormValue("description")
		if description == "" {
			logger.Info("Description of image is empty")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		logger.Info("Description of image read successfully", "text", description)
		//Upload image to object storage
		id := uuid.New().String()
		base_filename := id + filepath.Ext(handler.Filename)
		upload_filename := id + "_processed" + filepath.Ext(handler.Filename)
		metadata := make(map[string]string)
		metadata["description"] = description
		metadata["jobID"] = id
		err = minio_client.Upload(r.Context(),
			storage.UploadRequest{
				File:        image,
				Size:        handler.Size,
				FileName:    base_filename,
				ContentType: handler.Header.Get("Content-Type"),
				Metadata:    metadata,
			},
		)
		if err != nil {
			logger.Error("Error occurred during image upload", "error", err)
			w.WriteHeader(http.StatusFailedDependency)
		}

		//Get the required links from MinIO
		download_link, err := minio_client.Get_Download_URL(r.Context(), base_filename)
		if err != nil {
			logger.Error("Failed to get MinIO download link", "error", err)
			w.WriteHeader(http.StatusFailedDependency)
		}

		upload_link, err := minio_client.Get_Upload_URL(r.Context(), upload_filename)
		if err != nil {
			logger.Error("Failed to get MinIO upload link", "error", err)
			w.WriteHeader(http.StatusFailedDependency)
		}

		//Send message to RabbitMQ queue
		data, err := json.Marshal(queue.Message{
			Download_link: download_link,
			Upload_link:   upload_link,
			JobID:         id,
		})
		if err != nil {
			logger.Error("Failed to Marshal RabbitMQ message", "error", err)
		}
		publish_result, err := rabbitmq.PublishImageReady(data)
		if err != nil {
			logger.Error("Failed to publish message", "error", err)
		}
		switch publish_result.Outcome.(type) {
		case *rmq.StateAccepted:
			logger.Info("The ocr-request message was accepted by RabbitMQ.")
		default:
			logger.Info("Something happened during sending the ocr-request.")
		}
	}
}

func NewHealthzHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//This handler should only accept GET requests
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		json, err := json.Marshal("OK")
		if err != nil {
			logger.Info("Failed to marshal response", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			logger.Info("API request failed", "error", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(json)
		if err != nil {
			logger.Info("Failed to write response", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func NewUIHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "static/index.html")
	}
}
