package endpoints

import (
	"OCR/webservice/internal/storage"
	"encoding/json"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/google/uuid"
)

func NewOCRRequestHandler(logger *slog.Logger, minio_client *storage.MinioStorage) http.HandlerFunc {
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
		filename := id + filepath.Ext(handler.Filename)
		metadata := make(map[string]string)
		metadata["description"] = description
		metadata["jobID"] = id
		minio_client.Upload(r.Context(),
			storage.UploadRequest{
				File:        image,
				Size:        handler.Size,
				FileName:    filename,
				ContentType: handler.Header.Get("Content-Type"),
				Metadata:    metadata,
			},
		)
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
