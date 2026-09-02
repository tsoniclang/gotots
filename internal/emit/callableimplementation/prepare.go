package callableimplementation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"

	implementationcontract "github.com/tsoniclang/gotots/internal/contracts/implementation"
	"github.com/tsoniclang/gotots/internal/load"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
)

type Config struct {
	ContractPaths        []string
	CertificationSources []implementationcontract.CertificationSource
	BuildProfile         load.BuildProfile
	Compilation          CompilationDocument
}

func PrepareAll(config Config) (*Prepared, error) {
	if len(config.ContractPaths) == 0 || !config.BuildProfile.Valid() ||
		!config.Compilation.valid() {
		return nil, &Error{Operation: "configure", Reason: "required input is absent"}
	}
	paths := slices.Clone(config.ContractPaths)
	slices.Sort(paths)
	prepared := &Prepared{
		buildProfile: config.BuildProfile,
		compilation:  config.Compilation,
	}
	outputs := make(map[string]struct{}, len(paths))
	identities := make(map[string]struct{})
	for _, contractPath := range paths {
		module, err := prepareOne(config, contractPath)
		if err != nil {
			return nil, err
		}
		if prepared.sourceProgramDigest == "" {
			prepared.sourceProgramDigest = module.sourceProgramDigest
		} else if prepared.sourceProgramDigest != module.sourceProgramDigest {
			return nil, &Error{
				Operation: "join source program",
				Reason:    "contracts select different source snapshots",
			}
		}
		if _, duplicate := outputs[module.outputPath]; duplicate {
			return nil, &Error{
				Operation: "admit output", Subject: module.outputPath,
				Reason: "target module has multiple owners",
			}
		}
		outputs[module.outputPath] = struct{}{}
		for _, claim := range module.callableClaims {
			if _, duplicate := identities[claim.SourceIdentity]; duplicate {
				return nil, &Error{
					Operation: "admit callable", Subject: claim.SourceIdentity,
					Reason: "source callable has multiple implementation owners",
				}
			}
			identities[claim.SourceIdentity] = struct{}{}
		}
		prepared.modules = append(prepared.modules, module)
	}
	digest := sha256.New()
	for _, module := range prepared.modules {
		digest.Write([]byte(module.outputPath))
		digest.Write([]byte{0})
		digest.Write([]byte(module.digest))
		digest.Write([]byte{0})
	}
	prepared.digest = hex.EncodeToString(digest.Sum(nil))
	return prepared, nil
}

func prepareOne(config Config, contractPath string) (Module, error) {
	absolute, err := filepath.Abs(contractPath)
	if err != nil {
		return Module{}, contractError("resolve contract", contractPath, err)
	}
	payload, err := os.ReadFile(absolute)
	if err != nil {
		return Module{}, contractError("read contract", absolute, err)
	}
	document, err := decodeDocument(payload)
	if err != nil {
		return Module{}, err
	}
	if err := validateDocument(document); err != nil {
		return Module{}, err
	}
	if err := verifyBuild(config.BuildProfile, document.Build); err != nil {
		return Module{}, err
	}
	if document.Compilation != config.Compilation {
		return Module{}, &Error{
			Operation: "join compilation profile",
			Reason:    "selected profile differs",
		}
	}
	directory := filepath.Dir(absolute)
	sourcePath, err := resolveOwnedPath(directory, document.Source)
	if err != nil {
		return Module{}, err
	}
	certificationPaths, err := resolveOwnedPaths(directory, document.CertificationSources)
	if err != nil {
		return Module{}, err
	}
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return Module{}, contractError("read source", sourcePath, err)
	}
	canonical, err := json.Marshal(document)
	if err != nil {
		return Module{}, contractError("encode contract", absolute, err)
	}
	digest := sha256.New()
	digest.Write(canonical)
	digest.Write([]byte{0})
	digest.Write(source)
	localCertificationSources, err := implementationcontract.LoadCertificationSources(
		certificationPaths,
	)
	if err != nil {
		return Module{}, err
	}
	certificationSources, err := implementationcontract.MergeCertificationSources(
		config.CertificationSources,
		localCertificationSources,
	)
	if err != nil {
		return Module{}, err
	}
	for _, selected := range certificationSources {
		evidence, readErr := implementationcontract.VerifyCertificationSource(selected)
		if readErr != nil {
			return Module{}, readErr
		}
		digest.Write([]byte{0})
		digest.Write(evidence)
	}
	sourceHash := sha256.Sum256(source)
	return Module{
		sourceProgramDigest:  document.SourceProgramDigest,
		packagePath:          document.Package.ImportPath,
		modulePath:           document.Package.ModulePath,
		moduleVersion:        document.Package.ModuleVersion,
		sourcePath:           sourcePath,
		outputPath:           document.Output,
		sourceDigest:         hex.EncodeToString(sourceHash[:]),
		digest:               hex.EncodeToString(digest.Sum(nil)),
		envelope:             cloneEnvelope(document.Envelope),
		callableClaims:       slices.Clone(document.Callables),
		certificationSources: certificationSources,
	}, nil
}

func cloneEnvelope(
	source implementationcontract.Envelope,
) implementationcontract.Envelope {
	result := source
	result.PreservedObservables = slices.Clone(source.PreservedObservables)
	result.Evidence = slices.Clone(source.Evidence)
	return result
}

func decodeDocument(payload []byte) (Document, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var document Document
	if err := decoder.Decode(&document); err != nil {
		return Document{}, contractError("decode", "", err)
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return Document{}, contractError("decode", "", err)
	}
	return document, nil
}

func validateDocument(document Document) error {
	if document.SchemaVersion != SchemaVersion {
		return &Error{Operation: "validate", Reason: "schema version is unsupported"}
	}
	if !validSHA256(document.SourceProgramDigest) ||
		document.Package.ImportPath == "" || document.Package.ModulePath == "" {
		return &Error{Operation: "validate", Reason: "package identity is incomplete"}
	}
	if document.Build.GoVersion == "" || document.Build.GOOS == "" ||
		document.Build.GOARCH == "" || !document.Compilation.valid() {
		return &Error{Operation: "validate", Reason: "profile is incomplete"}
	}
	if document.Source == "" || len(document.Callables) == 0 ||
		!document.Envelope.Valid() {
		return &Error{Operation: "validate", Reason: "implementation evidence is incomplete"}
	}
	if _, err := targetoutput.CallableImplementationPath(document.Output); err != nil {
		return &Error{Operation: "validate output", Subject: document.Output, Reason: err.Error()}
	}
	if !sort.StringsAreSorted(document.CertificationSources) {
		return &Error{Operation: "validate", Reason: "certification sources are not sorted"}
	}
	for index, source := range document.CertificationSources {
		if source == "" || source == document.Source ||
			index > 0 && document.CertificationSources[index-1] == source {
			return &Error{Operation: "validate", Reason: "certification source is invalid"}
		}
	}
	seenExports := make(map[string]struct{}, len(document.Callables))
	previous := ""
	for _, callable := range document.Callables {
		if callable.SourceIdentity == "" || callable.SourceSignature == "" ||
			!validSHA256(callable.SourceBodyDigest) ||
			!callable.Variant.Valid() || callable.Export == "" {
			return &Error{Operation: "validate callable", Reason: "claim is incomplete"}
		}
		key := callable.SourceIdentity + "\x00" + string(callable.Variant)
		if previous != "" && previous >= key {
			return &Error{
				Operation: "validate callable", Subject: callable.SourceIdentity,
				Reason: "claims are not strictly ordered",
			}
		}
		previous = key
		if _, duplicate := seenExports[callable.Export]; duplicate {
			return &Error{
				Operation: "validate callable", Subject: callable.Export,
				Reason: "target export is duplicated",
			}
		}
		seenExports[callable.Export] = struct{}{}
	}
	return nil
}

func validSHA256(encoded string) bool {
	decoded, err := hex.DecodeString(encoded)
	return err == nil && len(decoded) == sha256.Size
}

func verifyBuild(selected load.BuildProfile, document BuildDocument) error {
	if !selected.Valid() || selected.ToolchainVersion() != document.GoVersion ||
		selected.GOOS() != document.GOOS || selected.GOARCH() != document.GOARCH ||
		selected.CgoEnabled() != document.CGOEnabled ||
		!slices.Equal(selected.Tags(), document.BuildTags) {
		return &Error{Operation: "join build profile", Reason: "selected profile differs"}
	}
	return nil
}

func resolveOwnedPaths(root string, paths []string) ([]string, error) {
	result := make([]string, len(paths))
	for index, path := range paths {
		resolved, err := resolveOwnedPath(root, path)
		if err != nil {
			return nil, err
		}
		result[index] = resolved
	}
	return result, nil
}

func resolveOwnedPath(root string, selected string) (string, error) {
	if selected == "" || filepath.IsAbs(selected) {
		return "", &Error{Operation: "resolve path", Subject: selected, Reason: "path is not relative"}
	}
	clean := filepath.Clean(selected)
	if clean == "." || clean == ".." || len(clean) >= 3 && clean[:3] == ".."+string(filepath.Separator) {
		return "", &Error{Operation: "resolve path", Subject: selected, Reason: "path escapes its contract"}
	}
	return filepath.Join(root, clean), nil
}

func contractError(operation string, subject string, cause error) error {
	return &Error{Operation: operation, Subject: subject, Reason: cause.Error()}
}
