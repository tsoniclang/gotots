package callableimplementation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type StagedVerificationConfig struct {
	RepositoryRoot string
	ScratchRoot    string
	TSGoTool       tsgo.Tool
	Generated      []StagedTarget
	Modules        []StagedModule
	Callables      []StagedCallable
}

func VerifyStagedGeneratedContracts(
	config StagedVerificationConfig,
) (verified []VerifiedModule, resultErr error) {
	if config.RepositoryRoot == "" || !filepath.IsAbs(config.ScratchRoot) ||
		!config.TSGoTool.Valid() || len(config.Generated) == 0 ||
		len(config.Modules) == 0 || len(config.Callables) == 0 {
		return nil, &Error{
			Operation: "verify staged generated contract",
			Reason:    "verification configuration is incomplete",
		}
	}
	generated, err := validateStagedTargets(config.Generated)
	if err != nil {
		return nil, err
	}
	modules, err := validateStagedModules(config.Modules)
	if err != nil {
		return nil, err
	}
	if err := validateStagedCallables(config.Callables, generated, modules); err != nil {
		return nil, err
	}
	if err := os.Mkdir(config.ScratchRoot, 0o700); err != nil {
		return nil, contractError("create verification scratch", config.ScratchRoot, err)
	}
	client, err := tsgo.StartClientWithTool(config.TSGoTool, config.ScratchRoot)
	if err != nil {
		return nil, err
	}
	closed := false
	defer func() {
		if !closed {
			if closeErr := client.Close(); resultErr == nil && closeErr != nil {
				resultErr = closeErr
			}
		}
	}()
	projectRoot := filepath.Join(config.ScratchRoot, "project")
	if err := materializeGeneratedTargets(client, projectRoot, config.Generated); err != nil {
		return nil, err
	}
	if err := materializeImplementationModules(projectRoot, config.Modules); err != nil {
		return nil, err
	}
	certificationPaths, err := materializeCertificationSources(projectRoot, config.Modules)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(
		filepath.Join(projectRoot, "package.json"),
		[]byte("{\"type\":\"module\"}\n"),
		0o644,
	); err != nil {
		return nil, contractError("write package identity", projectRoot, err)
	}
	configPath := filepath.Join(projectRoot, "tsconfig.json")
	configPayload, err := tsgo.EncodeStrictProjectConfig(
		config.RepositoryRoot,
		projectRoot,
	)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(configPath, configPayload, 0o644); err != nil {
		return nil, contractError("write strict project", configPath, err)
	}
	if err := tsgo.CompileWithTool(
		context.Background(),
		config.TSGoTool,
		projectRoot,
		[]string{"--noEmit", "-p", configPath},
	); err != nil {
		return nil, contractError("typecheck staged project", configPath, err)
	}
	project, err := client.OpenProject(configPath)
	if err != nil {
		return nil, err
	}
	for _, module := range config.Modules {
		path := filepath.Join(
			projectRoot,
			filepath.FromSlash(module.outputPath),
		)
		if err := rejectCallableImplementationSource(
			project,
			path,
			module.outputPath,
			tsgo.CallableImplementationSourceModule,
		); err != nil {
			return nil, err
		}
	}
	for _, path := range certificationPaths {
		if err := rejectCallableImplementationSource(
			project,
			path,
			path,
			tsgo.CallableImplementationSourceCertification,
		); err != nil {
			return nil, err
		}
	}
	moduleExports, err := inspectModuleExports(project, projectRoot, config.Modules)
	if err != nil {
		return nil, err
	}
	generatedExports := make(map[string][]tsgo.ProjectExport)
	for _, callable := range config.Callables {
		manual := moduleExports[callable.implementationOutput][callable.implementationExport]
		path := callable.generated.outputPath
		exports := generatedExports[path]
		if exports == nil {
			exports, err = project.Exports(filepath.Join(projectRoot, filepath.FromSlash(path)))
			if err != nil {
				return nil, err
			}
			generatedExports[path] = exports
		}
		if err := verifyCallableTarget(project, callable, manual, exports); err != nil {
			return nil, err
		}
	}
	orderedModules := slices.Clone(config.Modules)
	sort.Slice(orderedModules, func(left, right int) bool {
		return orderedModules[left].outputPath < orderedModules[right].outputPath
	})
	verified = make([]VerifiedModule, len(orderedModules))
	for index, module := range orderedModules {
		sourceFile, sourceErr := project.SourceFile(filepath.Join(
			projectRoot,
			filepath.FromSlash(module.outputPath),
		))
		if sourceErr != nil {
			return nil, sourceErr
		}
		verified[index] = VerifiedModule{
			outputPath: module.outputPath,
			sourceFile: sourceFile,
		}
	}
	if err := client.Close(); err != nil {
		return nil, err
	}
	closed = true
	if err := os.RemoveAll(config.ScratchRoot); err != nil {
		return nil, contractError("remove verification scratch", config.ScratchRoot, err)
	}
	return verified, nil
}

func validateStagedTargets(targets []StagedTarget) (map[string]struct{}, error) {
	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		if !validTargetPath(target.outputPath) || !filepath.IsAbs(target.protocolPath) ||
			target.protocolHash == ([sha256.Size]byte{}) {
			return nil, &Error{Operation: "validate staged target", Reason: "target is invalid"}
		}
		if _, duplicate := seen[target.outputPath]; duplicate {
			return nil, &Error{
				Operation: "validate staged target", Subject: target.outputPath,
				Reason: "target is duplicated",
			}
		}
		seen[target.outputPath] = struct{}{}
	}
	return seen, nil
}

func validateStagedModules(
	modules []StagedModule,
) (map[string]StagedModule, error) {
	seen := make(map[string]StagedModule, len(modules))
	for _, module := range modules {
		digest, digestErr := hex.DecodeString(module.sourceDigest)
		if !filepath.IsAbs(module.sourcePath) || filepath.Clean(module.sourcePath) != module.sourcePath ||
			!validTargetPath(module.outputPath) ||
			digestErr != nil || len(digest) != sha256.Size || len(module.exports) == 0 ||
			!sort.StringsAreSorted(module.exports) ||
			!sort.SliceIsSorted(module.certificationSources, func(left, right int) bool {
				return module.certificationSources[left].sourcePath <
					module.certificationSources[right].sourcePath
			}) {
			return nil, &Error{
				Operation: "validate staged module", Subject: module.outputPath,
				Reason: "module is invalid",
			}
		}
		for index, source := range module.certificationSources {
			if !source.Valid() || source.sourcePath == module.sourcePath ||
				index > 0 && module.certificationSources[index-1].sourcePath == source.sourcePath {
				return nil, &Error{
					Operation: "validate staged module", Subject: module.outputPath,
					Reason: "certification source is invalid",
				}
			}
		}
		for index, name := range module.exports {
			if name == "" || index > 0 && module.exports[index-1] == name {
				return nil, &Error{
					Operation: "validate staged module", Subject: module.outputPath,
					Reason: "module exports are invalid",
				}
			}
		}
		if _, duplicate := seen[module.outputPath]; duplicate {
			return nil, &Error{
				Operation: "validate staged module", Subject: module.outputPath,
				Reason: "module is duplicated",
			}
		}
		seen[module.outputPath] = module
	}
	return seen, nil
}

func validateStagedCallables(
	callables []StagedCallable,
	generated map[string]struct{},
	modules map[string]StagedModule,
) error {
	seen := make(map[string]struct{}, len(callables))
	used := make(map[string]map[string]struct{}, len(modules))
	for _, callable := range callables {
		module, moduleOK := modules[callable.implementationOutput]
		_, generatedOK := generated[callable.generated.outputPath]
		if callable.sourceIdentity == "" || callable.sourceSignature == "" ||
			!validSHA256(callable.sourceBodyDigest) ||
			!callable.variant.Valid() || !callable.generated.Valid() ||
			!moduleOK || !generatedOK ||
			!slices.Contains(module.exports, callable.implementationExport) {
			return &Error{
				Operation: "validate staged callable", Subject: callable.sourceIdentity,
				Reason: "callable is invalid",
			}
		}
		if _, duplicate := seen[callable.sourceIdentity]; duplicate {
			return &Error{
				Operation: "validate staged callable", Subject: callable.sourceIdentity,
				Reason: "callable is duplicated",
			}
		}
		seen[callable.sourceIdentity] = struct{}{}
		if used[callable.implementationOutput] == nil {
			used[callable.implementationOutput] = make(map[string]struct{})
		}
		used[callable.implementationOutput][callable.implementationExport] = struct{}{}
	}
	for outputPath, module := range modules {
		if len(used[outputPath]) != len(module.exports) {
			return &Error{
				Operation: "validate staged callable", Subject: outputPath,
				Reason: "module export was not consumed exactly once",
			}
		}
	}
	return nil
}

func materializeGeneratedTargets(
	client *tsgo.Client,
	root string,
	targets []StagedTarget,
) error {
	for _, target := range targets {
		payload, err := os.ReadFile(target.protocolPath)
		if err != nil {
			return contractError("read staged target", target.protocolPath, err)
		}
		if sha256.Sum256(payload) != target.protocolHash {
			return &Error{
				Operation: "verify staged target", Subject: target.outputPath,
				Reason: "protocol payload digest changed",
			}
		}
		printed, err := client.PrintEncodedSourceFile(payload, tsgo.PrintOptions{})
		if err != nil {
			return err
		}
		path := filepath.Join(root, filepath.FromSlash(target.outputPath))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return contractError("create staged target directory", path, err)
		}
		if err := os.WriteFile(path, []byte(printed), 0o644); err != nil {
			return contractError("write staged target", path, err)
		}
	}
	return nil
}

func materializeImplementationModules(root string, modules []StagedModule) error {
	for _, module := range modules {
		payload, err := os.ReadFile(module.sourcePath)
		if err != nil {
			return contractError("read implementation source", module.sourcePath, err)
		}
		digest := sha256.Sum256(payload)
		if hex.EncodeToString(digest[:]) != module.sourceDigest {
			return &Error{
				Operation: "verify implementation source", Subject: module.outputPath,
				Reason: "source digest changed",
			}
		}
		path := filepath.Join(root, filepath.FromSlash(module.outputPath))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return contractError("create implementation directory", path, err)
		}
		if err := os.WriteFile(path, payload, 0o644); err != nil {
			return contractError("write implementation source", path, err)
		}
	}
	return nil
}

func materializeCertificationSources(
	root string,
	modules []StagedModule,
) ([]string, error) {
	byDigest := make(map[string][]byte)
	for _, module := range modules {
		for _, source := range module.certificationSources {
			payload, err := os.ReadFile(source.sourcePath)
			if err != nil {
				return nil, contractError("read certification source", source.sourcePath, err)
			}
			digest := sha256.Sum256(payload)
			if hex.EncodeToString(digest[:]) != source.sourceDigest {
				return nil, &Error{
					Operation: "verify certification source", Subject: source.sourcePath,
					Reason: "source digest changed",
				}
			}
			byDigest[source.sourceDigest] = payload
		}
	}
	if len(byDigest) == 0 {
		return nil, nil
	}
	digests := make([]string, 0, len(byDigest))
	for digest := range byDigest {
		digests = append(digests, digest)
	}
	sort.Strings(digests)
	directory := filepath.Join(root, "certification")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, contractError("create certification directory", directory, err)
	}
	paths := make([]string, len(digests))
	for index, digest := range digests {
		payload := byDigest[digest]
		path := filepath.Join(directory, fmt.Sprintf("%06d.d.ts", index))
		if err := os.WriteFile(path, payload, 0o644); err != nil {
			return nil, contractError("write certification source", path, err)
		}
		paths[index] = path
	}
	return paths, nil
}

func rejectCallableImplementationSource(
	project *tsgo.ProjectInspection,
	path string,
	subject string,
	role tsgo.CallableImplementationSourceRole,
) error {
	violations, err := project.CallableImplementationSourceViolations(path, role)
	if err != nil {
		return err
	}
	if len(violations) == 0 {
		return nil
	}
	return &Error{
		Operation: "verify staged module",
		Subject:   subject,
		Reason: fmt.Sprintf(
			"authored contract violates callable source policy %v",
			violations,
		),
	}
}

func inspectModuleExports(
	project *tsgo.ProjectInspection,
	root string,
	modules []StagedModule,
) (map[string]map[string]tsgo.ProjectExport, error) {
	result := make(map[string]map[string]tsgo.ProjectExport, len(modules))
	for _, module := range modules {
		exports, err := project.Exports(filepath.Join(root, filepath.FromSlash(module.outputPath)))
		if err != nil {
			return nil, err
		}
		byName := make(map[string]tsgo.ProjectExport, len(exports))
		actual := make([]string, len(exports))
		for index, export := range exports {
			actual[index] = export.Name()
			byName[export.Name()] = export
		}
		sort.Strings(actual)
		if !slices.Equal(actual, module.exports) {
			return nil, &Error{
				Operation: "join implementation exports", Subject: module.outputPath,
				Reason: fmt.Sprintf("exports %v differ from contract %v", actual, module.exports),
			}
		}
		result[module.outputPath] = byName
	}
	return result, nil
}

func verifyCallableTarget(
	project *tsgo.ProjectInspection,
	callable StagedCallable,
	manual tsgo.ProjectExport,
	exports []tsgo.ProjectExport,
) error {
	if manual.Name() != callable.implementationExport {
		return &Error{
			Operation: "join callable", Subject: callable.sourceIdentity,
			Reason: "implementation export is absent",
		}
	}
	if callable.generated.kind == GeneratedTargetModuleFunction {
		generated, ok := findProjectExport(exports, callable.generated.export)
		if !ok {
			return &Error{
				Operation: "join callable", Subject: callable.sourceIdentity,
				Reason: "generated module function is absent",
			}
		}
		return compareCallableExports(project, callable.sourceIdentity, generated, manual)
	}
	classExport, ok := findProjectExport(exports, callable.generated.className)
	if !ok {
		return &Error{
			Operation: "join callable", Subject: callable.sourceIdentity,
			Reason: "generated receiver class is absent",
		}
	}
	member, ok := classExport.ValueMember(callable.generated.memberName)
	if !ok || !member.Visible() {
		return &Error{
			Operation: "join callable", Subject: callable.sourceIdentity,
			Reason: "generated static method is absent or non-public",
		}
	}
	return compareCallableMember(project, callable.sourceIdentity, member, manual)
}

func compareCallableExports(
	project *tsgo.ProjectInspection,
	subject string,
	generated tsgo.ProjectExport,
	manual tsgo.ProjectExport,
) error {
	generatedTypes, err := project.CallableTypeParameterCount(generated)
	if err != nil {
		return err
	}
	manualTypes, err := project.CallableTypeParameterCount(manual)
	if err != nil {
		return err
	}
	generatedCount, err := project.CallableParameterCount(generated)
	if err != nil {
		return err
	}
	manualCount, err := project.CallableParameterCount(manual)
	if err != nil {
		return err
	}
	equivalent, err := project.CallableTypesEquivalent(generated, manual)
	if err != nil {
		return err
	}
	return compareCallableComponents(
		subject,
		generatedTypes,
		manualTypes,
		generatedCount,
		manualCount,
		equivalent,
		generated.TypeString(),
		manual.TypeString(),
	)
}

func compareCallableMember(
	project *tsgo.ProjectInspection,
	subject string,
	generated tsgo.ProjectMember,
	manual tsgo.ProjectExport,
) error {
	generatedTypes, err := project.CallableTypeParameterCount(generated)
	if err != nil {
		return err
	}
	manualTypes, err := project.CallableTypeParameterCount(manual)
	if err != nil {
		return err
	}
	generatedCount, err := project.CallableParameterCount(generated)
	if err != nil {
		return err
	}
	manualCount, err := project.CallableParameterCount(manual)
	if err != nil {
		return err
	}
	equivalent, err := project.CallableTypesEquivalent(generated, manual)
	if err != nil {
		return err
	}
	return compareCallableComponents(
		subject,
		generatedTypes,
		manualTypes,
		generatedCount,
		manualCount,
		equivalent,
		generated.TypeString(),
		manual.TypeString(),
	)
}

func compareCallableComponents(
	subject string,
	generatedTypes int,
	manualTypes int,
	generatedCount int,
	manualCount int,
	equivalent bool,
	generatedType string,
	manualType string,
) error {
	if generatedTypes != manualTypes {
		return callableTypeError(subject, fmt.Sprint(generatedTypes), fmt.Sprint(manualTypes))
	}
	if generatedCount != manualCount {
		return callableTypeError(subject, fmt.Sprint(generatedCount), fmt.Sprint(manualCount))
	}
	if !equivalent {
		return callableTypeError(subject, generatedType, manualType)
	}
	return nil
}

func findProjectExport(
	exports []tsgo.ProjectExport,
	name string,
) (tsgo.ProjectExport, bool) {
	for _, export := range exports {
		if export.Name() == name {
			return export, true
		}
	}
	return tsgo.ProjectExport{}, false
}

func callableTypeError(subject string, generated string, manual string) error {
	return &Error{
		Operation: "join callable type", Subject: subject,
		Reason: fmt.Sprintf("generated %q differs from implementation %q", generated, manual),
	}
}
