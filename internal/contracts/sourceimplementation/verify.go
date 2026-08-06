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

	"github.com/tsoniclang/gotots/internal/contracts/callableabi"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Config struct {
	RepositoryRoot string
	Program        *load.Program
	ContractPaths  []string
	ScratchRoot    string
	Compilation    CompilationDocument
}

func VerifyAll(config Config) (
	certificate *Certificate,
	resultErr error,
) {
	if config.RepositoryRoot == "" || config.Program == nil ||
		len(config.ContractPaths) == 0 || config.ScratchRoot == "" ||
		!config.Compilation.valid() {
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
	certificate = &Certificate{
		byPath:     make(map[string]Implementation),
		byCallable: make(map[string]callableabi.Callable),
	}
	if err := certificate.bindCompilation(config.Compilation); err != nil {
		return nil, err
	}
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
	if document.Compilation != config.Compilation {
		return Implementation{}, &Error{
			Operation: "join compilation profile",
			Reason:    "selected profile differs",
		}
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
	certificationPaths, err := resolveOwnedPaths(
		directory,
		document.CertificationSources,
	)
	if err != nil {
		return Implementation{}, err
	}
	privatePaths := make([]string, len(document.PrivateModules))
	for index, module := range document.PrivateModules {
		privatePaths[index], err = resolveOwnedPath(directory, module.Source)
		if err != nil {
			return Implementation{}, err
		}
	}
	projectSources := append([]string{sourcePath}, privatePaths...)
	projectSources = append(projectSources, certificationPaths...)
	tsconfig, err := verifyTSConfig(tsconfigPath, directory, projectSources)
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
	callables, err := verifyCallableABIs(
		selected,
		project,
		projectExports,
		document.Compilation,
	)
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
	privateModules := make([]PrivateModule, 0, len(document.PrivateModules))
	privateHashes := make([][32]byte, 0, len(document.PrivateModules))
	for _, selectedModule := range document.PrivateModules {
		module, moduleHash, moduleErr := verifyPrivateModule(
			selected,
			directory,
			project,
			selectedModule,
		)
		if moduleErr != nil {
			return Implementation{}, moduleErr
		}
		privateModules = append(privateModules, module)
		privateHashes = append(privateHashes, moduleHash)
	}
	implementationHash := sha256.New()
	implementationHash.Write(canonical)
	implementationHash.Write([]byte{0})
	implementationHash.Write(sourceHash[:])
	implementationHash.Write([]byte{0})
	implementationHash.Write(tsconfig)
	for _, certificationPath := range certificationPaths {
		payload, readErr := os.ReadFile(certificationPath)
		if readErr != nil {
			return Implementation{}, implementationError(
				"read certification source",
				certificationPath,
				readErr,
			)
		}
		implementationHash.Write([]byte{0})
		implementationHash.Write(payload)
	}
	for _, export := range exports {
		implementationHash.Write([]byte{0})
		implementationHash.Write([]byte(export.fingerprint))
	}
	for _, callable := range callables {
		implementationHash.Write([]byte{0})
		implementationHash.Write([]byte(callable.Fingerprint()))
	}
	for index, module := range privateModules {
		implementationHash.Write([]byte{0})
		implementationHash.Write([]byte(module.goFile))
		implementationHash.Write([]byte{0})
		implementationHash.Write(privateHashes[index][:])
		for _, export := range module.exports {
			implementationHash.Write([]byte{0})
			implementationHash.Write([]byte(export.fingerprint))
		}
	}
	return Implementation{
		packagePath:    document.Package.ImportPath,
		modulePath:     document.Package.ModulePath,
		moduleVersion:  document.Package.ModuleVersion,
		sourcePath:     sourcePath,
		digest:         hex.EncodeToString(implementationHash.Sum(nil)),
		sourceDigest:   hex.EncodeToString(sourceHash[:]),
		envelope:       document.Envelope.Kind,
		exports:        exports,
		sourceFile:     sourceFile,
		privateModules: privateModules,
		callables:      callables,
	}, nil
}

func verifyPrivateModule(
	selected *load.Package,
	directory string,
	project *tsgo.ProjectInspection,
	document PrivateModuleDocument,
) (PrivateModule, [32]byte, error) {
	var zero [32]byte
	matched := false
	for _, sourceFile := range selected.Files() {
		if filepath.Base(sourceFile.Path()) == document.GoFile {
			matched = true
			break
		}
	}
	if !matched {
		return PrivateModule{}, zero, &Error{
			Operation: "join private module",
			Subject:   document.GoFile,
			Reason:    "selected Go source file is absent",
		}
	}
	sourcePath, err := resolveOwnedPath(directory, document.Source)
	if err != nil {
		return PrivateModule{}, zero, err
	}
	projectExports, err := project.Exports(sourcePath)
	if err != nil {
		return PrivateModule{}, zero, err
	}
	exports, err := verifyExports(document.Exports, projectExports)
	if err != nil {
		return PrivateModule{}, zero, err
	}
	sourceFile, err := project.SourceFile(sourcePath)
	if err != nil {
		return PrivateModule{}, zero, err
	}
	if err := verifyBodyFree(project, sourcePath, document.GoFile); err != nil {
		return PrivateModule{}, zero, err
	}
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return PrivateModule{}, zero, implementationError("read source", sourcePath, err)
	}
	digest := sha256.Sum256(source)
	return PrivateModule{
		goFile:       document.GoFile,
		sourcePath:   sourcePath,
		sourceDigest: hex.EncodeToString(digest[:]),
		exports:      exports,
		sourceFile:   sourceFile,
	}, digest, nil
}

func verifyBodyFree(
	project *tsgo.ProjectInspection,
	sourcePath string,
	goFile string,
) error {
	statements, err := project.SourceStatements(sourcePath)
	if err != nil {
		return err
	}
	for _, statement := range statements {
		if statement.Kind() == tsgo.SyntaxKindInterfaceDeclaration ||
			statement.Kind() == tsgo.SyntaxKindTypeAliasDeclaration ||
			(statement.Kind() == tsgo.SyntaxKindImportDeclaration ||
				statement.Kind() == tsgo.SyntaxKindExportDeclaration) && statement.TypeOnly() {
			continue
		}
		return &Error{
			Operation: "validate private module",
			Subject:   goFile,
			Reason:    fmt.Sprintf("statement kind %d is executable or unsupported", statement.Kind()),
		}
	}
	return nil
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
		document.Source == "" || document.TSConfig == "" || len(document.Exports) == 0 ||
		!document.Compilation.valid() {
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
	seenSources := map[string]struct{}{document.Source: {}}
	if document.TSConfig == document.Source {
		return &Error{Operation: "validate contract", Reason: "implementation source is duplicated"}
	}
	if !sortedUnique(document.CertificationSources) {
		return &Error{Operation: "validate contract", Reason: "certification sources are not sorted and unique"}
	}
	for _, source := range document.CertificationSources {
		if source == "" {
			return &Error{Operation: "validate contract", Reason: "certification source is invalid"}
		}
		if _, duplicate := seenSources[source]; duplicate || source == document.TSConfig {
			return &Error{Operation: "validate contract", Reason: "implementation source is duplicated"}
		}
		seenSources[source] = struct{}{}
	}
	previousGoFile := ""
	for _, module := range document.PrivateModules {
		if module.GoFile == "" || filepath.Base(module.GoFile) != module.GoFile ||
			filepath.Ext(module.GoFile) != ".go" || module.Source == "" ||
			len(module.Exports) == 0 || !sortedUnique(module.Exports) {
			return &Error{Operation: "validate contract", Reason: "private module is invalid"}
		}
		if previousGoFile >= module.GoFile {
			return &Error{Operation: "validate contract", Reason: "private modules are not sorted and unique"}
		}
		previousGoFile = module.GoFile
		if _, duplicate := seenSources[module.Source]; duplicate {
			return &Error{Operation: "validate contract", Reason: "implementation source is duplicated"}
		}
		seenSources[module.Source] = struct{}{}
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

func resolveOwnedPaths(root string, selected []string) ([]string, error) {
	result := make([]string, len(selected))
	for index, path := range selected {
		resolved, err := resolveOwnedPath(root, path)
		if err != nil {
			return nil, err
		}
		result[index] = resolved
	}
	return result, nil
}

func verifyTSConfig(
	path string,
	root string,
	sourcePaths []string,
) ([]byte, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, implementationError("read tsconfig", path, err)
	}
	var document map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(payload))
	if err := decoder.Decode(&document); err != nil {
		return nil, implementationError("decode tsconfig", path, err)
	}
	for key := range document {
		if key != "compilerOptions" && key != "files" {
			return nil, &Error{
				Operation: "validate tsconfig",
				Subject:   path,
				Reason:    "unsupported top-level field " + key,
			}
		}
	}
	var files []string
	if err := json.Unmarshal(document["files"], &files); err != nil || !sortedUnique(files) {
		return nil, &Error{
			Operation: "validate tsconfig",
			Subject:   path,
			Reason:    "files must be a sorted unique array",
		}
	}
	expected := make([]string, len(sourcePaths))
	for index, sourcePath := range sourcePaths {
		relative, relativeErr := filepath.Rel(root, sourcePath)
		if relativeErr != nil {
			return nil, implementationError("relativize tsconfig source", sourcePath, relativeErr)
		}
		expected[index] = filepath.ToSlash(relative)
	}
	slices.Sort(expected)
	if !slices.Equal(files, expected) {
		return nil, &Error{
			Operation: "validate tsconfig",
			Subject:   path,
			Reason:    fmt.Sprintf("files %v differ from contract sources %v", files, expected),
		}
	}
	var options map[string]json.RawMessage
	if err := json.Unmarshal(document["compilerOptions"], &options); err != nil {
		return nil, implementationError("decode tsconfig compiler options", path, err)
	}
	if !jsonBool(options, "strict") || !jsonBool(options, "noEmit") ||
		jsonBool(options, "skipLibCheck") || jsonBool(options, "allowJs") ||
		jsonBool(options, "noCheck") || !jsonString(options, "target", "ES2022") ||
		!jsonString(options, "module", "NodeNext") ||
		!jsonString(options, "moduleResolution", "NodeNext") {
		return nil, &Error{
			Operation: "validate tsconfig",
			Subject:   path,
			Reason:    "strict NodeNext no-emit compiler options are required",
		}
	}
	return payload, nil
}

func jsonBool(values map[string]json.RawMessage, name string) bool {
	var selected bool
	return json.Unmarshal(values[name], &selected) == nil && selected
}

func jsonString(
	values map[string]json.RawMessage,
	name string,
	expected string,
) bool {
	var selected string
	return json.Unmarshal(values[name], &selected) == nil && selected == expected
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
