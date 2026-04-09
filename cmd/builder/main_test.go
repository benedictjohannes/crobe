package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuilderRun(t *testing.T) {
	// 1. Test missing playbook
	if code := run([]string{}); code != 1 {
		t.Errorf("Expected exit code 1 for missing playbook, got %d", code)
	}

	// 2. Test --schema
	if code := run([]string{"--schema"}); code != 0 {
		t.Errorf("Expected exit code 0 for --schema, got %d", code)
	}

	// 3. Test --preprocess (invalid input)
	if code := run([]string{"--preprocess"}); code != 1 {
		t.Errorf("Expected exit code 1 for --preprocess without --input, got %d", code)
	}

	// 4. Test --preprocess (happy path)
	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "raw.yaml")
	outputPath := filepath.Join(tmpDir, "baked.yaml")
	inputContent := `
title: Raw
sections:
  - title: S1
    assertions:
      - code: T1
        title: T1
        cmds:
          - exec:
              script: echo hello
`
	if err := os.WriteFile(inputPath, []byte(inputContent), 0644); err != nil {
		t.Fatal(err)
	}

	if code := run([]string{"--preprocess", "--input", inputPath, "--output", outputPath}); code != 0 {
		t.Errorf("Expected exit code 0 for --preprocess happy path, got %d", code)
	}

	// 6. Test --preprocess with invalid input file
	if code := run([]string{"--preprocess", "--input", "non-existent.yaml", "--output", outputPath}); code != 1 {
		t.Errorf("Expected exit code 1 for --preprocess with non-existent input, got %d", code)
	}

	// 7. Test normal run with validation error (duplicate codes)
	invalidPbPath := filepath.Join(tmpDir, "invalid_val.yaml")
	invalidContent := `
title: Invalid
sections:
  - title: S1
    description: [D1]
    assertions:
      - code: T1
        title: T1
        cmds: [{exec: {script: echo}}]
      - code: T1
        title: T2
        cmds: [{exec: {script: echo}}]
`
	os.WriteFile(invalidPbPath, []byte(invalidContent), 0644)
	// 8. Happy path for normal run
	pbPath := filepath.Join(tmpDir, "test_run.yaml")
	pbContent := `
title: Test Run
sections:
  - title: S1
    assertions:
      - code: T1
        title: T1
        cmds: [{exec: {script: echo hello}}]
`
	os.WriteFile(pbPath, []byte(pbContent), 0644)
	if code := run([]string{"-folder", tmpDir, pbPath}); code != 0 {
		t.Errorf("Expected exit code 0 for normal run, got %d", code)
	}

	// 9. Test --preprocess failure (invalid YAML)
	invalidInputPath := filepath.Join(tmpDir, "invalid_raw.yaml")
	os.WriteFile(invalidInputPath, []byte("invalid: : :"), 0644)
	if code := run([]string{"--preprocess", "--input", invalidInputPath, "--output", outputPath}); code != 1 {
		t.Errorf("Expected exit code 1 for --preprocess with invalid YAML, got %d", code)
	}

	// 10. Test transpilation failure (missing funcFile)
	missingFuncFilePbPath := filepath.Join(tmpDir, "missing_func.yaml")
	missingFuncFileContent := `
title: Missing Func
sections:
  - title: S1
    assertions:
      - code: T1
        title: T1
        cmds:
          - exec:
              funcFile: missing.js
`
	os.WriteFile(missingFuncFilePbPath, []byte(missingFuncFileContent), 0644)
	if code := run([]string{"-folder", tmpDir, missingFuncFilePbPath}); code != 1 {
		t.Errorf("Expected exit code 1 for missing funcFile, got %d", code)
	}

	// 11. Test custom header flag
	if code := run([]string{"-H", "Auth: token", "--folder", tmpDir, pbPath}); code != 0 {
		t.Errorf("Expected exit code 0 with custom header, got %d", code)
	}

	// 12. Test invalid flag
	if code := run([]string{"--invalid-flag"}); code != 1 {
		t.Errorf("Expected exit code 1 for invalid flag, got %d", code)
	}

	// 13. Test failing assertion
	failingPbPath := filepath.Join(tmpDir, "failing.yaml")
	failingPbContent := `
title: Failing
sections:
  - title: S1
    assertions:
      - code: F1
        title: F1
        cmds: [{exec: {script: exit 1}}]
`
	os.WriteFile(failingPbPath, []byte(failingPbContent), 0644)
	if code := run([]string{"-folder", tmpDir, failingPbPath}); code != 1 {
		t.Errorf("Expected exit code 1 for failing assertion, got %d", code)
	}

	// 14. Test DispatchReport failure
	dispatchErrPbPath := filepath.Join(tmpDir, "dispatch_err.yaml")
	dispatchErrPbContent := `
title: Dispatch Error
reportDestination: https
sections:
  - title: S1
    assertions:
      - code: T1
        title: T1
        cmds: [{exec: {script: echo}}]
`
	os.WriteFile(dispatchErrPbPath, []byte(dispatchErrPbContent), 0644)
	// Do NOT provide -folder flag so it doesn't override to folder
	if code := run([]string{dispatchErrPbPath}); code != 1 {
		t.Errorf("Expected exit code 1 for dispatch error, got %d", code)
	}

	// 15. Test LoadConfig failure
	if code := run([]string{"non-existent.yaml"}); code != 1 {
		t.Errorf("Expected exit code 1 for non-existent playbook, got %d", code)
	}
}

