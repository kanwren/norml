# norml

A simple tool to normalize Kubernetes YAML files.

## Installation

```bash
go install github.com/kanwren/norml/cmd/norml@latest
```

## Usage

```bash
# Normalize a file and print to stdout
norml file.yaml

# Normalize multiple files and print to stdout
norml file1.yaml file2.yaml

# Normalize files in-place
norml -i file1.yaml file2.yaml

# Normalize from stdin to stdout
cat file.yaml | norml
```
