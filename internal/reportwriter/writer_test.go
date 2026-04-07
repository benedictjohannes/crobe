package reportwriter

import (
	"compliance-probe/playbook"
	"compliance-probe/report"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDispatchReport(t *testing.T) {
	res := report.FinalResult{
		Structured: report.FinalReport{
			Username: "testuser",
		},
		Markdown: "# Test",
		Log:      "test log",
	}

	t.Run("folder destination", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "dispatch-folder-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		oldDir := DefaultReportsDir
		DefaultReportsDir = tmpDir
		defer func() { DefaultReportsDir = oldDir }()

		config := &playbook.ReportConfig{
			ReportDestination: playbook.ReportDestinationFolder,
		}
		err = DispatchReport(config, res)
		if err != nil {
			t.Fatalf("DispatchReport to folder failed: %v", err)
		}

		// Verify files exist in tmpDir
		files, _ := os.ReadDir(tmpDir)
		if len(files) != 3 {
			t.Errorf("Expected 3 files in reports directory, got %d", len(files))
		}
	})

	t.Run("unknown destination", func(t *testing.T) {
		config := &playbook.ReportConfig{
			ReportDestination: "somewhere-else",
		}
		err := DispatchReport(config, res)
		if err == nil || !strings.Contains(err.Error(), "unknown reportDestination") {
			t.Errorf("Expected error for unknown destination, got %v", err)
		}
	})

	t.Run("https missing URL", func(t *testing.T) {
		config := &playbook.ReportConfig{
			ReportDestination: playbook.ReportDestinationHTTPS,
		}
		err := DispatchReport(config, res)
		if err == nil || !strings.Contains(err.Error(), "reportDestinationHttps is missing") {
			t.Errorf("Expected error for missing URL, got %v", err)
		}
	})

	t.Run("https success", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		config := &playbook.ReportConfig{
			ReportDestination: playbook.ReportDestinationHTTPS,
			ReportDestinationHTTPS: &playbook.ReportDestinationConfig{
				URL: server.URL,
			},
		}
		err := DispatchReport(config, res)
		if err != nil {
			t.Fatalf("DispatchReport to HTTPS failed: %v", err)
		}
	})
}

func TestWriteToFolder(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "reportwriter-write-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	res := report.FinalResult{
		Structured: report.FinalReport{
			Username: "testuser",
		},
		Markdown: "# Test Markdown",
		Log:      "test log",
	}

	err = WriteToFolder(tmpDir, res)
	if err != nil {
		t.Fatalf("WriteToFolder failed: %v", err)
	}

	// Verify files exist
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 3 {
		t.Errorf("Expected 3 files in reports directory, got %d", len(files))
	}

	foundLog := false
	foundMD := false
	foundJSON := false

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".log") {
			foundLog = true
		}
		if strings.HasSuffix(f.Name(), ".md") {
			foundMD = true
		}
		if strings.HasSuffix(f.Name(), ".json") {
			foundJSON = true
		}
	}

	if !foundLog || !foundMD || !foundJSON {
		t.Errorf("Missing report files: log=%v, md=%v, json=%v", foundLog, foundMD, foundJSON)
	}
}

func TestWriteToFolder_Errors(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "reportwriter-error-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file where the directory should be
	errPath := filepath.Join(tmpDir, "somefile")
	os.WriteFile(errPath, []byte("not a directory"), 0644)

	res := report.FinalResult{}
	err = WriteToFolder(errPath, res)
	if err == nil {
		t.Errorf("Expected error when target directory is a file, got nil")
	}
}

func TestWriteToHTTP_Errors(t *testing.T) {
	res := report.FinalResult{}

	t.Run("insecure URL", func(t *testing.T) {
		config := &playbook.ReportDestinationConfig{
			URL: "http://example.com",
		}
		err := WriteToHTTP(config, res)
		if err == nil || !strings.Contains(err.Error(), "insecure HTTP report submission is not allowed") {
			t.Errorf("Expected error for insecure URL, got %v", err)
		}
	})

	t.Run("invalid URL", func(t *testing.T) {
		config := &playbook.ReportDestinationConfig{
			URL: "https://invalid space",
		}
		err := WriteToHTTP(config, res)
		if err == nil {
			t.Errorf("Expected error for invalid URL, got nil")
		}
	})

	t.Run("server error status", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal error"))
		}))
		defer server.Close()

		config := &playbook.ReportDestinationConfig{
			URL: server.URL,
		}
		err := WriteToHTTP(config, res)
		if err == nil || !strings.Contains(err.Error(), "status 500") {
			t.Errorf("Expected error for 500 status, got %v", err)
		}
	})

	t.Run("connection error", func(t *testing.T) {
		// Create a server and immediately close it
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := server.URL
		server.Close()

		config := &playbook.ReportDestinationConfig{
			URL: url,
		}
		err := WriteToHTTP(config, res)
		if err == nil {
			t.Errorf("Expected connection error, got nil")
		}
	})
}
