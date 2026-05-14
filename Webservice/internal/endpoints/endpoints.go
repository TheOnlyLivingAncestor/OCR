package endpoints

import (
	"OCR/webservice/internal/storage"
	"encoding/json"
	"log/slog"
	"net/http"
	"ocr/packages/queue"
	"path/filepath"
	"sync"

	"github.com/gorilla/websocket"
	rmq "github.com/rabbitmq/rabbitmq-amqp-go-client/pkg/rabbitmqamqp"
)

var Clients = make(map[string]*websocket.Conn)
var Mu sync.RWMutex

func NewWebSocketHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("WS HIT", "url", r.URL.String())
		jobID := r.URL.Query().Get("jobID")

		//HTTP -> Websocket upgrade
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}
		con, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Error("Failed to upgrade to Websocket connection", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		logger.Info("Connected to websocket")

		Mu.Lock()
		Clients[jobID] = con
		Mu.Unlock()

		logger.Info("Websocket saved", "jobID", jobID)

		for {
			_, _, err := con.ReadMessage()
			if err != nil {
				logger.Error("Websocket closed", "error", err)
				break
			}
		}
	}
}

func NewOCRRequestHandler(logger *slog.Logger, minio_client storage.Storage, rabbitmq queue.Queue) http.HandlerFunc {
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

		id := r.FormValue("jobID")
		if id == "" {
			logger.Info("JobID is empty")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		//Upload image to object storage
		base_filename := id + filepath.Ext(handler.Filename)
		upload_filename := id + "_processed.json"
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
		data, err := json.Marshal(queue.RmqMessage{
			Download_link: download_link,
			Upload_link:   upload_link,
			JobID:         id,
		})
		if err != nil {
			logger.Error("Failed to Marshal RabbitMQ message", "error", err)
		}
		publish_result, err := rabbitmq.PublishMessage(data)
		if err != nil {
			logger.Error("Failed to publish message", "error", err)
		}
		//Bele kell tenni egy globális változóba az OCR requestet
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
