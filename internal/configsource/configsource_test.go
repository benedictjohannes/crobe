package configsource

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

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
		w.Header().Set("Content-Type", "application/x-yaml")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	data, contentType, err := fetchHttpsPlaybook(server.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("Expected %s, got %s", string(content), string(data))
	}
	if !strings.HasPrefix(contentType, "application/x-yaml") {
		t.Errorf("Expected application/x-yaml, got %s", contentType)
	}

	// Test 404
	errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer errServer.Close()

	_, _, err = fetchHttpsPlaybook(errServer.URL)
	if err == nil || !strings.Contains(err.Error(), "status 404") {
		t.Errorf("Expected 404 error, got %v", err)
	}

	// Test connection error
	_, _, err = fetchHttpsPlaybook("http://localhost:1")
	if err == nil {
		t.Errorf("Expected connection error")
	}
}

func TestFetchHttpsPlaybook_InvalidUrl(t *testing.T) {
	_, _, err := fetchHttpsPlaybook(":%: invalid-url")
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

func TestLoadConfig_Json(t *testing.T) {
	content := []byte(`{"title": "Json Playbook", "sections": []}`)
	tmpfile, err := os.CreateTemp("", "playbook*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// Local file uses JSON parser because of .json extension
	config, _, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if config.Title != "Json Playbook" {
		t.Errorf("Expected config title to be 'Json Playbook', got %s", config.Title)
	}
}

func TestLoadConfig_LocalJsonError(t *testing.T) {
	content := []byte(`invalid json`)
	tmpfile, err := os.CreateTemp("", "invalid*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	_, _, err = LoadConfig(tmpfile.Name())
	if err == nil || !strings.Contains(err.Error(), "failed to parse JSON") {
		t.Errorf("Expected JSON parse error, got %v", err)
	}
}

func TestLoadConfig_HttpsJson(t *testing.T) {
	content := []byte(`{"title": "Remote Json", "sections": []}`)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	// Since LoadConfig checks for https:// prefix, we use a hack:
	// Use fetchHttpsPlaybook directly to test the fetching logic
	// Or we can mock the fetch function if we want to test LoadConfig's logic.
	// But let's just test that fetchHttpsPlaybook returns the correct content-type.
	data, contentType, err := fetchHttpsPlaybook(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != string(content) {
		t.Errorf("Expected %s, got %s", string(content), string(data))
	}

	if !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("Expected application/json, got %s", contentType)
	}

	// Now we can manually call the unmarshal logic that LoadConfig uses
	// or we can test LoadConfig if we could pass a URL that it accepts.
	// Since we can't easily mock the prefix check without changing the code,
	// let's just assume LoadConfig works if fetchHttpsPlaybook returns the right thing
	// and we already tested LoadConfig's parsing logic separately (implicitly).

	// Actually, I can test LoadConfig with a local file and then manually triggering the JSON branch
	// but there is no way to set contentType for local files.
}

func TestLoadConfig_HttpsTls(t *testing.T) {
	content := []byte(`{"title": "TLS Playbook", "sections": []}`)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	// Configure default transport to trust the self-signed cert
	transport := http.DefaultTransport.(*http.Transport)
	oldTLSConfig := transport.TLSClientConfig
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	defer func() { transport.TLSClientConfig = oldTLSConfig }()

	config, data, err := LoadConfig(server.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if config.Title != "TLS Playbook" {
		t.Errorf("Expected TLS Playbook, got %s", config.Title)
	}

	if string(data) != string(content) {
		t.Error("Data mismatch")
	}

	// Test JSON error
	errorServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer errorServer.Close()

	_, _, err = LoadConfig(errorServer.URL)
	if err == nil || !strings.Contains(err.Error(), "failed to parse JSON") {
		t.Errorf("Expected JSON parse error, got %v", err)
	}
}

func TestFetchHttpsPlaybook_ReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set content length but don't write enough data
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)

		// Hijack the connection and close it to cause an unexpected EOF or similar read error
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("webserver doesn't support hijacking")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Fatal(err)
		}
		conn.Close()
	}))
	defer server.Close()

	_, _, err := fetchHttpsPlaybook(server.URL)
	if err == nil {
		t.Error("Expected read error")
	}
}
