package configsource

import (
	"github.com/benedictjohannes/ComplianceProbe/playbook"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// GetConfigSource detects the config source from path or defaults to playbook.yaml if it exists.
func GetConfigSource(path string) string {
	if path != "" {
		return path
	}
	if fileExists("playbook.yaml") {
		return "playbook.yaml"
	}
	return ""
}

// LoadConfig loads the playbook from either a local file or an HTTPS URL.
func LoadConfig(path string) (*playbook.ReportConfig, []byte, error) {
	var data []byte
	var err error

	if strings.HasPrefix(path, "http://") {
		return nil, nil, fmt.Errorf("insecure HTTP connections are not allowed: %s", path)
	}

	if strings.HasPrefix(path, "https://") {
		data, err = fetchHttpsPlaybook(path)
	} else {
		data, err = os.ReadFile(path)
	}

	if err != nil {
		return nil, nil, err
	}

	var config playbook.ReportConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &config, data, nil
}

func fetchHttpsPlaybook(url string) ([]byte, error) {
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch remote playbook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch remote playbook: status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
