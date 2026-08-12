package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
	ToolSum         string            `json:"toolSum"`
	Revision        string            `json:"revision"`
	ProtocolVersion int               `json:"protocolVersion"`
	Files           []json.RawMessage `json:"files"`

	// contractDigest is the sha256 of the canonical manifest bytes; it
	// content-addresses the complete pinned schema contract.
	contractDigest string
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
	if manifest.Module == "" || manifest.ToolPackage == "" ||
		manifest.ToolVersion == "" || manifest.ToolSum == "" {
		return targetManifest{}, fmt.Errorf("manifest.json has incomplete tool identity")
	}
	if manifest.Revision == "" {
		return targetManifest{}, fmt.Errorf("manifest.json has no pinned revision")
	}
	digest := sha256.Sum256(data)
	manifest.contractDigest = hex.EncodeToString(digest[:])
	return manifest, nil
}
