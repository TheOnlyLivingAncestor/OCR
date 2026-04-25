package handler

import (
	"OCR/webservice/internal/queue"
	"OCR/webservice/internal/storage"
)

type Handler struct {
	storage storage.Storage
	queue   queue.Queue
}

func NewHandler(storage storage.Storage, queue queue.Queue) *Handler {
	return &Handler{
		storage: storage,
		queue:   queue,
	}
}
