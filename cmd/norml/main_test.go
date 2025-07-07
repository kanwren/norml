package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// discardLogger returns a logger that discards all output
func discardLogger() *log.Logger {
	return log.New(io.Discard, "", 0)
}

func TestRun_Version(t *testing.T) {
	t.Parallel()

	version := Version()

	var logOutput bytes.Buffer
	logger := log.New(&logOutput, "", 0)

	var stdout bytes.Buffer
	stdin := strings.NewReader("")

	if err := run(t.Context(), logger, stdin, &stdout, []string{"-version"}); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, version) {
		t.Errorf("expected version output to contain version '%s', got: %s", version, output)
	}
}

func TestRun_StdinToStdout(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple yaml",
			input:    "key: value\n",
			expected: "key: value\n",
		},
		{
			name: "nested yaml",
			input: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  labels:
    app: test
`,
			expected: `apiVersion: v1
kind: Pod
metadata:
  labels:
    app: test
  name: test-pod
`,
		},
		{
			name:     "empty input",
			input:    "",
			expected: "null\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stdin := strings.NewReader(tc.input)
			var stdout bytes.Buffer

			logger := discardLogger()
			ctx := t.Context()
			if err := run(ctx, logger, stdin, &stdout, []string{}); err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			result := stdout.String()
			if result != tc.expected {
				t.Errorf("expected output %q, but got %q", tc.expected, result)
			}
		})
	}
}

func TestRun_SingleFileToStdout(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.yaml")

	input := `metadata:
  name: test-pod
  labels:
    app: test
apiVersion: v1
kind: Pod
`

	expected := `apiVersion: v1
kind: Pod
metadata:
  labels:
    app: test
  name: test-pod
`

	if err := os.WriteFile(filename, []byte(input), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	logger := discardLogger()
	ctx := t.Context()
	if err := run(ctx, logger, stdin, &stdout, []string{filename}); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	result := stdout.String()
	if result != expected {
		t.Errorf("expected output %q, but got %q", expected, result)
	}
}

func TestRun_MultipleFilesToStdout(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "test1.yaml")
	file2 := filepath.Join(tmpDir, "test2.yaml")

	input1 := `key1: value1
key2: value2
`
	input2 := `key3: value3
key4: value4
`

	expected := `key1: value1
key2: value2
---
key3: value3
key4: value4
`

	if err := os.WriteFile(file1, []byte(input1), 0644); err != nil {
		t.Fatalf("failed to write test file 1: %v", err)
	}
	if err := os.WriteFile(file2, []byte(input2), 0644); err != nil {
		t.Fatalf("failed to write test file 2: %v", err)
	}

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	logger := discardLogger()
	ctx := t.Context()
	if err := run(ctx, logger, stdin, &stdout, []string{file1, file2}); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	result := stdout.String()
	if result != expected {
		t.Errorf("expected output %q, but got %q", expected, result)
	}
}

func TestRun_InPlaceProcessing(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.yaml")

	input := `metadata:
  name: test-pod
  labels:
    app: test
apiVersion: v1
kind: Pod
`

	expected := `apiVersion: v1
kind: Pod
metadata:
  labels:
    app: test
  name: test-pod
`

	if err := os.WriteFile(filename, []byte(input), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	logger := discardLogger()
	ctx := t.Context()
	if err := run(ctx, logger, stdin, &stdout, []string{"-i", filename}); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read modified file: %v", err)
	}

	result := string(content)
	if result != expected {
		t.Errorf("expected file content %q, but got %q", expected, result)
	}
}

func TestRun_InPlaceMultipleFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "test1.yaml")
	file2 := filepath.Join(tmpDir, "test2.yaml")

	input1 := `key2: value2
key1: value1
`
	input2 := `key4: value4
key3: value3
`

	expected1 := `key1: value1
key2: value2
`
	expected2 := `key3: value3
key4: value4
`

	if err := os.WriteFile(file1, []byte(input1), 0644); err != nil {
		t.Fatalf("failed to write test file 1: %v", err)
	}
	if err := os.WriteFile(file2, []byte(input2), 0644); err != nil {
		t.Fatalf("failed to write test file 2: %v", err)
	}

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	logger := discardLogger()
	ctx := t.Context()
	if err := run(ctx, logger, stdin, &stdout, []string{"-i", file1, file2}); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	content1, err := os.ReadFile(file1)
	if err != nil {
		t.Fatalf("failed to read modified file 1: %v", err)
	}
	if string(content1) != expected1 {
		t.Errorf("expected file 1 content %q, but got %q", expected1, string(content1))
	}

	content2, err := os.ReadFile(file2)
	if err != nil {
		t.Fatalf("failed to read modified file 2: %v", err)
	}
	if string(content2) != expected2 {
		t.Errorf("expected file 2 content %q, but got %q", expected2, string(content2))
	}
}

func TestRun_VerboseMode(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.yaml")

	input := `key: value
`

	if err := os.WriteFile(filename, []byte(input), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	var logOutput bytes.Buffer
	logger := log.New(&logOutput, "", 0)

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	ctx := t.Context()
	if err := run(ctx, logger, stdin, &stdout, []string{"-v", "-i", filename}); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	logString := logOutput.String()
	if logString == "" {
		t.Errorf("expected logger output, got: %s", logString)
	}
}

func TestRun_WorkersFlag(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	var files []string
	for i := 0; i < 5; i++ {
		filename := filepath.Join(tmpDir, fmt.Sprintf("test%d.yaml", i))
		content := fmt.Sprintf("key%d: value%d\n", i, i)
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file %d: %v", i, err)
		}
		files = append(files, filename)
	}

	logger := discardLogger()
	ctx := t.Context()

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	args := append([]string{"-j", "2", "-i"}, files...)
	if err := run(ctx, logger, stdin, &stdout, args); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	for i, filename := range files {
		content, err := os.ReadFile(filename)
		if err != nil {
			t.Errorf("failed to read file %d: %v", i, err)
		}
		expected := fmt.Sprintf("key%d: value%d\n", i, i)
		if string(content) != expected {
			t.Errorf("file %d content mismatch. Expected: %s, Got: %s", i, expected, string(content))
		}
	}
}

func TestRun_ErrorNonExistentFile(t *testing.T) {
	t.Parallel()

	logger := discardLogger()

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	err := run(t.Context(), logger, stdin, &stdout, []string{"nonexistent.yaml"})
	if err == nil {
		t.Error("expected error for non-existent file, but got none")
	}
}

func TestRun_ErrorInvalidYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "invalid.yaml")

	invalidYAML := `key: value
  invalid: indentation
`

	if err := os.WriteFile(filename, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("failed to write invalid YAML file: %v", err)
	}

	logger := discardLogger()

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	if err := run(t.Context(), logger, stdin, &stdout, []string{filename}); err == nil {
		t.Error("expected error for invalid YAML, but got none")
	}
}

func TestRun_WorkerCountValidation(t *testing.T) {
	t.Parallel()

	logger := discardLogger()
	ctx := t.Context()

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	if err := run(ctx, logger, stdin, &stdout, []string{"-j", "0"}); err != nil {
		t.Errorf("worker count of 0 should be handled gracefully, got: %v", err)
	}

	if err := run(ctx, logger, stdin, &stdout, []string{"-j", "-1"}); err != nil {
		t.Errorf("negative worker count should be handled gracefully, got: %v", err)
	}
}

func TestRun_HelpFlag(t *testing.T) {
	t.Parallel()

	logger := discardLogger()
	ctx := t.Context()

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	if err := run(ctx, logger, stdin, &stdout, []string{"-h"}); err != nil {
		t.Errorf("help flag should not return error, got: %v", err)
	}

	stdout.Reset()
	if err := run(ctx, logger, stdin, &stdout, []string{"--help"}); err != nil {
		t.Errorf("help flag should not return error, got: %v", err)
	}
}

func TestRun_InvalidFlagSyntax(t *testing.T) {
	t.Parallel()

	const flagErrCode = 2

	logger := discardLogger()
	ctx := t.Context()

	testCases := []struct {
		name string
		args []string
	}{
		{
			name: "unknown flag",
			args: []string{"-unknown"},
		},
		{
			name: "invalid flag format",
			args: []string{"-j", "invalid"},
		},
		{
			name: "missing flag argument",
			args: []string{"-j"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stdin := strings.NewReader("")
			var stdout bytes.Buffer

			err := run(ctx, logger, stdin, &stdout, tc.args)
			if err == nil {
				t.Error("expected error for invalid flag syntax, but got none")
			}

			var exitErr *errWithExitCode
			if !errors.As(err, &exitErr) {
				t.Errorf("expected errWithExitCode, got %T: %v", err, err)
			}
			if exitErr.Code != flagErrCode {
				t.Errorf("expected exit code %d, got %d", flagErrCode, exitErr.Code)
			}
		})
	}
}

func TestRun_ConcurrentProcessing(t *testing.T) {
	t.Parallel()

	logger := discardLogger()

	const fileCount = 10

	workerCounts := []int{1, 2, 5, 10}
	for _, workers := range workerCounts {
		t.Run(fmt.Sprintf("workers_%d", workers), func(t *testing.T) {
			t.Parallel()

			var files []string
			tmpDir := t.TempDir()

			for i := range fileCount {
				filename := filepath.Join(tmpDir, fmt.Sprintf("test%d.yaml", i))
				content := fmt.Sprintf(`key%d: value%d
nested:
  subkey%d: subvalue%d
`, i, i, i, i)
				if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
					t.Fatalf("failed to recreate test file %d: %v", i, err)
				}
				files = append(files, filename)
			}

			stdin := strings.NewReader("")
			var stdout bytes.Buffer

			start := time.Now()
			args := append([]string{"-j", fmt.Sprintf("%d", workers), "-i"}, files...)
			if err := run(t.Context(), logger, stdin, &stdout, args); err != nil {
				t.Errorf("expected no error with %d workers, got: %v", workers, err)
			}
			t.Logf("processing %d files with %d workers took %v", fileCount, workers, time.Since(start))

			for i, filename := range files {
				content, err := os.ReadFile(filename)
				if err != nil {
					t.Errorf("failed to read file %d: %v", i, err)
				}
				expected := fmt.Sprintf(`key%d: value%d
nested:
  subkey%d: subvalue%d
`, i, i, i, i)
				if string(content) != expected {
					t.Errorf("file %d content mismatch after processing with %d workers", i, workers)
				}
			}
		})
	}
}

func TestRun_EmptyFileList(t *testing.T) {
	t.Parallel()

	logger := discardLogger()

	input := "key: value\n"
	stdin := strings.NewReader(input)
	var stdout bytes.Buffer

	if err := run(t.Context(), logger, stdin, &stdout, []string{}); err != nil {
		t.Errorf("expected no error when reading from stdin, got: %v", err)
	}

	result := stdout.String()
	expected := "key: value\n"
	if result != expected {
		t.Errorf("expected output %q, but got %q", expected, result)
	}
}

func TestRun_ContextCancellation(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.yaml")

	input := `key: value
`

	if err := os.WriteFile(filename, []byte(input), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	logger := discardLogger()

	stdin := strings.NewReader("")
	var stdout bytes.Buffer

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := run(ctx, logger, stdin, &stdout, []string{"-i", filename}); err != nil {
		t.Logf("context cancellation resulted in error (expected): %v", err)
	}
}

func TestNormalizeTo_EmptyResultsChannel(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.yaml")

	input := `key: value
`

	if err := os.WriteFile(filename, []byte(input), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	logger := discardLogger()

	var output bytes.Buffer
	if err := normalizeTo(t.Context(), logger, &output, []string{filename}, 1); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	result := output.String()
	expected := `key: value
`
	if result != expected {
		t.Errorf("expected output %q, but got %q", expected, result)
	}
}

func TestNormalizeInPlace_SingleFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.yaml")

	input := `key2: value2
key1: value1
`

	expected := `key1: value1
key2: value2
`

	if err := os.WriteFile(filename, []byte(input), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	logger := discardLogger()

	if err := normalizeInPlace(t.Context(), logger, []string{filename}, 1); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read modified file: %v", err)
	}

	result := string(content)
	if result != expected {
		t.Errorf("expected file content %q, but got %q", expected, result)
	}
}

func TestNormalizeInPlace_MultipleFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "test1.yaml")
	file2 := filepath.Join(tmpDir, "test2.yaml")

	input1 := `key2: value2
key1: value1
`
	input2 := `key4: value4
key3: value3
`

	expected1 := `key1: value1
key2: value2
`
	expected2 := `key3: value3
key4: value4
`

	if err := os.WriteFile(file1, []byte(input1), 0644); err != nil {
		t.Fatalf("failed to write test file 1: %v", err)
	}
	if err := os.WriteFile(file2, []byte(input2), 0644); err != nil {
		t.Fatalf("failed to write test file 2: %v", err)
	}

	logger := discardLogger()

	if err := normalizeInPlace(t.Context(), logger, []string{file1, file2}, 2); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	content1, err := os.ReadFile(file1)
	if err != nil {
		t.Fatalf("failed to read modified file 1: %v", err)
	}
	if string(content1) != expected1 {
		t.Errorf("expected file 1 content %q, but got %q", expected1, string(content1))
	}

	content2, err := os.ReadFile(file2)
	if err != nil {
		t.Fatalf("failed to read modified file 2: %v", err)
	}
	if string(content2) != expected2 {
		t.Errorf("expected file 2 content %q, but got %q", expected2, string(content2))
	}
}
