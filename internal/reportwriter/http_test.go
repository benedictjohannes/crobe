package reportwriter

import (
	"github.com/benedictjohannes/crobe/playbook"
	"github.com/benedictjohannes/crobe/report"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWriteToHTTP(t *testing.T) {
	// Disable TLS verification for self-signed cert from httptest
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// 1. Setup a mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		err := r.ParseMultipartForm(10 << 20) // 10MB
		if err != nil {
			t.Fatalf("Failed to parse multipart form: %v", err)
		}

		// Check files
		checkFile := func(name string, contentType string, expectedContent string, expectSignature bool) {
			f, h, err := r.FormFile(name)
			if err != nil {
				t.Errorf("Missing file part: %s", name)
				return
			}
			defer f.Close()

			content, _ := io.ReadAll(f)
			if h.Header.Get("Content-Type") != contentType {
				t.Errorf("Expected content type %s for %s, got %s", contentType, name, h.Header.Get("Content-Type"))
			}
			if h.Header.Get("Content-Transfer-Encoding") != "base64" {
				t.Errorf("Expected base64 encoding header for %s", name)
			}

			decoded, err := base64.StdEncoding.DecodeString(string(content))
			if err != nil {
				t.Errorf("Failed to decode base64 for %s: %v", name, err)
			}

			if strings.TrimSpace(string(decoded)) != strings.TrimSpace(expectedContent) {
				t.Errorf("Content mismatch for %s. Got %s, want %s", name, string(decoded), expectedContent)
			}

			if expectSignature {
				sigPart, _, err := r.FormFile(name + ".signature.txt")
				if err != nil {
					t.Errorf("Missing signature part for %s", name)
					return
				}
				defer sigPart.Close()

				sigBytes, _ := io.ReadAll(sigPart)
				expectedSig := calculateHMAC(string(content), "this-is-a-secret")
				if strings.TrimSpace(string(sigBytes)) != strings.TrimSpace(expectedSig) {
					t.Errorf("Signature mismatch for %s. Got %s, want %s", name, string(sigBytes), expectedSig)
				}
			}
		}

		checkFile("report.json", "application/json", "{\n  \"timestamps\": {\n    \"start\": \"2026-04-10T12:00:00Z\",\n    \"end\": \"2026-04-10T12:00:10Z\"\n  },\n  \"username\": \"testuser\",\n  \"os\": \"\",\n  \"arch\": \"\",\n  \"assertions\": null,\n  \"stats\": {\n    \"passed\": 0,\n    \"failed\": 0\n  }\n}", true)
		checkFile("report.md", "text/markdown", "# Test Markdown", true)
		checkFile("report.log", "text/plain", "test log", true)

		if r.Header.Get("X-Custom-Header") != "custom-value" {
			t.Errorf("Missing or incorrect custom header: %s", r.Header.Get("X-Custom-Header"))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// 2. Prepare test data
	res := report.FinalResult{
		Structured: report.FinalReport{
			Timestamps: struct {
				Start time.Time `json:"start"`
				End   time.Time `json:"end"`
			}{
				Start: time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC),
				End:   time.Date(2026, 4, 10, 12, 0, 10, 0, time.UTC),
			},
			Username: "testuser",
		},
		Markdown: "# Test Markdown",
		Log:      "test log",
	}

	config := &playbook.ReportDestinationConfig{
		URL:             server.URL,
		SignatureSecret: "this-is-a-secret",
		AdditionalHeaders: map[string]string{
			"X-Custom-Header": "custom-value",
		},
	}

	// 3. Execute
	err := WriteToHTTP(config, res)
	if err != nil {
		t.Fatalf("WriteToHTTP failed: %v", err)
	}
}

func TestWriteToHTTP_Signatures(t *testing.T) {
	// Simple test to verify HMAC logic directly
	data := "some content"
	secret := "secret"
	sig := calculateHMAC(data, secret)
	if len(sig) != 64 {
		t.Errorf("Expected 64-char hex signature for SHA-256, got %d", len(sig))
	}
}

func TestWriteToHTTP_JSON(t *testing.T) {
	// Disable TLS verification for self-signed cert from httptest
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// 1. Setup a mock server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)
		var payload JSONPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("Failed to decode JSON payload: %v", err)
		}

		checkField := func(name string, b64Content string, signature string, expectedRaw string) {
			decoded, err := base64.StdEncoding.DecodeString(b64Content)
			if err != nil {
				t.Errorf("Failed to decode base64 for %s: %v", name, err)
			}

			if signature == "" {
				t.Errorf("Missing signature for %s", name)
			}

			expectedSig := calculateHMAC(b64Content, "json-secret")
			if signature != expectedSig {
				t.Errorf("Signature mismatch for %s. Got %s, want %s", name, signature, expectedSig)
			}

			if !strings.Contains(string(decoded), expectedRaw) {
				t.Errorf("Content mismatch for %s. Got %s, want %s", name, string(decoded), expectedRaw)
			}
		}

		checkField("json", payload.JSON, payload.JSONSignature, "testuser-json")
		checkField("md", payload.MD, payload.MDSignature, "# Test JSON Markdown")
		checkField("log", payload.Log, payload.LogSignature, "test json log")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// 2. Prepare test data
	res := report.FinalResult{
		Structured: report.FinalReport{
			Username: "testuser-json",
		},
		Markdown: "# Test JSON Markdown",
		Log:      "test json log",
	}

	config := &playbook.ReportDestinationConfig{
		Format:          playbook.ReportFormatJSON,
		URL:             server.URL,
		SignatureSecret: "json-secret",
	}

	// 3. Execute
	err := WriteToHTTP(config, res)
	if err != nil {
		t.Fatalf("WriteToHTTP JSON failed: %v", err)
	}
}

func TestWriteToHTTP_ServerErrors(t *testing.T) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	config := &playbook.ReportDestinationConfig{
		URL: server.URL,
	}
	res := report.FinalResult{}
	err := WriteToHTTP(config, res)
	if err == nil {
		t.Error("Expected error for 500 response, got none")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestWriteToHTTP_InvalidURL(t *testing.T) {
	config := &playbook.ReportDestinationConfig{
		URL: "https://   invalid", // Spaces make it invalid for NewRequest
	}
	res := report.FinalResult{}
	err := WriteToHTTP(config, res)
	if err == nil {
		t.Error("Expected error for invalid URL, got none")
	}
	if !strings.Contains(err.Error(), "failed to create HTTP request") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestWriteToHTTP_InsecureURL(t *testing.T) {
	config := &playbook.ReportDestinationConfig{
		URL: "http://insecure.example.com",
	}
	res := report.FinalResult{}
	err := WriteToHTTP(config, res)
	if err == nil {
		t.Error("Expected error for insecure URL, got none")
	}
	if !strings.Contains(err.Error(), "insecure HTTP report submission is not allowed") {
		t.Errorf("Unexpected error message: %v", err)
	}
}
