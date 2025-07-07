package normalizer

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"go.yaml.in/yaml/v3"
)

func normalizeNode(node *yaml.Node, preserveComments bool) {
	// Reset style
	node.Style = 0

	// Strip comments
	if !preserveComments {
		node.HeadComment = ""
		node.LineComment = ""
		node.FootComment = ""
	}

	// Normalize children
	for _, node := range node.Content {
		normalizeNode(node, preserveComments)
	}

	if node.Kind == yaml.MappingNode {
		node.Content = sortMapKeys(node.Content)
	}
}

func Normalize(r io.Reader, w io.Writer, preserveComments bool) error {
	dec := yaml.NewDecoder(r)
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)

	for {
		var node yaml.Node

		err := dec.Decode(&node)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to decode YAML input: %w", err)
		}

		normalizeNode(&node, preserveComments)

		err = enc.Encode(&node)
		if err != nil {
			return fmt.Errorf("failed to encode normalized YAML: %w", err)
		}
	}

	return nil
}

func NormalizeFile(filename string, preserveComments bool) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	outFile, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file for writing: %w", err)
	}
	defer outFile.Close()

	return Normalize(bytes.NewReader(data), outFile, preserveComments)
}
