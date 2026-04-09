package test_e2e

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/benedictjohannes/crobe/internal/reportwriter"
	"github.com/benedictjohannes/crobe/playbook"
	"github.com/benedictjohannes/crobe/report"
	"gopkg.in/yaml.v3"
)

func TestHTTPSScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// 1. Build the binary once
	tmpDir, err := os.MkdirTemp("", "crobe-e2e-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	probeBin := filepath.Join(tmpDir, "crobe")

	fmt.Println("🔨 Building probe binary for E2E test...")
	buildProbe := exec.Command("go", "build", "-o", probeBin, "../cmd/probe")
	if out, err := buildProbe.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build probe: %v\n%s", err, string(out))
	}

	// 2. Define the playbook in code
	basePlaybook := playbook.Playbook{
		Title: "E2E HTTPS Test Playbook",
		Sections: []playbook.Section{
			{
				Title:       "Network Checks",
				Description: []string{"Verify remote connectivity and reporting"},
				Assertions: []playbook.Assertion{
					{
						Code:            "NET-001",
						Title:           "Local Echo",
						Description:     "Run a simple echo command to verify execution",
						PassDescription: "Echo successful",
						FailDescription: "Echo failed",
						Cmds: []playbook.Cmd{
							{
								Exec: playbook.Exec{
									Script: "echo 'E2E HTTPS Test'",
								},
							},
						},
					},
				},
			},
		},
		ReportDestination: playbook.ReportDestinationHTTPS,
	}

	const sigSecret = "e2e-debug-secret-key"

	tests := []struct {
		name                string
		playbookContentType string
		submissionFormat    playbook.ReportFormat
		marshal             func(interface{}) ([]byte, error)
	}{
		{
			name:                "YAMLServing_MultipartSubmission",
			playbookContentType: "application/x-yaml",
			submissionFormat:    playbook.ReportFormatMultipart,
			marshal:             yaml.Marshal,
		},
		{
			name:                "YAMLServing_JsonSubmission",
			playbookContentType: "application/x-yaml",
			submissionFormat:    playbook.ReportFormatJSON,
			marshal:             yaml.Marshal,
		},
		{
			name:                "JSONServing_MultipartSubmission",
			playbookContentType: "application/json",
			submissionFormat:    playbook.ReportFormatMultipart,
			marshal:             json.Marshal,
		},
		{
			name:                "JSONServing_JsonSubmission",
			playbookContentType: "application/json",
			submissionFormat:    playbook.ReportFormatJSON,
			marshal:             json.Marshal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 3. Setup HTTPS Server for this subtest
			var reportReceived bool
			var receivedContentType string
			var receivedBody []byte
			var reportMu sync.Mutex
			var authHeaderReceived string

			// We'll prepare the content later once we have the server URL
			var playbookContent []byte

			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/playbook" {
					authHeaderReceived = r.Header.Get("Authorization")
					w.Header().Set("Content-Type", tt.playbookContentType)
					w.Write(playbookContent)
					return
				}
				if r.Method == "POST" && r.URL.Path == "/report" {
					body, err := io.ReadAll(r.Body)
					if err != nil {
						http.Error(w, "Failed to read body", http.StatusInternalServerError)
						return
					}
					reportMu.Lock()
					reportReceived = true
					receivedContentType = r.Header.Get("Content-Type")
					receivedBody = body
					reportMu.Unlock()
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("OK"))
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			// 4. Update playbook with server URL and marshal it
			pb := basePlaybook
			pb.ReportDestinationHTTPS = &playbook.ReportDestinationConfig{
				URL:             server.URL + "/report",
				Format:          tt.submissionFormat,
				SignatureSecret: sigSecret,
			}

			content, err := tt.marshal(pb)
			if err != nil {
				t.Fatalf("Failed to marshal playbook: %v", err)
			}
			playbookContent = content

			// 5. Export Server Certificate
			cert := server.Certificate()
			certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
			certFile := filepath.Join(tmpDir, "server-"+tt.name+".crt")
			if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
				t.Fatal(err)
			}

			// 6. Run the Probe
			fmt.Printf("🚀 Running E2E HTTPS scenario (%s) against %s\n", tt.name, server.URL)
			probeCmd := exec.Command(probeBin, "-H", "Authorization: Bearer e2e-token", server.URL+"/playbook")

			probeCmd.Env = append(os.Environ(), "SSL_CERT_FILE="+certFile)

			var stdout, stderr bytes.Buffer
			probeCmd.Stdout = &stdout
			probeCmd.Stderr = &stderr

			err = probeCmd.Run()
			if err != nil {
				t.Errorf("Probe execution failed: %v", err)
				t.Logf("STDOUT: %s", stdout.String())
				t.Logf("STDERR: %s", stderr.String())
			}

			// 7. Verifications
			if authHeaderReceived != "Bearer e2e-token" {
				t.Errorf("Expected Authorization header 'Bearer e2e-token', got '%s'", authHeaderReceived)
			}

			reportMu.Lock()
			received := reportReceived
			contentType := receivedContentType
			body := receivedBody
			reportMu.Unlock()

			if !received {
				t.Fatalf("Server never received the report submission")
			}

			if !strings.Contains(stdout.String(), "✅ Submission Complete!") {
				t.Errorf("Probe output did not indicate successful submission")
			}

			// 8. Verify Submission Content
			if tt.submissionFormat == playbook.ReportFormatJSON {
				if !strings.HasPrefix(contentType, "application/json") {
					t.Errorf("Expected application/json content type, got %s", contentType)
				}
				var payload reportwriter.JSONPayload
				if err := json.Unmarshal(body, &payload); err != nil {
					t.Fatalf("Failed to unmarshal JSON submission: %v", err)
				}

				// Verify Signatures
				verifySig := func(name, data, sig string) {
					if sig == "" {
						t.Errorf("Missing signature for %s", name)
						return
					}
					h := hmac.New(sha256.New, []byte(sigSecret))
					h.Write([]byte(data))
					expected := hex.EncodeToString(h.Sum(nil))
					if sig != expected {
						t.Errorf("Invalid signature for %s. Expected %s, got %s", name, expected, sig)
					}
				}

				verifySig("report.json", payload.JSON, payload.JSONSignature)
				verifySig("report.md", payload.MD, payload.MDSignature)
				verifySig("report.log", payload.Log, payload.LogSignature)

				// Verify Base64 content
				reportJSON, err := base64.StdEncoding.DecodeString(payload.JSON)
				if err != nil {
					t.Fatalf("Failed to decode report.json: %v", err)
				}
				var finalReport report.FinalReport
				if err := json.Unmarshal(reportJSON, &finalReport); err != nil {
					t.Fatalf("Failed to unmarshal final report: %v", err)
				}
				if finalReport.Stats.Passed != 1 {
					t.Errorf("Expected 1 passed assertion, got %d", finalReport.Stats.Passed)
				}
			} else {
				// Multipart
				mediaType, params, err := mime.ParseMediaType(contentType)
				if err != nil || mediaType != "multipart/form-data" {
					t.Fatalf("Expected multipart/form-data, got %s", contentType)
				}
				mr := multipart.NewReader(bytes.NewReader(body), params["boundary"])
				parts := make(map[string]string)
				for {
					p, err := mr.NextPart()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Fatalf("Failed to read multiparty: %v", err)
					}
					content, err := io.ReadAll(p)
					if err != nil {
						t.Fatalf("Failed to read part content: %v", err)
					}
					parts[p.FormName()] = string(content)

					if p.FormName() == "report.json" {
						if p.Header.Get("Content-Transfer-Encoding") != "base64" {
							t.Errorf("Expected Content-Transfer-Encoding: base64, got %s", p.Header.Get("Content-Transfer-Encoding"))
						}
						decoded, err := base64.StdEncoding.DecodeString(string(content))
						if err != nil {
							t.Fatalf("Failed to decode report.json part: %v", err)
						}
						var finalReport report.FinalReport
						if err := json.Unmarshal(decoded, &finalReport); err != nil {
							t.Fatalf("Failed to unmarshal final report from multipart: %v", err)
						}
						if finalReport.Stats.Passed != 1 {
							t.Errorf("Expected 1 passed assertion, got %d", finalReport.Stats.Passed)
						}
					}
				}

				// Verify Multipart Signatures
				verifyPartSig := func(filename string) {
					data, ok := parts[filename]
					if !ok {
						t.Errorf("Missing part %s", filename)
						return
					}
					sig, ok := parts[filename+".signature.txt"]
					if !ok {
						t.Errorf("Missing signature part for %s", filename)
						return
					}

					h := hmac.New(sha256.New, []byte(sigSecret))
					h.Write([]byte(data))
					expected := hex.EncodeToString(h.Sum(nil))
					if sig != expected {
						t.Errorf("Invalid signature for %s. Expected %s, got %s", filename, expected, sig)
					}
				}

				verifyPartSig("report.json")
				verifyPartSig("report.md")
				verifyPartSig("report.log")
			}
		})
	}

	fmt.Println("✅ HTTPS E2E Dual-Format Scenarios Passed!")
}
