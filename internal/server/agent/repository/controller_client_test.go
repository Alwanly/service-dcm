package repository

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Alwanly/service-distribute-management/internal/config"
	"github.com/Alwanly/service-distribute-management/internal/models"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
)

func TestGetConfiguration_OK(t *testing.T) {
	// prepare a test server that returns a configuration and ETag
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", "v1")
		cfg := models.Configuration{ID: 1, ETag: "v1", ConfigData: "{\"url\":\"http://example\"}"}
		_ = json.NewEncoder(w).Encode(cfg)
	}))
	defer ts.Close()

	// construct controller client with test server URL
	cfg := &config.AgentConfig{ControllerURL: ts.URL, RequestTimeout: 2 * time.Second}
	log, _ := logger.NewLoggerFromEnv("test")
	client := NewControllerClient(cfg, log)

	c, etag, notModified, err := client.GetConfiguration(context.Background(), "agent-1", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notModified {
		t.Fatalf("expected notModified=false, got true")
	}
	if etag != "v1" {
		t.Fatalf("expected etag v1, got %s", etag)
	}
	if c == nil || c.ConfigData == "" {
		t.Fatalf("expected configuration body, got %+v", c)
	}
}

func TestGetConfiguration_NotModified(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// if client sends If-None-Match, return 304
		if inm := r.Header.Get("If-None-Match"); inm != "v1" {
			w.WriteHeader(http.StatusOK)
			cfg := models.Configuration{ID: 1, ETag: "v2", ConfigData: "{\"url\":\"http://example\"}"}
			_ = json.NewEncoder(w).Encode(cfg)
			return
		}
		w.WriteHeader(http.StatusNotModified)
	}))
	defer ts.Close()

	cfg := &config.AgentConfig{ControllerURL: ts.URL, RequestTimeout: 2 * time.Second}
	log, _ := logger.NewLoggerFromEnv("test")
	client := NewControllerClient(cfg, log)

	// First call without ETag -> should return body
	c, _, notModified, err := client.GetConfiguration(context.Background(), "agent-1", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notModified {
		t.Fatalf("expected notModified=false on first call")
	}
	if c == nil {
		t.Fatalf("expected configuration on first call")
	}
	// Now call with ETag v1 to trigger 304
	_, _, notModified, err = client.GetConfiguration(context.Background(), "agent-1", "", "v1")
	if err != nil {
		t.Fatalf("unexpected error on conditional request: %v", err)
	}
	if !notModified {
		t.Fatalf("expected notModified=true when server returns 304")
	}
}
