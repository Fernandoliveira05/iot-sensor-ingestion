package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := setupRouter()

	req, err := http.NewRequest(http.MethodGet, "/ping", nil)
	if err != nil {
		t.Fatalf("erro ao criar request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperado status %d, mas recebeu %d", http.StatusOK, w.Code)
	}

	bodyEsperado := `{"message":"pong"}`
	if w.Body.String() != bodyEsperado {
		t.Errorf("esperado body %s, mas recebeu %s", bodyEsperado, w.Body.String())
	}
}

func TestPost(t *testing.T) {
	gin.SetMode(gin.TestMode)

	publishMessage = func(message Telemetry) error {
		return nil
	}
	defer func() {
		publishMessage = sendToRabbitMQ
	}()

	router := setupRouter()

	jsonBody := `{
		"device_id": "1",
		"timestamp": "2026-03-17T11:36:00Z",
		"sensor_type": "temperature",
		"reading_type": "analog",
		"value": 23.5
	}`

	req, err := http.NewRequest(
		http.MethodPost,
		"/telemetria",
		bytes.NewBufferString(jsonBody),
	)
	if err != nil {
		t.Fatalf("erro ao criar request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("esperado status %d, mas recebeu %d", http.StatusAccepted, w.Code)
	}

	bodyEsperado := `{"data":{"device_id":"1","sensor_type":"temperature","reading_type":"analog","value":23.5,"timestamp":"2026-03-17T11:36:00Z"},"message":"Enviado para processamento assíncrono"}`
	if w.Body.String() != bodyEsperado {
		t.Errorf("esperado body %s, mas recebeu %s", bodyEsperado, w.Body.String())
	}
}