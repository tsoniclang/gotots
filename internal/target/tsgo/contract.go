package tsgo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

const contractSchemaVersion = 1

type Contract struct {
	repository      string
	module          string
	toolPackage     string
	toolVersion     string
	toolSum         string
	revision        string
	protocolVersion int
	files           []ContractFile
}

type ContractFile struct {
	sourcePath string
	path       string
	sha256     string
}

type ContractError struct {
	Path   string
	Reason string
}

func (e *ContractError) Error() string {
	if e.Path == "" {
		return "verify pinned TS-Go contract: " + e.Reason
	}
	return fmt.Sprintf("verify pinned TS-Go contract %s: %s", e.Path, e.Reason)
}

type contractManifest struct {
	SchemaVersion   int                    `json:"schemaVersion"`
	Repository      string                 `json:"repository"`
	Module          string                 `json:"module"`
	ToolPackage     string                 `json:"toolPackage"`
	ToolVersion     string                 `json:"toolVersion"`
	ToolSum         string                 `json:"toolSum"`
	Revision        string                 `json:"revision"`
	ProtocolVersion int                    `json:"protocolVersion"`
	Files           []contractManifestFile `json:"files"`
}

type contractManifestFile struct {
	SourcePath string `json:"sourcePath"`
	Path       string `json:"path"`
	SHA256     string `json:"sha256"`
}

func VerifyPinnedContract(schemaDirectory string) (Contract, error) {
	manifestPath := filepath.Join(schemaDirectory, "manifest.json")
	manifestValue, err := readContractManifest(manifestPath)
	if err != nil {
		return Contract{}, err
	}
	if err := validateManifest(manifestValue); err != nil {
		return Contract{}, err
	}
	if err := verifyContractFiles(schemaDirectory, manifestValue.Files); err != nil {
		return Contract{}, err
	}
	files := make([]ContractFile, len(manifestValue.Files))
	for index, file := range manifestValue.Files {
		files[index] = ContractFile{
			sourcePath: file.SourcePath,
			path:       file.Path,
			sha256:     file.SHA256,
		}
	}
	return Contract{
		repository:      manifestValue.Repository,
		module:          manifestValue.Module,
		toolPackage:     manifestValue.ToolPackage,
		toolVersion:     manifestValue.ToolVersion,
		toolSum:         manifestValue.ToolSum,
		revision:        manifestValue.Revision,
		protocolVersion: manifestValue.ProtocolVersion,
		files:           files,
	}, nil
}

func (c Contract) Repository() string {
	return c.repository
}

func (c Contract) Module() string {
	return c.module
}

func (c Contract) ToolPackage() string {
	return c.toolPackage
}

func (c Contract) ToolVersion() string {
	return c.toolVersion
}

func (c Contract) ToolSum() string {
	return c.toolSum
}

func (c Contract) Revision() string {
	return c.revision
}

func (c Contract) ProtocolVersion() int {
	return c.protocolVersion
}

func (c Contract) Files() []ContractFile {
	return slices.Clone(c.files)
}

func (f ContractFile) SourcePath() string {
	return f.sourcePath
}

func (f ContractFile) Path() string {
	return f.path
}

func (f ContractFile) SHA256() string {
	return f.sha256
}

func readContractManifest(manifestPath string) (contractManifest, error) {
	file, err := os.Open(manifestPath)
	if err != nil {
		return contractManifest{}, &ContractError{Path: manifestPath, Reason: err.Error()}
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var manifestValue contractManifest
	if err := decoder.Decode(&manifestValue); err != nil {
		return contractManifest{}, &ContractError{Path: manifestPath, Reason: err.Error()}
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		return contractManifest{}, &ContractError{Path: manifestPath, Reason: err.Error()}
	}
	return manifestValue, nil
}

func validateManifest(manifestValue contractManifest) error {
	if manifestValue.SchemaVersion != contractSchemaVersion {
		return &ContractError{Reason: fmt.Sprintf(
			"manifest schema version %d, want %d",
			manifestValue.SchemaVersion,
			contractSchemaVersion,
		)}
	}
	if manifestValue.Repository == "" ||
		manifestValue.Module == "" ||
		manifestValue.ToolPackage == "" ||
		manifestValue.ToolVersion == "" || manifestValue.ToolSum == "" {
		return &ContractError{Reason: "repository, module, toolPackage, toolVersion, and toolSum are required"}
	}
	if !strings.HasPrefix(manifestValue.ToolPackage, manifestValue.Module+"/") {
		return &ContractError{Reason: "toolPackage is outside module"}
	}
	if !validHex(manifestValue.Revision, 20) {
		return &ContractError{Reason: "revision must be a lowercase 40-digit hexadecimal commit"}
	}
	if !strings.HasSuffix(manifestValue.ToolVersion, "-"+manifestValue.Revision[:12]) {
		return &ContractError{Reason: "toolVersion does not identify revision"}
	}
	if !strings.HasPrefix(manifestValue.ToolSum, "h1:") {
		return &ContractError{Reason: "toolSum is invalid"}
	}
	if manifestValue.ProtocolVersion <= 0 {
		return &ContractError{Reason: "protocolVersion must be positive"}
	}
	if len(manifestValue.Files) == 0 {
		return &ContractError{Reason: "files must not be empty"}
	}

	seenSource := make(map[string]struct{}, len(manifestValue.Files))
	seenPath := make(map[string]struct{}, len(manifestValue.Files))
	previousPath := ""
	for _, file := range manifestValue.Files {
		if !validRelativeSlashPath(file.SourcePath) {
			return &ContractError{Path: file.SourcePath, Reason: "invalid sourcePath"}
		}
		if !validRelativeSlashPath(file.Path) || !strings.HasPrefix(file.Path, "upstream/") {
			return &ContractError{Path: file.Path, Reason: "path must be under upstream/"}
		}
		if !validHex(file.SHA256, sha256.Size) {
			return &ContractError{Path: file.Path, Reason: "invalid lowercase SHA-256"}
		}
		if _, exists := seenSource[file.SourcePath]; exists {
			return &ContractError{Path: file.SourcePath, Reason: "duplicate sourcePath"}
		}
		if _, exists := seenPath[file.Path]; exists {
			return &ContractError{Path: file.Path, Reason: "duplicate path"}
		}
		if previousPath != "" && file.Path <= previousPath {
			return &ContractError{Path: file.Path, Reason: "files are not sorted by path"}
		}
		seenSource[file.SourcePath] = struct{}{}
		seenPath[file.Path] = struct{}{}
		previousPath = file.Path
	}
	return nil
}

func verifyContractFiles(schemaDirectory string, files []contractManifestFile) error {
	expected := make(map[string]contractManifestFile, len(files))
	for _, file := range files {
		expected[file.Path] = file
		fullPath := filepath.Join(schemaDirectory, filepath.FromSlash(file.Path))
		info, err := os.Lstat(fullPath)
		if err != nil {
			return &ContractError{Path: file.Path, Reason: err.Error()}
		}
		if !info.Mode().IsRegular() {
			return &ContractError{Path: file.Path, Reason: "not a regular file"}
		}
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return &ContractError{Path: file.Path, Reason: err.Error()}
		}
		digest := sha256.Sum256(data)
		actual := hex.EncodeToString(digest[:])
		if actual != file.SHA256 {
			return &ContractError{Path: file.Path, Reason: fmt.Sprintf(
				"SHA-256 %s, want %s",
				actual,
				file.SHA256,
			)}
		}
	}

	upstreamDirectory := filepath.Join(schemaDirectory, "upstream")
	return filepath.WalkDir(upstreamDirectory, func(fullPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return &ContractError{Path: fullPath, Reason: walkErr.Error()}
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(schemaDirectory, fullPath)
		if err != nil {
			return &ContractError{Path: fullPath, Reason: err.Error()}
		}
		slashPath := filepath.ToSlash(relative)
		if _, exists := expected[slashPath]; !exists {
			return &ContractError{Path: slashPath, Reason: "unlisted contract input"}
		}
		return nil
	})
}

func validRelativeSlashPath(value string) bool {
	return value != "" &&
		!strings.Contains(value, `\`) &&
		!strings.HasPrefix(value, "/") &&
		path.Clean(value) == value &&
		value != "." &&
		!strings.HasPrefix(value, "../")
}

func validHex(value string, byteLength int) bool {
	if len(value) != byteLength*2 || strings.ToLower(value) != value {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == byteLength
}
