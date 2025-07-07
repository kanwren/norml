package normalizer

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
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
			name:     "empty file",
			input:    ``,
			expected: ``,
		},
		{
			name:     "only whitespace",
			input:    `   `,
			expected: ``,
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

			err := NormalizeFile(filename, true)

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
			name:     "empty input",
			input:    ``,
			expected: ``,
		},
		{
			name:     "only whitespace",
			input:    `   `,
			expected: ``,
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

			err := Normalize(input, &output, true)

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

	err := NormalizeFile("nonexistent.yaml", true)
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

	err := NormalizeFile(filename, true)
	if err == nil {
		t.Error("Expected error for unwritable file, but got none")
	}
}

func TestNormalize_ReaderError(t *testing.T) {
	t.Parallel()

	badReader := &badReader{}
	var output bytes.Buffer

	err := Normalize(badReader, &output, true)
	if err == nil {
		t.Error("Expected error for bad reader, but got none")
	}
}

func TestNormalize_WriterError(t *testing.T) {
	t.Parallel()

	input := strings.NewReader("key: value\n")
	badWriter := &badWriter{}

	err := Normalize(input, badWriter, true)
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
	err := Normalize(strings.NewReader(input), &output, true)
	if err != nil {
		t.Fatalf("Normalize failed: %v", err)
	}

	var result map[string]any
	if err := yaml.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse normalized output: %v", err)
	}

	if str, ok := result["string"].(string); !ok || str != "hello" {
		t.Errorf("String value not preserved: %v %v", reflect.TypeOf(result["string"]), result["string"])
	}

	if num, ok := result["integer"].(int); !ok || num != 42 {
		t.Errorf("Integer value not preserved: %v %v", reflect.TypeOf(result["integer"]), result["integer"])
	}

	if num, ok := result["float"].(float64); !ok || num != 3.14 {
		t.Errorf("Float value not preserved: %v %v", reflect.TypeOf(result["float"]), result["float"])
	}

	if b, ok := result["boolean"].(bool); !ok || !b {
		t.Errorf("Boolean value not preserved: %v %v", reflect.TypeOf(result["boolean"]), result["boolean"])
	}

	if result["null_value"] != nil {
		t.Errorf("Null value not preserved: %v %v", reflect.TypeOf(result["null_value"]), result["null_value"])
	}

	if arr, ok := result["array"].([]any); !ok || len(arr) != 3 {
		t.Errorf("Array not preserved: %v %v", reflect.TypeOf(result["array"]), result["array"])
	}

	if obj, ok := result["object"].(map[string]any); !ok || obj["nested"] != "value" {
		t.Errorf("Object not preserved: %v %v", reflect.TypeOf(result["object"]), result["object"])
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
	err := Normalize(strings.NewReader(input), &output, true)
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
		// Multi-document test cases
		{
			name: "multi-document with various types",
			input: `name: test
version: 1
---
items:
- apple
- banana
- cherry
---
config:
  debug: true
  timeout: 30
`,
		},
		{
			name: "multi-document with empty documents",
			input: `first: document
---
---
second: document
`,
		},
		{
			name: "multi-document with YAML aliases",
			input: `defaults: &defaults
  timeout: 30
  retries: 3
---
service1:
  <<: *defaults
  name: frontend
---
service2:
  <<: *defaults
  name: backend
`,
		},
		{
			name: "large multi-document file",
			input: func() string {
				var parts []string
				for i := range 10 {
					parts = append(parts, fmt.Sprintf("doc%d:\n  key%d: value%d\n  items:\n    - item1\n    - item2", i, i, i))
				}
				return strings.Join(parts, "\n---\n")
			}(),
		},
		{
			name: "multi-document with kubernetes manifests",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  config.yaml: |
    debug: true
    timeout: 30
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
      - name: app
        image: myapp:latest
---
apiVersion: v1
kind: Service
metadata:
  name: app-service
spec:
  selector:
    app: myapp
  ports:
  - port: 80
    targetPort: 8080
`,
		},
		{
			name: "multi-document with unicode content",
			input: `name: "café"
greeting: "こんにちは"
---
unicode_array:
  - "résumé"
  - "naïve"
  - "Москва"
---
mixed: "ASCII and 中文 and العربية"
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

			err := NormalizeFile(filename, true)
			if err != nil {
				t.Fatalf("NormalizeFile failed: %v", err)
			}

			fileContent, err := os.ReadFile(filename)
			if err != nil {
				t.Fatalf("Failed to read normalized file: %v", err)
			}

			var bufferContent bytes.Buffer
			err = Normalize(strings.NewReader(tc.input), &bufferContent, true)
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

func TestNormalize_MultipleDocuments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name: "basic multi-document stream",
			input: `key1: value1
---
key2: value2
`,
			expected: `key1: value1
---
key2: value2
`,
		},
		{
			name: "mixed document types",
			input: `name: test
age: 30
---
- item1
- item2
- item3
---
"simple scalar"
`,
			expected: `age: 30
name: test
---
- item1
- item2
- item3
---
simple scalar
`,
		},
		{
			name: "empty documents in stream",
			input: `key: value
---
---
another: document
`,
			expected: `key: value
---

---
another: document
`,
		},
		{
			name: "complex documents with nested structures",
			input: `apiVersion: v1
kind: Pod
metadata:
  name: pod1
  labels:
    app: test
spec:
  containers:
  - name: nginx
    image: nginx:latest
---
apiVersion: v1
kind: Service
metadata:
  name: service1
spec:
  selector:
    app: test
  ports:
  - port: 80
`,
			expected: `apiVersion: v1
kind: Pod
metadata:
  labels:
    app: test
  name: pod1
spec:
  containers:
    - image: nginx:latest
      name: nginx
---
apiVersion: v1
kind: Service
metadata:
  name: service1
spec:
  ports:
    - port: 80
  selector:
    app: test
`,
		},
		{
			name: "documents with YAML aliases",
			input: `defaults: &default
  timeout: 30
  retries: 3
---
service1:
  <<: *default
  name: frontend
---
service2:
  <<: *default
  name: backend
`,
			expected: `defaults: &default
  retries: 3
  timeout: 30
---
service1:
  !!merge <<: *default
  name: frontend
---
service2:
  !!merge <<: *default
  name: backend
`,
		},
		{
			name: "single document (regression test)",
			input: `key: value
nested:
  sub: key
`,
			expected: `key: value
nested:
  sub: key
`,
		},
		{
			name: "documents with different data types",
			input: `string: "hello"
number: 42
boolean: true
null_value: null
---
array:
  - 1
  - 2
  - 3
---
nested:
  deep:
    structure: value
`,
			expected: `boolean: true
null_value: null
number: 42
string: hello
---
array:
  - 1
  - 2
  - 3
---
nested:
  deep:
    structure: value
`,
		},
		{
			name: "malformed document in stream",
			input: `key: value
---
  invalid: indentation
    very: bad
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := strings.NewReader(tt.input)
			var output bytes.Buffer

			err := Normalize(input, &output, true)

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
		})
	}
}

func TestNormalizeFile_MultipleDocuments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name: "in-place multi-document normalization",
			input: `name: test
version: 1
---
items:
- apple
- banana
- cherry
---
config:
  debug: true
  timeout: 30
`,
			expected: `name: test
version: 1
---
items:
  - apple
  - banana
  - cherry
---
config:
  debug: true
  timeout: 30
`,
		},
		{
			name: "document order preservation",
			input: `doc1: first
---
doc2: second
---
doc3: third
---
doc4: fourth
---
doc5: fifth
`,
			expected: `doc1: first
---
doc2: second
---
doc3: third
---
doc4: fourth
---
doc5: fifth
`,
		},
		{
			name: "mixed valid and invalid documents",
			input: `valid: document
---
  invalid: indentation
    very: bad
`,
			expectError: true,
		},
		{
			name: "empty documents mixed with valid ones",
			input: `first: document
---
---
second: document
`,
			expected: `first: document
---

---
second: document
`,
		},
		{
			name: "complex kubernetes-style documents",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-service
spec:
  selector:
    app: nginx
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
`,
			expected: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-service
spec:
  ports:
    - port: 80
      protocol: TCP
      targetPort: 8080
  selector:
    app: nginx
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

			err := NormalizeFile(filename, true)

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

			// Verify each document in the stream is valid YAML
			parts := strings.Split(got, "---\n")
			for i, part := range parts {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				var obj any
				if err := yaml.Unmarshal([]byte(part), &obj); err != nil {
					t.Errorf("Document %d in normalized output is not valid YAML: %v", i, err)
				}
			}
		})
	}
}

func TestNormalize_CommandLineIntegration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		files       []string
		expected    string
		expectError bool
	}{
		{
			name: "multiple files to stdout",
			files: []string{
				`key1: value1
nested:
  sub: key1
`,
				`key2: value2
nested:
  sub: key2
`,
				`key3: value3
nested:
  sub: key3
`,
			},
			expected: `key1: value1
nested:
  sub: key1
---
key2: value2
nested:
  sub: key2
---
key3: value3
nested:
  sub: key3
`,
		},
		{
			name: "single file with multiple documents",
			files: []string{
				`doc1: first
---
doc2: second
---
doc3: third
`,
			},
			expected: `doc1: first
---
doc2: second
---
doc3: third
`,
		},
		{
			name: "files with empty content",
			files: []string{
				`key: value
`,
				``,
				`another: key
`,
			},
			expected: `key: value
---
---
another: key
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			filenames := make([]string, len(tt.files))

			// Create test files
			for i, content := range tt.files {
				filename := filepath.Join(tmpDir, fmt.Sprintf("test%d.yaml", i))
				if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to write test file %d: %v", i, err)
				}
				filenames[i] = filename
			}

			// Simulate command-line behavior by normalizing each file and combining output
			var output bytes.Buffer
			for i, filename := range filenames {
				file, err := os.Open(filename)
				if err != nil {
					t.Fatalf("Failed to open file %s: %v", filename, err)
				}

				var buf bytes.Buffer
				err = Normalize(file, &buf, true)
				file.Close()

				if tt.expectError {
					if err == nil {
						t.Errorf("Expected error but got none")
					}
					return
				}

				if err != nil {
					t.Fatalf("Normalize failed for file %s: %v", filename, err)
				}

				// Add document separator before non-first files
				if i > 0 {
					output.WriteString("---\n")
				}
				output.Write(buf.Bytes())
			}

			got := output.String()
			if got != tt.expected {
				t.Errorf("Command-line integration test = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNormalize_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name: "large document stream",
			input: func() string {
				var parts []string
				for i := range 100 {
					parts = append(parts, fmt.Sprintf("doc%d: value%d", i, i))
				}
				return strings.Join(parts, "\n---\n")
			}(),
			expected: func() string {
				var parts []string
				for i := range 100 {
					parts = append(parts, fmt.Sprintf("doc%d: value%d", i, i))
				}
				return strings.Join(parts, "\n---\n") + "\n"
			}(),
		},
		{
			name: "very large single document in stream",
			input: func() string {
				var largeDoc strings.Builder
				largeDoc.WriteString("large_doc:\n")
				for i := range 1000 {
					largeDoc.WriteString(fmt.Sprintf("  key%d: value%d\n", i, i))
				}
				largeDoc.WriteString("---\nsmall: doc")
				return largeDoc.String()
			}(),
			expected: func() string {
				var largeDoc strings.Builder
				largeDoc.WriteString("large_doc:\n")
				for i := range 1000 {
					largeDoc.WriteString(fmt.Sprintf("  key%d: value%d\n", i, i))
				}
				largeDoc.WriteString("---\nsmall: doc\n")
				return largeDoc.String()
			}(),
		},
		{
			name: "documents with unicode content",
			input: `name: "café"
greeting: "こんにちは"
emoji: "🚀"
---
unicode_array:
  - "résumé"
  - "naïve"
  - "Москва"
---
mixed: "ASCII and 中文 and العربية"
`,
			expected: `emoji: "\U0001F680"
greeting: こんにちは
name: café
---
unicode_array:
  - résumé
  - naïve
  - Москва
---
mixed: ASCII and 中文 and العربية
`,
		},
		{
			name: "documents with complex nested aliases",
			input: `defaults: &defaults
  timeout: 30
  retries: 3
  config: &config
    debug: true
    level: info
---
service: &service
  name: test-service
  settings:
    <<: *defaults
    custom: value
---
deployment:
  template:
    spec:
      <<: *service
      replicas: 3
  config:
    <<: *config
`,
			expected: `defaults: &defaults
  config: &config
    debug: true
    level: info
  retries: 3
  timeout: 30
---
service: &service
  name: test-service
  settings:
    !!merge <<: *defaults
    custom: value
---
deployment:
  config:
    !!merge <<: *config
  template:
    spec:
      !!merge <<: *service
      replicas: 3
`,
		},
		{
			name: "documents with multiline strings",
			input: `description: |
  This is a long
  multiline description
  that should be preserved
---
script: >
  echo "This is a folded
  string that should
  be on one line"
---
literal: |2
    This has custom
    indentation that
    should be preserved
`,
			expected: `description: |
  This is a long
  multiline description
  that should be preserved
---
script: |
  echo "This is a folded string that should be on one line"
---
literal: |2
    This has custom
    indentation that
    should be preserved
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := strings.NewReader(tt.input)
			var output bytes.Buffer

			err := Normalize(input, &output, true)

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
				t.Errorf("Edge case test = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNormalize_ErrorHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name: "malformed document in stream",
			input: `valid: document
---
  invalid: indentation
    very: bad
`,
			expectError: true,
		},
		{
			name: "partial malformed stream",
			input: `first: valid
---
second: valid
---
  third: invalid
    very: bad
`,
			expectError: true,
		},
		{
			name: "completely malformed YAML",
			input: `key: value
---
[invalid, yaml: structure
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := strings.NewReader(tt.input)
			var output bytes.Buffer

			err := Normalize(input, &output, true)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Normalize failed: %v", err)
			}
		})
	}
}

func TestNormalize_StreamWriterFailure(t *testing.T) {
	t.Parallel()

	input := strings.NewReader(`doc1: value1
---
doc2: value2
---
doc3: value3
`)

	// Create a writer that fails after the first document
	failingWriter := &failingWriter{failAfter: 20}

	err := Normalize(input, failingWriter, true)
	if err == nil {
		t.Error("Expected error for failing writer, but got none")
	}
}

type failingWriter struct {
	written   int
	failAfter int
}

func (w *failingWriter) Write(p []byte) (n int, err error) {
	if w.written+len(p) > w.failAfter {
		return 0, &os.PathError{Op: "write", Path: "failing", Err: os.ErrInvalid}
	}
	w.written += len(p)
	return len(p), nil
}
