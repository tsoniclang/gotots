package certify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/tsoniclang/gotots/internal/contracts/externals"
)

const seedSchemaVersion = 1

type seedDocument struct {
	SchemaVersion int           `json:"schemaVersion"`
	Bindings      []bindingSeed `json:"bindings"`
}

type bindingSeed struct {
	SourcePackage   string               `json:"sourcePackage"`
	SourceName      string               `json:"sourceName"`
	TargetKind      externals.TargetKind `json:"targetKind"`
	TargetName      string               `json:"targetName,omitempty"`
	ModuleSpecifier string               `json:"moduleSpecifier,omitempty"`
	SourcePath      string               `json:"sourcePath,omitempty"`
	Export          string               `json:"export,omitempty"`
}

func readSeeds(sourcePath string) ([]bindingSeed, error) {
	payload, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, certifyError("read binding map", sourcePath, err.Error())
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var document seedDocument
	if err := decoder.Decode(&document); err != nil {
		return nil, certifyError("read binding map", sourcePath, err.Error())
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return nil, certifyError("read binding map", sourcePath, err.Error())
	}
	if document.SchemaVersion != seedSchemaVersion || len(document.Bindings) == 0 {
		return nil, certifyError(
			"read binding map",
			sourcePath,
			"schema or binding set is invalid",
		)
	}
	previous := ""
	for index, seed := range document.Bindings {
		key := seed.SourcePackage + "\x00" + seed.SourceName
		if !canonicalImportPath(seed.SourcePackage) ||
			!exportOrPrivateName(seed.SourceName) ||
			!seed.TargetKind.Valid() {
			return nil, seedError(sourcePath, index, "source or target identity is invalid")
		}
		if previous != "" && key <= previous {
			return nil, seedError(sourcePath, index, "bindings are not strictly ordered")
		}
		previous = key
		switch seed.TargetKind {
		case externals.TargetModule:
			if seed.TargetName != "" ||
				!providerSpecifier(seed.ModuleSpecifier) ||
				!providerSourcePath(seed.SourcePath) ||
				!exportOrPrivateName(seed.Export) {
				return nil, seedError(sourcePath, index, "module target is invalid")
			}
		case externals.TargetSource:
			if !exportOrPrivateName(seed.TargetName) ||
				seed.ModuleSpecifier != "" || seed.SourcePath != "" ||
				seed.Export != "" {
				return nil, seedError(sourcePath, index, "source target is invalid")
			}
		}
	}
	return append([]bindingSeed(nil), document.Bindings...), nil
}

func seedError(sourcePath string, index int, reason string) error {
	return certifyError(
		"read binding map",
		fmt.Sprintf("%s#bindings[%d]", sourcePath, index),
		reason,
	)
}

func canonicalImportPath(value string) bool {
	return value != "" && value != "." && !path.IsAbs(value) &&
		path.Clean(value) == value && !strings.HasPrefix(value, "../")
}

func providerSpecifier(value string) bool {
	return strings.HasPrefix(value, externals.PackageName+"/") &&
		strings.HasSuffix(value, ".js") && !strings.Contains(value, "..")
}

func providerSourcePath(value string) bool {
	return canonicalImportPath(value) && strings.HasPrefix(value, "src/") &&
		strings.HasSuffix(value, ".ts")
}

func exportOrPrivateName(value string) bool {
	if value == "" {
		return false
	}
	for index, character := range value {
		if character == '_' || character == '$' ||
			character >= 'a' && character <= 'z' ||
			character >= 'A' && character <= 'Z' ||
			index > 0 && character >= '0' && character <= '9' {
			continue
		}
		return false
	}
	return true
}
