package normalizer

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

func TestNormalizeFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name: "simple key-value",
			input: `key: value
`,
			expected: `key: value
`,
		},
		{
			name: "nested structure",
			input: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  labels:
    app: test
spec:
  containers:
  - name: nginx
    image: nginx:latest
    ports:
    - containerPort: 80
`,
			expected: `apiVersion: v1
kind: Pod
metadata:
  labels:
    app: test
  name: test-pod
spec:
  containers:
  - image: nginx:latest
    name: nginx
    ports:
    - containerPort: 80
`,
		},
		{
			name: "array with mixed types",
			input: `items:
- string
- 42
- true
- null
- nested:
    key: value
`,
			expected: `items:
- string
- 42
- true
- null
- nested:
    key: value
`,
		},
		{
			name: "quoted strings",
			input: `name: "quoted string"
description: 'single quoted'
special: "with \"quotes\" inside"
`,
			expected: `description: single quoted
name: quoted string
special: with "quotes" inside
`,
		},
		{
			name: "multiline strings",
			input: `description: |
  This is a
  multiline string
  with line breaks
`,
			expected: `description: |
  This is a
  multiline string
  with line breaks
`,
		},
		{
			name: "invalid YAML",
			input: `key: value
  invalid: indentation
`,
			expectError: true,
		},
		{
			name:  "empty file",
			input: ``,
			expected: `null
`,
		},
		{
			name:  "only whitespace",
			input: `   `,
			expected: `null
`,
		},
		{
			name: "complex nested structure",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
        env:
        - name: NGINX_HOST
          value: "localhost"
        - name: NGINX_PORT
          value: "80"
        resources:
          limits:
            cpu: "500m"
            memory: "512Mi"
          requests:
            cpu: "250m"
            memory: "256Mi"
`,
			expected: `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: nginx
  name: nginx-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - env:
        - name: NGINX_HOST
          value: localhost
        - name: NGINX_PORT
          value: "80"
        image: nginx:1.14.2
        name: nginx
        ports:
        - containerPort: 80
        resources:
          limits:
            cpu: 500m
            memory: 512Mi
          requests:
            cpu: 250m
            memory: 256Mi
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			filename := filepath.Join(tmpDir, "test.yaml")

			if err := os.WriteFile(filename, []byte(tt.input), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			err := NormalizeFile(filename)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("NormalizeFile failed: %v", err)
			}

			content, err := os.ReadFile(filename)
			if err != nil {
				t.Fatalf("Failed to read normalized file: %v", err)
			}

			got := string(content)
			if got != tt.expected {
				t.Errorf("NormalizeFile() = %q, want %q", got, tt.expected)
			}

			var obj any
			if err := yaml.Unmarshal(content, &obj); err != nil {
				t.Errorf("Normalized output is not valid YAML: %v", err)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name: "simple key-value",
			input: `key: value
`,
			expected: `key: value
`,
		},
		{
			name: "nested structure",
			input: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
`,
			expected: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
`,
		},
		{
			name: "invalid YAML",
			input: `key: value
  invalid: indentation
`,
			expectError: true,
		},
		{
			name:  "empty input",
			input: ``,
			expected: `null
`,
		},
		{
			name:  "only whitespace",
			input: `   `,
			expected: `null
`,
		},
		{
			name: "numbers and booleans",
			input: `integer: 42
float: 3.14
boolean: true
null_value: null
`,
			expected: `boolean: true
float: 3.14
integer: 42
null_value: null
`,
		},
		{
			name: "arrays and objects",
			input: `array: [1, 2, 3]
object:
  nested: value
mixed:
  - item1
  - item2
  - nested:
      key: value
`,
			expected: `array:
- 1
- 2
- 3
mixed:
- item1
- item2
- nested:
    key: value
object:
  nested: value
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := strings.NewReader(tt.input)

			var output bytes.Buffer

			err := Normalize(input, &output)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Normalize failed: %v", err)
			}

			got := output.String()
			if got != tt.expected {
				t.Errorf("Normalize() = %q, want %q", got, tt.expected)
			}

			var obj any
			if err := yaml.Unmarshal(output.Bytes(), &obj); err != nil {
				t.Errorf("Normalized output is not valid YAML: %v", err)
			}
		})
	}
}

func TestNormalizeFile_NonExistentFile(t *testing.T) {
	t.Parallel()

	err := NormalizeFile("nonexistent.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file, but got none")
	}
}

func TestNormalizeFile_UnwritableFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.yaml")

	if err := os.WriteFile(filename, []byte("key: value\n"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	if err := os.Chmod(filename, 0444); err != nil {
		t.Fatalf("Failed to make file read-only: %v", err)
	}

	err := NormalizeFile(filename)
	if err == nil {
		t.Error("Expected error for unwritable file, but got none")
	}
}

func TestNormalize_ReaderError(t *testing.T) {
	t.Parallel()

	badReader := &badReader{}
	var output bytes.Buffer

	err := Normalize(badReader, &output)
	if err == nil {
		t.Error("Expected error for bad reader, but got none")
	}
}

func TestNormalize_WriterError(t *testing.T) {
	t.Parallel()

	input := strings.NewReader("key: value\n")
	badWriter := &badWriter{}

	err := Normalize(input, badWriter)
	if err == nil {
		t.Error("Expected error for bad writer, but got none")
	}
}

type badReader struct{}

func (r *badReader) Read(p []byte) (n int, err error) {
	return 0, &os.PathError{Op: "read", Path: "bad", Err: os.ErrInvalid}
}

type badWriter struct{}

func (w *badWriter) Write(p []byte) (n int, err error) {
	return 0, &os.PathError{Op: "write", Path: "bad", Err: os.ErrInvalid}
}

func TestNormalize_PreservesDataTypes(t *testing.T) {
	t.Parallel()

	input := `string: "hello"
integer: 42
float: 3.14
boolean: true
null_value: null
array: [1, 2, 3]
object:
  nested: value
`

	var output bytes.Buffer
	err := Normalize(strings.NewReader(input), &output)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	var result map[string]any
	if err := yaml.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse normalized output: %v", err)
	}

	if str, ok := result["string"].(string); !ok || str != "hello" {
		t.Errorf("String value not preserved: %v", result["string"])
	}

	if num, ok := result["integer"].(float64); !ok || num != 42 {
		t.Errorf("Integer value not preserved: %v", result["integer"])
	}

	if num, ok := result["float"].(float64); !ok || num != 3.14 {
		t.Errorf("Float value not preserved: %v", result["float"])
	}

	if b, ok := result["boolean"].(bool); !ok || !b {
		t.Errorf("Boolean value not preserved: %v", result["boolean"])
	}

	if result["null_value"] != nil {
		t.Errorf("Null value not preserved: %v", result["null_value"])
	}

	if arr, ok := result["array"].([]any); !ok || len(arr) != 3 {
		t.Errorf("Array not preserved: %v", result["array"])
	}

	if obj, ok := result["object"].(map[string]any); !ok || obj["nested"] != "value" {
		t.Errorf("Object not preserved: %v", result["object"])
	}
}

func TestNormalize_HandlesSpecialCharacters(t *testing.T) {
	t.Parallel()

	input := `special_chars: "line1\nline2\ttabbed"
quotes: "with \"double\" quotes"
apostrophe: "don't"
unicode: "café"
`

	var output bytes.Buffer
	err := Normalize(strings.NewReader(input), &output)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	var result map[string]any
	if err := yaml.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse normalized output: %v", err)
	}

	if _, ok := result["special_chars"].(string); !ok {
		t.Errorf("Special characters not preserved: %v", result["special_chars"])
	}

	if _, ok := result["quotes"].(string); !ok {
		t.Errorf("Quotes not preserved: %v", result["quotes"])
	}

	if _, ok := result["apostrophe"].(string); !ok {
		t.Errorf("Apostrophe not preserved: %v", result["apostrophe"])
	}

	if _, ok := result["unicode"].(string); !ok {
		t.Errorf("Unicode not preserved: %v", result["unicode"])
	}
}

func TestNormalizeFileAndNormalizeEquivalence(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input string
	}{
		{
			name: "simple key-value",
			input: `key: value
`,
		},
		{
			name: "nested structure",
			input: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  labels:
    app: test
spec:
  containers:
  - name: nginx
    image: nginx:latest
    ports:
    - containerPort: 80
`,
		},
		{
			name: "array with mixed types",
			input: `items:
- string
- 42
- true
- null
- nested:
    key: value
`,
		},
		{
			name: "quoted strings",
			input: `name: "quoted string"
description: 'single quoted'
special: "with \"quotes\" inside"
`,
		},
		{
			name: "multiline strings",
			input: `description: |
  This is a
  multiline string
  with line breaks
`,
		},
		{
			name: "complex nested structure",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
        env:
        - name: NGINX_HOST
          value: "localhost"
        - name: NGINX_PORT
          value: "80"
        resources:
          limits:
            cpu: "500m"
            memory: "512Mi"
          requests:
            cpu: "250m"
            memory: "256Mi"
`,
		},
		{
			name: "numbers and booleans",
			input: `integer: 42
float: 3.14
boolean: true
null_value: null
`,
		},
		{
			name: "arrays and objects",
			input: `array: [1, 2, 3]
object:
  nested: value
mixed:
  - item1
  - item2
  - nested:
      key: value
`,
		},
		{
			name:  "empty input",
			input: ``,
		},
		{
			name:  "only whitespace",
			input: `   `,
		},
		{
			name: "special characters",
			input: `special_chars: "line1\nline2\ttabbed"
quotes: "with \"double\" quotes"
apostrophe: "don't"
unicode: "café"
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			filename := filepath.Join(tmpDir, "test.yaml")

			if err := os.WriteFile(filename, []byte(tc.input), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			err := NormalizeFile(filename)
			if err != nil {
				t.Fatalf("NormalizeFile failed: %v", err)
			}

			fileContent, err := os.ReadFile(filename)
			if err != nil {
				t.Fatalf("Failed to read normalized file: %v", err)
			}

			var bufferContent bytes.Buffer
			err = Normalize(strings.NewReader(tc.input), &bufferContent)
			if err != nil {
				t.Fatalf("Normalize failed: %v", err)
			}

			fileResult := string(fileContent)
			bufferResult := bufferContent.String()

			if fileResult != bufferResult {
				t.Errorf("NormalizeFile and Normalize produced different results:\n"+
					"NormalizeFile result:\n%s\n"+
					"Normalize result:\n%s",
					fileResult, bufferResult)
			}

			var fileObj, bufferObj any
			if err := yaml.Unmarshal(fileContent, &fileObj); err != nil {
				t.Errorf("NormalizeFile result is not valid YAML: %v", err)
			}
			if err := yaml.Unmarshal(bufferContent.Bytes(), &bufferObj); err != nil {
				t.Errorf("Normalize result is not valid YAML: %v", err)
			}

			if !reflect.DeepEqual(fileObj, bufferObj) {
				t.Errorf("Parsed objects are not equivalent:\n"+
					"NormalizeFile object: %+v\n"+
					"Normalize object: %+v",
					fileObj, bufferObj)
			}
		})
	}
}
