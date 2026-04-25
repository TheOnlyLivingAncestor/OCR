package endpoints

import (
	"encoding/json"
	"log"
	"net/http"
)

func OcrRequestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	image, handler, err := r.FormFile("image")
	if err != nil {
		log.Printf("Error during image parsing: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer image.Close()
	log.Printf("Image from request read successfully with name %s and size %v bytes", handler.Filename, handler.Size)

	description := r.FormValue("description")
	if description == "" {
		log.Println("Description of image is empty")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Printf("Description of image read successfully: %v", description)
}

func HealthzHandler(w http.ResponseWriter, r *http.Request) {
	//This handler should only accept GET requests
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	json, err := json.Marshal("OK")
	if err != nil {
		log.Printf("Failed to marshal response with error %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("API request failed: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(json)
	if err != nil {
		log.Printf("Failed to write response with error %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func UIHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "static/index.html")
}
