package sourceimplementation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Config struct {
	RepositoryRoot string
	Program        *load.Program
	ContractPaths  []string
	ScratchRoot    string
}

func VerifyAll(config Config) (
	certificate *Certificate,
	resultErr error,
) {
	if config.RepositoryRoot == "" || config.Program == nil ||
		len(config.ContractPaths) == 0 || config.ScratchRoot == "" {
		return nil, &Error{Operation: "configure", Reason: "required input is absent"}
	}
	if err := os.MkdirAll(config.ScratchRoot, 0o755); err != nil {
		return nil, &Error{Operation: "configure", Subject: config.ScratchRoot, Reason: err.Error()}
	}
	client, err := tsgo.StartClient(config.RepositoryRoot, config.ScratchRoot)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := client.Close(); resultErr == nil && err != nil {
			certificate = nil
			resultErr = implementationError("close TS-Go client", "", err)
		}
	}()
	paths := slices.Clone(config.ContractPaths)
	slices.Sort(paths)
	certificate = &Certificate{byPath: make(map[string]Implementation)}
	for _, contractPath := range paths {
		implementation, verifyErr := verifyOne(config, client, contractPath)
		if verifyErr != nil {
			return nil, verifyErr
		}
		if err := certificate.add(implementation); err != nil {
			return nil, err
		}
	}
	digest := sha256.New()
	for _, implementation := range certificate.Implementations() {
		digest.Write([]byte(implementation.packagePath))
		digest.Write([]byte{0})
		digest.Write([]byte(implementation.digest))
		digest.Write([]byte{0})
	}
	certificate.digest = hex.EncodeToString(digest.Sum(nil))
	return certificate, nil
}

func verifyOne(
	config Config,
	client *tsgo.Client,
	contractPath string,
) (Implementation, error) {
	absolute, err := filepath.Abs(contractPath)
	if err != nil {
		return Implementation{}, implementationError("resolve contract", contractPath, err)
	}
	payload, err := os.ReadFile(absolute)
	if err != nil {
		return Implementation{}, implementationError("read contract", absolute, err)
	}
	document, err := decodeDocument(payload)
	if err != nil {
		return Implementation{}, err
	}
	if err := validateDocument(document); err != nil {
		return Implementation{}, err
	}
	selected := config.Program.PackageByPath(document.Package.ImportPath)
	if selected == nil || selected.Kind() != load.PackageSource ||
		selected.ModulePath() != document.Package.ModulePath ||
		selected.ModuleVersion() != document.Package.ModuleVersion {
		return Implementation{}, &Error{
			Operation: "join package",
			Subject:   document.Package.ImportPath,
			Reason:    "selected source package identity differs",
		}
	}
	if err := verifyBuild(config.Program, document.Build); err != nil {
		return Implementation{}, err
	}
	directory := filepath.Dir(absolute)
	sourcePath, err := resolveOwnedPath(directory, document.Source)
	if err != nil {
		return Implementation{}, err
	}
	tsconfigPath, err := resolveOwnedPath(directory, document.TSConfig)
	if err != nil {
		return Implementation{}, err
	}
	if err := tsgo.Compile(
		context.Background(),
		config.RepositoryRoot,
		directory,
		[]string{"--noEmit", "-p", tsconfigPath},
	); err != nil {
		return Implementation{}, implementationError("typecheck", tsconfigPath, err)
	}
	project, err := client.OpenProject(tsconfigPath)
	if err != nil {
		return Implementation{}, err
	}
	projectExports, err := project.Exports(sourcePath)
	if err != nil {
		return Implementation{}, err
	}
	exports, err := verifyExports(document.Exports, projectExports)
	if err != nil {
		return Implementation{}, err
	}
	sourceFile, err := project.SourceFile(sourcePath)
	if err != nil {
		return Implementation{}, err
	}
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return Implementation{}, implementationError("read source", sourcePath, err)
	}
	canonical, err := json.Marshal(document)
	if err != nil {
		return Implementation{}, implementationError("encode contract", absolute, err)
	}
	sourceHash := sha256.Sum256(source)
	implementationHash := sha256.New()
	implementationHash.Write(canonical)
	implementationHash.Write([]byte{0})
	implementationHash.Write(sourceHash[:])
	for _, export := range exports {
		implementationHash.Write([]byte{0})
		implementationHash.Write([]byte(export.fingerprint))
	}
	return Implementation{
		packagePath:   document.Package.ImportPath,
		modulePath:    document.Package.ModulePath,
		moduleVersion: document.Package.ModuleVersion,
		sourcePath:    sourcePath,
		digest:        hex.EncodeToString(implementationHash.Sum(nil)),
		sourceDigest:  hex.EncodeToString(sourceHash[:]),
		envelope:      document.Envelope.Kind,
		exports:       exports,
		sourceFile:    sourceFile,
	}, nil
}

func decodeDocument(payload []byte) (Document, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var document Document
	if err := decoder.Decode(&document); err != nil {
		return Document{}, implementationError("decode contract", "", err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return Document{}, implementationError("decode contract", "", err)
	}
	return document, nil
}

func validateDocument(document Document) error {
	if document.SchemaVersion != SchemaVersion {
		return &Error{Operation: "validate contract", Reason: "schema version is unsupported"}
	}
	if document.Package.ImportPath == "" || document.Package.ModulePath == "" ||
		document.Source == "" || document.TSConfig == "" || len(document.Exports) == 0 {
		return &Error{Operation: "validate contract", Reason: "required field is absent"}
	}
	if !document.Envelope.Kind.Valid() {
		return &Error{Operation: "validate contract", Reason: "equivalence envelope is invalid"}
	}
	if document.Envelope.Kind == EnvelopeExact {
		if document.Envelope.RelaxedBehavior != "" ||
			len(document.Envelope.PreservedObservables) != 0 ||
			len(document.Envelope.Evidence) != 0 {
			return &Error{Operation: "validate contract", Reason: "exact envelope carries a relaxation"}
		}
	} else if document.Envelope.RelaxedBehavior == "" ||
		len(document.Envelope.PreservedObservables) == 0 ||
		len(document.Envelope.Evidence) == 0 {
		return &Error{Operation: "validate contract", Reason: "internal-algorithm envelope lacks proof"}
	}
	if !sortedUnique(document.Exports) ||
		!sortedUnique(document.Envelope.PreservedObservables) ||
		!sortedUnique(document.Envelope.Evidence) ||
		!sortedUnique(document.Build.BuildTags) {
		return &Error{Operation: "validate contract", Reason: "set field is not sorted and unique"}
	}
	for _, name := range document.Exports {
		if strings.TrimSpace(name) != name || name == "" {
			return &Error{Operation: "validate contract", Reason: "export name is invalid"}
		}
	}
	return nil
}

func verifyBuild(program *load.Program, selected BuildDocument) error {
	profile := program.BuildProfile()
	if profile.ToolchainVersion() != selected.GoVersion ||
		profile.GOOS() != selected.GOOS || profile.GOARCH() != selected.GOARCH ||
		profile.CgoEnabled() != selected.CGOEnabled ||
		!slices.Equal(profile.Tags(), selected.BuildTags) {
		return &Error{Operation: "join build profile", Reason: "selected profile differs"}
	}
	return nil
}

func verifyExports(expected []string, selected []tsgo.ProjectExport) ([]Export, error) {
	actual := make([]string, len(selected))
	result := make([]Export, len(selected))
	for index, export := range selected {
		actual[index] = export.Name()
		fingerprint := sha256.Sum256([]byte(export.Name() + "\x00" + export.TypeString()))
		result[index] = Export{
			name:        export.Name(),
			typeString:  export.TypeString(),
			fingerprint: hex.EncodeToString(fingerprint[:]),
		}
	}
	if !slices.Equal(expected, actual) {
		return nil, &Error{
			Operation: "join exports",
			Reason:    fmt.Sprintf("implementation exports %v, want %v", actual, expected),
		}
	}
	return result, nil
}

func resolveOwnedPath(root string, selected string) (string, error) {
	if filepath.IsAbs(selected) || filepath.Clean(selected) != selected ||
		selected == "." || strings.HasPrefix(selected, ".."+string(filepath.Separator)) {
		return "", &Error{Operation: "resolve path", Subject: selected, Reason: "path is not contract-relative"}
	}
	target := filepath.Join(root, selected)
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", &Error{Operation: "resolve path", Subject: selected, Reason: "path escapes contract directory"}
	}
	return target, nil
}

func sortedUnique(values []string) bool {
	if !slices.IsSorted(values) {
		return false
	}
	for index := 1; index < len(values); index++ {
		if values[index-1] == values[index] {
			return false
		}
	}
	return true
}

func implementationError(operation string, subject string, err error) error {
	return &Error{Operation: operation, Subject: subject, Reason: err.Error()}
}
