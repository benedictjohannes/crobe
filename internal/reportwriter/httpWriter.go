package reportwriter

import (
	"bytes"
	"compliance-probe/playbook"
	"compliance-probe/report"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"
)

type JSONPayload struct {
	JSON          string `json:"json"`
	JSONSignature string `json:"jsonSignature,omitempty"`
	MD            string `json:"md"`
	MDSignature   string `json:"mdSignature,omitempty"`
	Log           string `json:"log"`
	LogSignature  string `json:"logSignature,omitempty"`
}

type reportFile struct {
	filename    string
	contentType string
	content     []byte
}

// WriteToHTTP sends the report to a remote server.
func WriteToHTTP(config *playbook.ReportDestinationConfig, res report.FinalResult) (err error) {
	if !strings.HasPrefix(config.URL, "https://") {
		return fmt.Errorf("insecure HTTP report submission is not allowed: %s", config.URL)
	}

	jsonBytes, err := json.MarshalIndent(res.Structured, "", "  ")
	if err != nil {
		return
	}

	files := []reportFile{
		{"report.json", "application/json", jsonBytes},
		{"report.md", "text/markdown", []byte(res.Markdown)},
		{"report.log", "text/plain", []byte(res.Log)},
	}

	var body io.Reader
	var contentType string

	if config.Format == playbook.ReportFormatJSON {
		jsonB64 := base64.StdEncoding.EncodeToString(jsonBytes)
		mdB64 := base64.StdEncoding.EncodeToString([]byte(res.Markdown))
		logB64 := base64.StdEncoding.EncodeToString([]byte(res.Log))

		payload := JSONPayload{
			JSON: jsonB64,
			MD:   mdB64,
			Log:  logB64,
		}

		if config.SignatureSecret != "" {
			payload.JSONSignature = calculateHMAC(jsonB64, config.SignatureSecret)
			payload.MDSignature = calculateHMAC(mdB64, config.SignatureSecret)
			payload.LogSignature = calculateHMAC(logB64, config.SignatureSecret)
		}

		jsonBody, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return err
		}
		body = bytes.NewReader(jsonBody)
		contentType = "application/json"
	} else {
		// Default to Multipart
		b := &bytes.Buffer{}
		writer := multipart.NewWriter(b)
		for _, f := range files {
			if err = addFilePart(writer, f.filename, f.contentType, f.content, config.SignatureSecret); err != nil {
				return err
			}
		}
		writer.Close()
		body = b
		contentType = writer.FormDataContentType()
	}

	req, err := http.NewRequest("POST", config.URL, body)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	for k, v := range config.AdditionalHeaders {
		req.Header.Set(k, v)
	}

	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	fmt.Printf("📤 Submitting report to: %s (Format: %s)\n", config.URL, config.Format)
	if config.Format == "" {
		fmt.Printf("📤 Note: defaulting to multipart format\n")
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to submit report: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to submit report: status %d. Response: %s", resp.StatusCode, string(respBody))
	}

	fmt.Printf("\n✅ Submission Complete!\n")
	fmt.Printf("📊 PASS: %d, FAIL: %d\n", res.Structured.Stats.Passed, res.Structured.Stats.Failed)
	fmt.Printf("✅ Status: %d\n", resp.StatusCode)
	return nil
}

func addFilePart(writer *multipart.Writer, filename string, contentType string, content []byte, secret string) error {
	// Base64 encode the content
	b64Content := base64.StdEncoding.EncodeToString(content)

	// 1. Add the Base64-encoded file part
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, filename, filename))
	h.Set("Content-Type", contentType)
	h.Set("Content-Transfer-Encoding", "base64")

	part, err := writer.CreatePart(h)
	if err != nil {
		return fmt.Errorf("failed to create multipart for %s: %w", filename, err)
	}
	_, err = part.Write([]byte(b64Content))
	if err != nil {
		return fmt.Errorf("failed to write multipart content for %s: %w", filename, err)
	}

	// 2. Add the signature part if secret is provided
	if secret != "" {
		sig := calculateHMAC(b64Content, secret)
		sigFilename := filename + ".signature.txt"
		sigPart, err := writer.CreateFormFile(sigFilename, sigFilename)
		if err != nil {
			return fmt.Errorf("failed to create signature part for %s: %w", filename, err)
		}
		_, err = sigPart.Write([]byte(sig))
		if err != nil {
			return fmt.Errorf("failed to write signature content for %s: %w", filename, err)
		}
	}

	return nil
}

func calculateHMAC(data string, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
