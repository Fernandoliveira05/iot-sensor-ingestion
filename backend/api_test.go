package main

import (
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

func TestPost(t *testing.T){
	gin.SetMode(gin.TestMode)
	router := setupRouter()

	req, err := http.NewRequest(http.MethodPost, "/telemetria", nil)
	if err != nil {
		t.Fatalf("erro ao criar request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	
}