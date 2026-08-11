package load

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"golang.org/x/tools/go/packages"
)

type driverEvidence struct {
	Roots    []string                `json:"roots"`
	Packages []driverPackageEvidence `json:"packages"`
}

type driverPackageEvidence struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	PackagePath     string            `json:"packagePath"`
	Directory       string            `json:"directory"`
	ForTest         string            `json:"forTest"`
	Module          *packages.Module  `json:"module"`
	GoFiles         []string          `json:"goFiles"`
	CompiledGoFiles []string          `json:"compiledGoFiles"`
	OtherFiles      []string          `json:"otherFiles"`
	EmbedFiles      []string          `json:"embedFiles"`
	EmbedPatterns   []string          `json:"embedPatterns"`
	IgnoredFiles    []string          `json:"ignoredFiles"`
	Imports         map[string]string `json:"imports"`
}

func reserveDriverEvidence(cacheRoot string) (string, error) {
	root := filepath.Join(cacheRoot, "package-driver")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("load exact Go packages: create evidence root: %w", err)
	}
	file, err := os.CreateTemp(root, "evidence-*.json")
	if err != nil {
		return "", fmt.Errorf("load exact Go packages: reserve evidence: %w", err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("load exact Go packages: reserve evidence: %w", err)
	}
	return path, nil
}

func writeDriverEvidence(
	path string,
	roots []string,
	selected []*packages.Package,
) error {
	if path == "" {
		return fmt.Errorf("exact Go packages driver: evidence path is absent")
	}
	records := make([]driverPackageEvidence, len(selected))
	for index, current := range selected {
		imports := make(map[string]string, len(current.Imports))
		for importPath, imported := range current.Imports {
			if imported == nil || imported.ID == "" {
				return fmt.Errorf(
					"exact Go packages driver: package %q has an invalid import",
					current.ID,
				)
			}
			imports[importPath] = imported.ID
		}
		records[index] = driverPackageEvidence{
			ID: current.ID, Name: current.Name, PackagePath: current.PkgPath,
			Directory: current.Dir, ForTest: current.ForTest, Module: current.Module,
			GoFiles:         slices.Clone(current.GoFiles),
			CompiledGoFiles: slices.Clone(current.CompiledGoFiles),
			OtherFiles:      slices.Clone(current.OtherFiles),
			EmbedFiles:      slices.Clone(current.EmbedFiles),
			EmbedPatterns:   slices.Clone(current.EmbedPatterns),
			IgnoredFiles:    slices.Clone(current.IgnoredFiles),
			Imports:         imports,
		}
	}
	payload, err := json.Marshal(driverEvidence{
		Roots: slices.Clone(roots), Packages: records,
	})
	if err != nil {
		return fmt.Errorf("exact Go packages driver: encode evidence: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0)
	if err != nil {
		return fmt.Errorf("exact Go packages driver: open evidence: %w", err)
	}
	_, writeErr := file.Write(payload)
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := firstError(writeErr, syncErr, closeErr); err != nil {
		return fmt.Errorf("exact Go packages driver: write evidence: %w", err)
	}
	return nil
}

func joinDriverEvidence(path string, loaded []*packages.Package) error {
	payload, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("load exact Go packages: read driver evidence: %w", err)
	}
	var evidence driverEvidence
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&evidence); err != nil {
		return fmt.Errorf("load exact Go packages: decode driver evidence: %w", err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return fmt.Errorf("load exact Go packages: decode driver evidence: %w", err)
	}
	rootIDs := make([]string, len(loaded))
	for index, root := range loaded {
		rootIDs[index] = root.ID
	}
	if !slices.Equal(rootIDs, evidence.Roots) {
		return fmt.Errorf(
			"load exact Go packages: driver root join differs: response=%v evidence=%v",
			rootIDs,
			evidence.Roots,
		)
	}
	byID := make(map[string]*packages.Package)
	var duplicateResponseID string
	packages.Visit(loaded, nil, func(current *packages.Package) {
		if previous, duplicate := byID[current.ID]; duplicate && previous != current {
			duplicateResponseID = current.ID
		}
		byID[current.ID] = current
	})
	if duplicateResponseID != "" {
		return fmt.Errorf(
			"load exact Go packages: response package %q is duplicated",
			duplicateResponseID,
		)
	}
	if len(byID) != len(evidence.Packages) {
		return driverEvidenceSetError(byID, evidence.Packages)
	}
	seen := make(map[string]struct{}, len(evidence.Packages))
	for _, record := range evidence.Packages {
		current, ok := byID[record.ID]
		if !ok {
			return driverEvidenceSetError(byID, evidence.Packages)
		}
		if _, duplicate := seen[record.ID]; duplicate {
			return fmt.Errorf("load exact Go packages: driver evidence package %q is duplicated", record.ID)
		}
		seen[record.ID] = struct{}{}
		if err := joinDriverPackage(current, record); err != nil {
			return err
		}
	}
	return nil
}

func joinDriverPackage(
	current *packages.Package,
	record driverPackageEvidence,
) error {
	imports := make(map[string]string, len(current.Imports))
	for importPath, imported := range current.Imports {
		if imported == nil {
			return fmt.Errorf("load exact Go packages: response package %q has a nil import", current.ID)
		}
		imports[importPath] = imported.ID
	}
	var differences []string
	if current.Name != record.Name {
		differences = append(differences, "name")
	}
	if current.PkgPath != record.PackagePath {
		differences = append(differences, "package-path")
	}
	for _, comparison := range []struct {
		name  string
		left  []string
		right []string
	}{
		{"go-files", current.GoFiles, record.GoFiles},
		{"compiled-go-files", current.CompiledGoFiles, record.CompiledGoFiles},
		{"other-files", current.OtherFiles, record.OtherFiles},
		{"embed-files", current.EmbedFiles, record.EmbedFiles},
		{"embed-patterns", current.EmbedPatterns, record.EmbedPatterns},
		{"ignored-files", current.IgnoredFiles, record.IgnoredFiles},
	} {
		if !slices.Equal(comparison.left, comparison.right) {
			differences = append(differences, comparison.name)
		}
	}
	if !equalStringMap(imports, record.Imports) {
		differences = append(differences, "imports")
	}
	if len(differences) != 0 {
		return fmt.Errorf(
			"load exact Go packages: response and evidence differ for package %q fields %v",
			current.ID,
			differences,
		)
	}
	current.Dir = record.Directory
	current.ForTest = record.ForTest
	current.Module = record.Module
	return nil
}

func driverEvidenceSetError(
	response map[string]*packages.Package,
	records []driverPackageEvidence,
) error {
	evidence := make(map[string]struct{}, len(records))
	for _, record := range records {
		evidence[record.ID] = struct{}{}
	}
	var responseOnly []string
	var evidenceOnly []string
	for id := range response {
		if _, ok := evidence[id]; !ok {
			responseOnly = append(responseOnly, id)
		}
	}
	for id := range evidence {
		if _, ok := response[id]; !ok {
			evidenceOnly = append(evidenceOnly, id)
		}
	}
	sort.Strings(responseOnly)
	sort.Strings(evidenceOnly)
	return fmt.Errorf(
		"load exact Go packages: driver package join differs: response-only=%v evidence-only=%v",
		responseOnly,
		evidenceOnly,
	)
}

func equalStringMap(left map[string]string, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func firstError(errors ...error) error {
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}
