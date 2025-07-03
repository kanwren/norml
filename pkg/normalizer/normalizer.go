package normalizer

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"sigs.k8s.io/yaml"
)

func Normalize(r io.Reader, w io.Writer) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	var obj any
	if err := yaml.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	normalizedData, err := yaml.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal normalized YAML: %w", err)
	}

	if _, err := io.Copy(w, bytes.NewReader(normalizedData)); err != nil {
		return fmt.Errorf("failed to write normalized YAML: %w", err)
	}

	return nil
}

func NormalizeFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var obj any
	if err := yaml.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	normalizedData, err := yaml.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal normalized YAML: %w", err)
	}

	if err := os.WriteFile(filename, normalizedData, 0644); err != nil {
		return fmt.Errorf("failed to write normalized YAML: %w", err)
	}

	return nil
}
