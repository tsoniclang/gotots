package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type targetManifest struct {
	SchemaVersion   int               `json:"schemaVersion"`
	Repository      string            `json:"repository"`
	Module          string            `json:"module"`
	ToolPackage     string            `json:"toolPackage"`
	ToolVersion     string            `json:"toolVersion"`
	Revision        string            `json:"revision"`
	ProtocolVersion int               `json:"protocolVersion"`
	Files           []json.RawMessage `json:"files"`
}

func loadTargetManifest(directory string) (targetManifest, error) {
	data, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	if err != nil {
		return targetManifest{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var manifest targetManifest
	if err := decoder.Decode(&manifest); err != nil {
		return targetManifest{}, fmt.Errorf("decode manifest.json: %w", err)
	}
	if manifest.Module == "" || manifest.ToolPackage == "" || manifest.ToolVersion == "" {
		return targetManifest{}, fmt.Errorf("manifest.json has incomplete tool identity")
	}
	return manifest, nil
}
