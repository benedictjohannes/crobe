package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProbeRun(t *testing.T) {
	// 1. Test missing playbook
	if code := run([]string{}); code != 1 {
		t.Errorf("Expected exit code 1 for missing playbook, got %d", code)
	}

	// 2. Test invalid flag
	if code := run([]string{"--invalid-flag"}); code != 1 {
		t.Errorf("Expected exit code 1 for invalid flag, got %d", code)
	}

	// 3. Happy path (using a minimal playbook)
	tmpDir := t.TempDir()
	pbPath := filepath.Join(tmpDir, "test.yaml")
	pbContent := `
title: Test
sections:
  - title: S1
    assertions:
      - code: T1
        title: T1
        cmds:
          - exec:
              script: echo hello
`
	if err := os.WriteFile(pbPath, []byte(pbContent), 0644); err != nil {
		t.Fatal(err)
	}

	if code := run([]string{"-folder", tmpDir, pbPath}); code != 0 {
		t.Errorf("Expected exit code 0 for happy path, got %d", code)
	}

	// 4. Test missing playbook file
	if code := run([]string{"-folder", tmpDir, "non-existent.yaml"}); code != 1 {
		t.Errorf("Expected exit code 1 for non-existent playbook, got %d", code)
	}

	// 5. Test invalid playbook content (YAML error)
	invalidPbPath := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(invalidPbPath, []byte("invalid: yaml: :"), 0644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"-folder", tmpDir, invalidPbPath}); code != 1 {
		t.Errorf("Expected exit code 1 for invalid playbook content, got %d", code)
	}

	// 6. Test validation error (e.g. funcFile in agent mode)
	validationErrPbPath := filepath.Join(tmpDir, "validation_err.yaml")
	validationErrPbContent := `
title: Test
sections:
  - title: S1
    assertions:
      - code: T1
        title: T1
        cmds:
          - exec:
              funcFile: some-script.js
`
	if err := os.WriteFile(validationErrPbPath, []byte(validationErrPbContent), 0644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"-folder", tmpDir, validationErrPbPath}); code != 1 {
		t.Errorf("Expected exit code 1 for validation error, got %d", code)
	}

	// 7. Test playbook with failing assertion
	failingPbPath := filepath.Join(tmpDir, "failing.yaml")
	failingPbContent := `
title: Test
sections:
  - title: S1
    assertions:
      - code: F1
        title: F1
        cmds:
          - exec:
              script: exit 1
`
	if err := os.WriteFile(failingPbPath, []byte(failingPbContent), 0644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"-folder", tmpDir, failingPbPath}); code != 1 {
		t.Errorf("Expected exit code 1 for failing assertion, got %d", code)
	}

	// 8. Test DispatchReport failure
	dispatchErrPbPath := filepath.Join(tmpDir, "dispatch_err.yaml")
	dispatchErrPbContent := `
title: Dispatch Error
reportDestination: https
sections:
  - title: S1
    assertions:
      - code: T1
        title: T1
        cmds:
          - exec:
              script: echo
`
	if err := os.WriteFile(dispatchErrPbPath, []byte(dispatchErrPbContent), 0644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{dispatchErrPbPath}); code != 1 {
		t.Errorf("Expected exit code 1 for dispatch error, got %d", code)
	}
}
