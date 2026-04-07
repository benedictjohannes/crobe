package configsource

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestGetConfigSource(t *testing.T) {
	// 1. Explicit path
	if GetConfigSource("test.yaml") != "test.yaml" {
		t.Errorf("Expected test.yaml")
	}

	// 2. Default playbook.yaml
	os.WriteFile("playbook.yaml", []byte("test"), 0644)
	defer os.Remove("playbook.yaml")

	if GetConfigSource("") != "playbook.yaml" {
		t.Errorf("Expected playbook.yaml as default")
	}

	// 3. None
	os.Remove("playbook.yaml")
	if GetConfigSource("") != "" {
		t.Errorf("Expected empty string when no playbook.yaml exists")
	}
}

func TestLoadConfig_Insecure(t *testing.T) {
	_, _, err := LoadConfig("http://example.com/playbook.yaml")
	if err == nil || !strings.Contains(err.Error(), "insecure HTTP") {
		t.Errorf("Expected error for insecure HTTP")
	}
}

func TestLoadConfig_LocalFile(t *testing.T) {
	// 1. Success
	content := []byte("title: Test Playbook\nsections: []")
	tmpfile, err := os.CreateTemp("", "playbook*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	config, data, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("Expected data to match content")
	}
	if config.Title != "Test Playbook" {
		t.Errorf("Expected config title to be 'Test Playbook', got %s", config.Title)
	}

	// 2. Not found
	_, _, err = LoadConfig("non-existent-file.yaml")
	if err == nil {
		t.Errorf("Expected error for non-existent file")
	}

	// 3. Invalid YAML
	tmpfileInvalid, err := os.CreateTemp("", "invalid*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfileInvalid.Name())
	if _, err := tmpfileInvalid.Write([]byte("invalid: yaml: : content")); err != nil {
		t.Fatal(err)
	}
	tmpfileInvalid.Close()

	_, _, err = LoadConfig(tmpfileInvalid.Name())
	if err == nil || !strings.Contains(err.Error(), "failed to parse YAML") {
		t.Errorf("Expected parsing error, got %v", err)
	}

	// 4. Directory
	tmpDir, _ := os.MkdirTemp("", "testdir")
	defer os.RemoveAll(tmpDir)
	_, _, err = LoadConfig(tmpDir)
	if err == nil {
		t.Errorf("Expected error for directory path")
	}
}

func TestFetchHttpsPlaybook(t *testing.T) {
	content := []byte("test content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	data, err := fetchHttpsPlaybook(server.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("Expected %s, got %s", string(content), string(data))
	}

	// Test 404
	errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer errServer.Close()

	_, err = fetchHttpsPlaybook(errServer.URL)
	if err == nil || !strings.Contains(err.Error(), "status 404") {
		t.Errorf("Expected 404 error, got %v", err)
	}

	// Test connection error
	_, err = fetchHttpsPlaybook("http://localhost:1")
	if err == nil {
		t.Errorf("Expected connection error")
	}
}

func TestFetchHttpsPlaybook_InvalidUrl(t *testing.T) {
	_, err := fetchHttpsPlaybook(":%: invalid-url")
	if err == nil {
		t.Errorf("Expected error for invalid URL")
	}
}

func TestLoadConfig_HttpsMock(t *testing.T) {
	// Since LoadConfig checks for https:// prefix, we can't easily use httptest.NewServer
	// which returns http://. We could use httptest.NewTLSServer but then we'd need to
	// configure the client to trust the self-signed cert.
	// However, we already tested fetchHttpsPlaybook separately.
	// To test the branch in LoadConfig, we can temporarily mock or just trust our
	// fetchHttpsPlaybook tests.
	// Let's try to use a real HTTPS server if possible, or just accept that we covered
	// that branch via fetchHttpsPlaybook tests and the prefix check test.

	// Actually, we can use a small trick: test with a URL that starts with https://
	// but is invalid, just to see if it reaches fetchHttpsPlaybook.
	_, _, err := LoadConfig("https://invalid.example.com/playbook.yaml")
	if err == nil {
		t.Errorf("Expected error for invalid HTTPS URL")
	}
	// The error should come from fetchHttpsPlaybook
	if !strings.Contains(err.Error(), "failed to fetch remote playbook") {
		t.Errorf("Expected 'failed to fetch remote playbook' error, got %v", err)
	}
}
