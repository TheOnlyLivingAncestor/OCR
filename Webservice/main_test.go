package main

import (
	"OCR/webservice/packages/endpoints"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestUI(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	//Simulate an incoming server request for "/"
	request, _ := http.NewRequest("GET", "/", nil)
	//Record the simulation of HTTP response
	response := httptest.NewRecorder()
	//Run the function I want to test
	UIHandlerfunc := endpoints.NewUIHandler(logger)
	UIHandlerfunc(response, request)
	//Check if the response is what we expect -> return code 200
	response_code := response.Code
	if response_code != 200 {
		t.Errorf("Expected to successfully reach UI, got %v ", response_code)
	}
}

func TestHealthz(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	//Simulate an incoming server request for "/healthz"
	request, _ := http.NewRequest("GET", "/healthz", nil)
	//Record the simulation of HTTP response
	response := httptest.NewRecorder()
	//Run the function I want to test
	HealthzHandler := endpoints.NewHealthzHandler(logger)
	HealthzHandler(response, request)
	//Check if the response is what we expect -> return code 200
	response_body := response.Body
	var body string
	json.Unmarshal(response_body.Bytes(), &body)
	want := "OK"
	if body != want {
		t.Errorf("Healthz endpoint was expected to return %v, but got %v", want, body)
	}
}
