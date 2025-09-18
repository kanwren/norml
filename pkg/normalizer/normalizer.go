package normalizer

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

func normalizeNode(node *yaml.Node, preserveComments bool) error {
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
		err := normalizeNode(node, preserveComments)
		if err != nil {
			return err
		}
	}

	if node.Kind == yaml.MappingNode {
		content, err := sortMapKeys(node.Content)
		if err != nil {
			return err
		}
		node.Content = content
	}

	return nil
}

func Normalize(r io.Reader, w io.Writer, preserveComments bool) error {
	dec := yaml.NewDecoder(r)
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)

	wrote := false
	for {
		var node yaml.Node

		err := dec.Decode(&node)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to decode YAML input: %w", err)
		}

		err = normalizeNode(&node, preserveComments)
		if err != nil {
			return fmt.Errorf("failed to normalize YAML node: %w", err)
		}

		err = enc.Encode(&node)
		if err != nil {
			return fmt.Errorf("failed to encode normalized YAML: %w", err)
		}

		wrote = true
	}

	var err error
	if wrote {
		err = enc.Close()
	}
	return err
}

func NormalizeFile(filename string, preserveComments bool) (finalErr error) {
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if fileInfo.Mode()&0200 == 0 {
		return fmt.Errorf("file to normalize is not writable: %s", filename)
	}

	// For small files (<1MiB), just read into memory; otherwise, stream to
	// temporary file and atomically rename
	const largeFileThreshold = 1 * 1024 * 1024
	if fileInfo.Size() <= largeFileThreshold {
		return normalizeFileSmall(filename, fileInfo.Mode(), preserveComments)
	}
	return normalizeFileLarge(filename, fileInfo.Mode(), preserveComments)
}

func normalizeFileLarge(filename string, mode os.FileMode, preserveComments bool) (finalErr error) {
	tmpFile := filepath.Join(filepath.Dir(filename), ".tmp_"+filepath.Base(filename))

	inFile, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	defer func() {
		if err := inFile.Close(); finalErr == nil && err != nil {
			finalErr = err
		}
	}()
	r := bufio.NewReader(inFile)

	err = normalizeToFile(r, tmpFile, mode, preserveComments)
	if err != nil {
		return err
	}

	err = os.Rename(tmpFile, filename)
	if err != nil {
		return fmt.Errorf("failed to replace original file: %w", err)
	}

	return nil
}

func normalizeFileSmall(filename string, mode os.FileMode, preserveComments bool) (finalErr error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	return normalizeToFile(bytes.NewReader(data), filename, mode, preserveComments)
}

func normalizeToFile(r io.Reader, filename string, mode os.FileMode, preserveComments bool) (finalErr error) {
	outFile, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("failed to open file for writing: %w", err)
	}
	defer func() {
		if err := outFile.Close(); finalErr == nil && err != nil {
			finalErr = err
		}
	}()

	w := bufio.NewWriter(outFile)
	defer func() {
		if err := w.Flush(); finalErr == nil && err != nil {
			finalErr = err
		}
	}()

	return Normalize(r, w, preserveComments)
}
