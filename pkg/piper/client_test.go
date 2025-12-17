package piper

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSynthesize(t *testing.T) {
	// Mock Piper Server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify Request
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)
		var req SynthesizeRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("Failed to unmarshal request: %v", err)
		}

		if req.Text != "Hello Fish" {
			t.Errorf("Expected 'Hello Fish', got '%s'", req.Text)
		}

		// Return Audio Data
		w.Write([]byte("mock-audio-data"))
	}))
	defer server.Close()

	// Test Client
	client := NewPiperClient(server.URL)
	audio, err := client.Synthesize("Hello Fish")

	if err != nil {
		t.Fatalf("Synthesize failed: %v", err)
	}
	if string(audio) != "mock-audio-data" {
		t.Errorf("Expected 'mock-audio-data', got '%s'", string(audio))
	}
}
