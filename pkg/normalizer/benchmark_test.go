package normalizer

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	goccy "github.com/goccy/go-yaml"
	goyaml "go.yaml.in/yaml/v3"
	sigsyaml "sigs.k8s.io/yaml"
)

var smallYAML = `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  namespace: default
  labels:
    app: myapp
    version: "1.0"
data:
  key1: value1
  key2: value2
  config.json: |
    {
      "debug": true,
      "port": 8080
    }
`

var mediumYAML = generateMediumYAML()

func generateMediumYAML() string {
	var b strings.Builder
	b.WriteString("apiVersion: apps/v1\nkind: Deployment\nmetadata:\n")
	b.WriteString("  name: test-deployment\n  namespace: production\n")
	b.WriteString("  labels:\n")
	for i := range 20 {
		fmt.Fprintf(&b, "    label%d: value%d\n", i, i)
	}
	b.WriteString("  annotations:\n")
	for i := range 20 {
		fmt.Fprintf(&b, "    annotation%d: value%d\n", i, i)
	}
	b.WriteString("spec:\n  replicas: 3\n  selector:\n    matchLabels:\n      app: test\n")
	b.WriteString("  template:\n    metadata:\n      labels:\n        app: test\n")
	b.WriteString("    spec:\n      containers:\n")
	for i := range 5 {
		fmt.Fprintf(&b, "      - name: container%d\n", i)
		fmt.Fprintf(&b, "        image: nginx:1.%d\n", i)
		b.WriteString("        ports:\n")
		b.WriteString("        - containerPort: 80\n")
		b.WriteString("        env:\n")
		for j := range 10 {
			fmt.Fprintf(&b, "        - name: ENV_%d_%d\n", i, j)
			fmt.Fprintf(&b, "          value: value_%d_%d\n", i, j)
		}
		b.WriteString("        resources:\n")
		b.WriteString("          limits:\n")
		b.WriteString("            cpu: 100m\n")
		b.WriteString("            memory: 128Mi\n")
		b.WriteString("          requests:\n")
		b.WriteString("            cpu: 50m\n")
		b.WriteString("            memory: 64Mi\n")
	}
	return b.String()
}

var largeYAML = generateLargeYAML()

func generateLargeYAML() string {
	var b strings.Builder
	for doc := range 10 {
		if doc > 0 {
			b.WriteString("---\n")
		}
		fmt.Fprintf(&b, "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: config-%d\n", doc)
		b.WriteString("data:\n")
		for i := range 100 {
			fmt.Fprintf(&b, "  key%d: value%d\n", i, i)
		}
		b.WriteString("  nested:\n")
		for i := range 50 {
			fmt.Fprintf(&b, "    nested_key%d:\n", i)
			fmt.Fprintf(&b, "      sub_key1: sub_value1\n")
			fmt.Fprintf(&b, "      sub_key2: sub_value2\n")
			fmt.Fprintf(&b, "      sub_key3: sub_value3\n")
		}
	}
	return b.String()
}

// Deeply nested YAML to stress recursive operations
var deeplyNestedYAML = generateDeeplyNestedYAML(15)

func generateDeeplyNestedYAML(depth int) string {
	var b strings.Builder
	b.WriteString("root:\n")
	indent := "  "
	for i := range depth {
		fmt.Fprintf(&b, "%slevel%d:\n", indent, i)
		fmt.Fprintf(&b, "%s  data: value%d\n", indent, i)
		fmt.Fprintf(&b, "%s  sibling1: value1\n", indent)
		fmt.Fprintf(&b, "%s  sibling2: value2\n", indent)
		fmt.Fprintf(&b, "%s  nested:\n", indent)
		indent += "    "
	}
	fmt.Fprintf(&b, "%sleaf: final_value\n", indent)
	return b.String()
}

// Wide YAML with many keys at same level (stress sorting)
var wideYAML = generateWideYAML(500)

func generateWideYAML(numKeys int) string {
	var b strings.Builder
	b.WriteString("data:\n")
	for i := range numKeys {
		// Use random-ish key names to ensure sorting does work
		fmt.Fprintf(&b, "  zkey%d: value%d\n", numKeys-i, i)
		fmt.Fprintf(&b, "  akey%d: value%d\n", i, i)
	}
	return b.String()
}

// Benchmarks for norml (this library)

func BenchmarkNorml_Small(b *testing.B) {
	input := []byte(smallYAML)
	for b.Loop() {
		var buf bytes.Buffer
		if err := Normalize(bytes.NewReader(input), &buf, false); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNorml_Medium(b *testing.B) {
	input := []byte(mediumYAML)
	for b.Loop() {
		var buf bytes.Buffer
		if err := Normalize(bytes.NewReader(input), &buf, false); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNorml_Large(b *testing.B) {
	input := []byte(largeYAML)
	for b.Loop() {
		var buf bytes.Buffer
		if err := Normalize(bytes.NewReader(input), &buf, false); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNorml_DeeplyNested(b *testing.B) {
	input := []byte(deeplyNestedYAML)
	for b.Loop() {
		var buf bytes.Buffer
		if err := Normalize(bytes.NewReader(input), &buf, false); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNorml_Wide(b *testing.B) {
	input := []byte(wideYAML)
	for b.Loop() {
		var buf bytes.Buffer
		if err := Normalize(bytes.NewReader(input), &buf, false); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmarks for go.yaml.in/yaml/v3 (round-trip, no normalization)

func BenchmarkGoYaml_RoundTrip_Small(b *testing.B) {
	input := []byte(smallYAML)
	for b.Loop() {
		var node goyaml.Node
		if err := goyaml.Unmarshal(input, &node); err != nil {
			b.Fatal(err)
		}
		if _, err := goyaml.Marshal(&node); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGoYaml_RoundTrip_Medium(b *testing.B) {
	input := []byte(mediumYAML)
	for b.Loop() {
		var node goyaml.Node
		if err := goyaml.Unmarshal(input, &node); err != nil {
			b.Fatal(err)
		}
		if _, err := goyaml.Marshal(&node); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGoYaml_RoundTrip_Large(b *testing.B) {
	input := []byte(largeYAML)
	for b.Loop() {
		var node goyaml.Node
		if err := goyaml.Unmarshal(input, &node); err != nil {
			b.Fatal(err)
		}
		if _, err := goyaml.Marshal(&node); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGoYaml_RoundTrip_DeeplyNested(b *testing.B) {
	input := []byte(deeplyNestedYAML)
	for b.Loop() {
		var node goyaml.Node
		if err := goyaml.Unmarshal(input, &node); err != nil {
			b.Fatal(err)
		}
		if _, err := goyaml.Marshal(&node); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGoYaml_RoundTrip_Wide(b *testing.B) {
	input := []byte(wideYAML)
	for b.Loop() {
		var node goyaml.Node
		if err := goyaml.Unmarshal(input, &node); err != nil {
			b.Fatal(err)
		}
		if _, err := goyaml.Marshal(&node); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmarks for github.com/goccy/go-yaml (round-trip, no normalization)

func BenchmarkGoccy_RoundTrip_Small(b *testing.B) {
	input := []byte(smallYAML)
	for b.Loop() {
		var data any
		if err := goccy.Unmarshal(input, &data); err != nil {
			b.Fatal(err)
		}
		if _, err := goccy.Marshal(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGoccy_RoundTrip_Medium(b *testing.B) {
	input := []byte(mediumYAML)
	for b.Loop() {
		var data any
		if err := goccy.Unmarshal(input, &data); err != nil {
			b.Fatal(err)
		}
		if _, err := goccy.Marshal(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGoccy_RoundTrip_Large(b *testing.B) {
	input := []byte(largeYAML)
	for b.Loop() {
		var data any
		if err := goccy.Unmarshal(input, &data); err != nil {
			b.Fatal(err)
		}
		if _, err := goccy.Marshal(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGoccy_RoundTrip_DeeplyNested(b *testing.B) {
	input := []byte(deeplyNestedYAML)
	for b.Loop() {
		var data any
		if err := goccy.Unmarshal(input, &data); err != nil {
			b.Fatal(err)
		}
		if _, err := goccy.Marshal(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGoccy_RoundTrip_Wide(b *testing.B) {
	input := []byte(wideYAML)
	for b.Loop() {
		var data any
		if err := goccy.Unmarshal(input, &data); err != nil {
			b.Fatal(err)
		}
		if _, err := goccy.Marshal(data); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmarks for sigs.k8s.io/yaml (round-trip via JSON intermediate)

func BenchmarkSigsYaml_RoundTrip_Small(b *testing.B) {
	input := []byte(smallYAML)
	for b.Loop() {
		var data any
		if err := sigsyaml.Unmarshal(input, &data); err != nil {
			b.Fatal(err)
		}
		if _, err := sigsyaml.Marshal(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSigsYaml_RoundTrip_Medium(b *testing.B) {
	input := []byte(mediumYAML)
	for b.Loop() {
		var data any
		if err := sigsyaml.Unmarshal(input, &data); err != nil {
			b.Fatal(err)
		}
		if _, err := sigsyaml.Marshal(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSigsYaml_RoundTrip_Large(b *testing.B) {
	input := []byte(largeYAML)
	for b.Loop() {
		var data any
		if err := sigsyaml.Unmarshal(input, &data); err != nil {
			b.Fatal(err)
		}
		if _, err := sigsyaml.Marshal(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSigsYaml_RoundTrip_DeeplyNested(b *testing.B) {
	input := []byte(deeplyNestedYAML)
	for b.Loop() {
		var data any
		if err := sigsyaml.Unmarshal(input, &data); err != nil {
			b.Fatal(err)
		}
		if _, err := sigsyaml.Marshal(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSigsYaml_RoundTrip_Wide(b *testing.B) {
	input := []byte(wideYAML)
	for b.Loop() {
		var data any
		if err := sigsyaml.Unmarshal(input, &data); err != nil {
			b.Fatal(err)
		}
		if _, err := sigsyaml.Marshal(data); err != nil {
			b.Fatal(err)
		}
	}
}

// Parse-only benchmarks (decode without encode)

func BenchmarkGoYaml_ParseOnly_Large(b *testing.B) {
	input := []byte(largeYAML)
	for b.Loop() {
		var node goyaml.Node
		if err := goyaml.Unmarshal(input, &node); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGoccy_ParseOnly_Large(b *testing.B) {
	input := []byte(largeYAML)
	for b.Loop() {
		var data any
		if err := goccy.Unmarshal(input, &data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSigsYaml_ParseOnly_Large(b *testing.B) {
	input := []byte(largeYAML)
	for b.Loop() {
		var data any
		if err := sigsyaml.Unmarshal(input, &data); err != nil {
			b.Fatal(err)
		}
	}
}

// Encode-only benchmarks (encode pre-parsed data)

func BenchmarkGoYaml_EncodeOnly_Large(b *testing.B) {
	input := []byte(largeYAML)
	var node goyaml.Node
	if err := goyaml.Unmarshal(input, &node); err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		if _, err := goyaml.Marshal(&node); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGoccy_EncodeOnly_Large(b *testing.B) {
	input := []byte(largeYAML)
	var data any
	if err := goccy.Unmarshal(input, &data); err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		if _, err := goccy.Marshal(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSigsYaml_EncodeOnly_Large(b *testing.B) {
	input := []byte(largeYAML)
	var data any
	if err := sigsyaml.Unmarshal(input, &data); err != nil {
		b.Fatal(err)
	}
	for b.Loop() {
		if _, err := sigsyaml.Marshal(data); err != nil {
			b.Fatal(err)
		}
	}
}

// Memory allocation benchmarks

func BenchmarkNorml_Memory_Large(b *testing.B) {
	input := []byte(largeYAML)
	b.ReportAllocs()
	for b.Loop() {
		var buf bytes.Buffer
		if err := Normalize(bytes.NewReader(input), &buf, false); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGoYaml_Memory_Large(b *testing.B) {
	input := []byte(largeYAML)
	b.ReportAllocs()
	for b.Loop() {
		var node goyaml.Node
		if err := goyaml.Unmarshal(input, &node); err != nil {
			b.Fatal(err)
		}
		if _, err := goyaml.Marshal(&node); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGoccy_Memory_Large(b *testing.B) {
	input := []byte(largeYAML)
	b.ReportAllocs()
	for b.Loop() {
		var data any
		if err := goccy.Unmarshal(input, &data); err != nil {
			b.Fatal(err)
		}
		if _, err := goccy.Marshal(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSigsYaml_Memory_Large(b *testing.B) {
	input := []byte(largeYAML)
	b.ReportAllocs()
	for b.Loop() {
		var data any
		if err := sigsyaml.Unmarshal(input, &data); err != nil {
			b.Fatal(err)
		}
		if _, err := sigsyaml.Marshal(data); err != nil {
			b.Fatal(err)
		}
	}
}

// Test that verifies our test data is valid
func TestBenchmarkDataValidity(t *testing.T) {
	testCases := []struct {
		name string
		data string
	}{
		{"small", smallYAML},
		{"medium", mediumYAML},
		{"large", largeYAML},
		{"deeply_nested", deeplyNestedYAML},
		{"wide", wideYAML},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var node goyaml.Node
			if err := goyaml.Unmarshal([]byte(tc.data), &node); err != nil {
				t.Errorf("invalid YAML: %v", err)
			}

			// Also verify norml can process it
			var buf bytes.Buffer
			if err := Normalize(bytes.NewReader([]byte(tc.data)), &buf, false); err != nil {
				t.Errorf("norml failed: %v", err)
			}
		})
	}
}

// Print sizes of test data
func TestPrintBenchmarkDataSizes(t *testing.T) {
	t.Logf("smallYAML: %d bytes", len(smallYAML))
	t.Logf("mediumYAML: %d bytes", len(mediumYAML))
	t.Logf("largeYAML: %d bytes", len(largeYAML))
	t.Logf("deeplyNestedYAML: %d bytes", len(deeplyNestedYAML))
	t.Logf("wideYAML: %d bytes", len(wideYAML))
}

// Already-sorted YAML to test skip-sorting optimization
var alreadySortedYAML = generateAlreadySortedYAML()

func generateAlreadySortedYAML() string {
	var b strings.Builder
	b.WriteString("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: sorted-config\ndata:\n")
	// Keys are already in alphabetical order
	for i := range 100 {
		fmt.Fprintf(&b, "  key%03d: value%d\n", i, i) // key000, key001, ..., key099
	}
	return b.String()
}

// Benchmark for already-sorted YAML (tests skip-sorting optimization)
func BenchmarkNorml_AlreadySorted(b *testing.B) {
	input := []byte(alreadySortedYAML)
	for b.Loop() {
		var buf bytes.Buffer
		if err := Normalize(bytes.NewReader(input), &buf, false); err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark for file I/O operations (tests buffer size optimization)
func BenchmarkNorml_FileIO_Large(b *testing.B) {
	// Create a temporary file with large YAML content
	tmpFile, err := os.CreateTemp("", "norml-bench-*.yaml")
	if err != nil {
		b.Fatal(err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	// Write test data
	if _, err := tmpFile.WriteString(largeYAML); err != nil {
		_ = tmpFile.Close()
		b.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		// Reset file content before each iteration
		b.StopTimer()
		if err := os.WriteFile(tmpPath, []byte(largeYAML), 0644); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		if err := NormalizeFile(tmpPath, false); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGoYaml_RoundTrip_AlreadySorted(b *testing.B) {
	input := []byte(alreadySortedYAML)
	for b.Loop() {
		var node goyaml.Node
		if err := goyaml.Unmarshal(input, &node); err != nil {
			b.Fatal(err)
		}
		if _, err := goyaml.Marshal(&node); err != nil {
			b.Fatal(err)
		}
	}
}
