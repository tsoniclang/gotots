package main

import (
	"bytes"
	"fmt"
	"go/format"
	"path/filepath"
)

func renderAll(model *schemaModel) (map[string][]byte, error) {
	outputs := make(map[string][]byte)
	renderers := []struct {
		name   string
		render func(*schemaModel) ([]byte, error)
	}{
		{"kinds_generated.go", renderKinds},
		{"protocol_generated.go", renderProtocol},
		{"nodes_generated.go", renderNodes},
		{"factories_generated.go", renderFactories},
		{"encoding_generated.go", renderEncoding},
		{"contract_generated_test.go", renderContractEvidence},
	}
	for _, item := range renderers {
		data, err := item.render(model)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", item.name, err)
		}
		formatted, err := format.Source(data)
		if err != nil {
			return nil, fmt.Errorf("%s: format: %w\n%s", item.name, err, data)
		}
		outputs[item.name] = formatted
	}
	return outputs, nil
}

func generatedHeader(buffer *bytes.Buffer) {
	buffer.WriteString("// Code generated from schema/tsgo by go generate. DO NOT EDIT.\n\n")
	buffer.WriteString("package tsgo\n\n")
}

func schemaPath(model *schemaModel, name string) string {
	return filepath.Join(model.directory, "upstream", name)
}
