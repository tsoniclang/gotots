package config

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

	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"github.com/tsoniclang/gotots/internal/toolchain"
)

type document struct {
	SchemaVersion   int                     `json:"schemaVersion"`
	Distribution    distributionDocument    `json:"distribution"`
	Source          sourceDocument          `json:"source"`
	Go              goDocument              `json:"go"`
	Semantics       semanticsDocument       `json:"semantics"`
	Providers       providersDocument       `json:"providers"`
	Implementations implementationsDocument `json:"implementations"`
	Output          outputDocument          `json:"output"`
	Tools           toolsDocument           `json:"tools"`
}

type distributionDocument struct {
	Root *string `json:"root"`
}
type sourceDocument struct {
	Root    *string `json:"root"`
	Package *string `json:"package"`
	Mode    *string `json:"mode"`
}
type goDocument struct {
	GOOS   *string  `json:"goos"`
	GOARCH *string  `json:"goarch"`
	CGO    *bool    `json:"cgo"`
	Tags   []string `json:"tags"`
}
type semanticsDocument struct {
	Integers        *string `json:"integers"`
	EvaluationOrder *string `json:"evaluationOrder"`
}
type providersDocument struct {
	StandardLibrary *bool `json:"standardLibrary"`
	Externals       *bool `json:"externals"`
}
type implementationsDocument struct {
	Bundles []string `json:"bundles"`
}
type outputDocument struct {
	Directory *string `json:"directory"`
}
type toolsDocument struct {
	Go    *string `json:"go"`
	TSGo  *string `json:"tsgo"`
	Cache *string `json:"cache"`
}

func Load(request Request) (Project, error) {
	if request.ConfigPath == "" {
		return Project{}, projectError("select config", "", "path is empty")
	}
	configPath, err := filepath.Abs(request.ConfigPath)
	if err != nil {
		return Project{}, projectError("select config", request.ConfigPath, err.Error())
	}
	payload, err := os.ReadFile(configPath)
	if err != nil {
		return Project{}, projectError("read config", configPath, err.Error())
	}
	document, err := decode(payload)
	if err != nil {
		return Project{}, err
	}
	return resolve(configPath, document, request.Overrides)
}

func decode(payload []byte) (document, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var selected document
	if err := decoder.Decode(&selected); err != nil {
		return document{}, projectError("decode config", "", err.Error())
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return document{}, projectError("decode config", "", err.Error())
	}
	if selected.SchemaVersion != SchemaVersion {
		return document{}, projectError("decode config", "", "schema version is unsupported")
	}
	return selected, nil
}

func resolve(path string, selected document, overrides Overrides) (Project, error) {
	base := filepath.Dir(path)
	applyDefaults(&selected)
	applyOverrides(&selected, overrides)
	mode := RootMode(*selected.Source.Mode)
	if !mode.Valid() {
		return Project{}, projectError("validate config", "source.mode", "value is invalid")
	}
	distributionRoot, err := resolvePath(base, *selected.Distribution.Root)
	if err != nil {
		return Project{}, err
	}
	toolCacheRoot, err := resolvePath(base, *selected.Tools.Cache)
	if err != nil {
		return Project{}, err
	}
	goPath, err := resolveOptionalPath(base, *selected.Tools.Go)
	if err != nil {
		return Project{}, err
	}
	selectedGo, err := toolchain.ResolveGo(goPath, toolCacheRoot)
	if err != nil {
		return Project{}, projectError("resolve Go tool", goPath, err.Error())
	}
	defaultString(&selected.Go.GOOS, selectedGo.DefaultGOOS())
	defaultString(&selected.Go.GOARCH, selectedGo.DefaultGOARCH())
	build, err := loadBuildProfile(selected.Go, selectedGo.Version())
	if err != nil {
		return Project{}, err
	}
	if err := selectedGo.ValidateProfile(build); err != nil {
		return Project{}, projectError("validate config", "go.cgo", err.Error())
	}
	tsgoPath, err := resolveOptionalPath(base, *selected.Tools.TSGo)
	if err != nil {
		return Project{}, err
	}
	selectedTSGo, err := tsgo.ResolveTool(selectedGo, distributionRoot, tsgoPath)
	if err != nil {
		return Project{}, projectError("resolve TS-Go tool", tsgoPath, err.Error())
	}
	integer, err := parseInteger(*selected.Semantics.Integers)
	if err != nil {
		return Project{}, err
	}
	evaluation, err := parseEvaluation(*selected.Semantics.EvaluationOrder)
	if err != nil {
		return Project{}, err
	}
	sourceRoot, err := resolvePath(base, *selected.Source.Root)
	if err != nil {
		return Project{}, err
	}
	outputDirectory, err := resolvePath(base, *selected.Output.Directory)
	if err != nil {
		return Project{}, err
	}
	bundles, err := resolvePaths(base, selected.Implementations.Bundles)
	if err != nil {
		return Project{}, err
	}
	if *selected.Source.Package == "" {
		return Project{}, projectError("validate config", "source.package", "value is empty")
	}
	if *selected.Providers.Externals && !*selected.Providers.StandardLibrary {
		return Project{}, projectError(
			"validate config",
			"providers",
			"externals require the standard library provider",
		)
	}
	return Project{
		configPath:            path,
		distributionRoot:      distributionRoot,
		sourceRoot:            sourceRoot,
		packagePattern:        *selected.Source.Package,
		rootMode:              mode,
		buildProfile:          build,
		goTool:                selectedGo,
		tsgoTool:              selectedTSGo,
		toolCacheRoot:         toolCacheRoot,
		integer:               integer,
		evaluation:            evaluation,
		standardLibrary:       *selected.Providers.StandardLibrary,
		externals:             *selected.Providers.Externals,
		implementationBundles: bundles,
		outputDirectory:       outputDirectory,
	}, nil
}

func applyDefaults(selected *document) {
	defaultString(&selected.Distribution.Root, ".")
	defaultString(&selected.Source.Root, ".")
	defaultString(&selected.Source.Package, ".")
	defaultString(&selected.Source.Mode, string(RootModeMain))
	if selected.Go.CGO == nil {
		selected.Go.CGO = boolPointer(false)
	}
	defaultString(&selected.Semantics.Integers, "number")
	defaultString(&selected.Semantics.EvaluationOrder, "direct")
	if selected.Providers.StandardLibrary == nil {
		selected.Providers.StandardLibrary = boolPointer(false)
	}
	if selected.Providers.Externals == nil {
		selected.Providers.Externals = boolPointer(false)
	}
	defaultString(&selected.Output.Directory, "generated")
	defaultString(&selected.Tools.Go, "")
	defaultString(&selected.Tools.TSGo, "")
	defaultString(&selected.Tools.Cache, filepath.Join(".temp", "cache", "toolchain"))
}

func applyOverrides(selected *document, overrides Overrides) {
	applyString(overrides.DistributionRoot, &selected.Distribution.Root)
	applyString(overrides.GoExecutable, &selected.Tools.Go)
	applyString(overrides.TSGoExecutable, &selected.Tools.TSGo)
	applyString(overrides.ToolCacheRoot, &selected.Tools.Cache)
	applyString(overrides.SourceRoot, &selected.Source.Root)
	applyString(overrides.PackagePattern, &selected.Source.Package)
	applyString(overrides.RootMode, &selected.Source.Mode)
	applyString(overrides.GOOS, &selected.Go.GOOS)
	applyString(overrides.GOARCH, &selected.Go.GOARCH)
	if overrides.CGOEnabled != nil {
		selected.Go.CGO = overrides.CGOEnabled
	}
	if overrides.BuildTagsSet {
		selected.Go.Tags = slices.Clone(overrides.BuildTags)
	}
	applyString(overrides.IntegerRepresentation, &selected.Semantics.Integers)
	applyString(overrides.EvaluationOrder, &selected.Semantics.EvaluationOrder)
	if overrides.StandardLibrary != nil {
		selected.Providers.StandardLibrary = overrides.StandardLibrary
	}
	if overrides.Externals != nil {
		selected.Providers.Externals = overrides.Externals
	}
	if overrides.ImplementationSet {
		selected.Implementations.Bundles = slices.Clone(overrides.ImplementationBundles)
	}
	applyString(overrides.OutputDirectory, &selected.Output.Directory)
}

func (p Project) SemanticDigest(evidence EvidenceDigests) (string, error) {
	if evidence.Source == "" {
		return "", projectError("digest", "source", "evidence is absent")
	}
	if len(p.implementationBundles) != 0 && evidence.SourceImplementations == "" {
		return "", projectError("digest", "source implementations", "evidence is absent")
	}
	if p.standardLibrary && evidence.StandardLibrary == "" {
		return "", projectError("digest", "standard library", "evidence is absent")
	}
	if p.externals && evidence.Externals == "" {
		return "", projectError("digest", "externals", "evidence is absent")
	}
	document := struct {
		Schema          int      `json:"schema"`
		Package         string   `json:"package"`
		Mode            RootMode `json:"mode"`
		Toolchain       string   `json:"toolchain"`
		GoTool          string   `json:"goTool"`
		TSGoTool        string   `json:"tsgoTool"`
		GOOS            string   `json:"goos"`
		GOARCH          string   `json:"goarch"`
		CGO             bool     `json:"cgo"`
		Tags            []string `json:"tags"`
		Integer         string   `json:"integer"`
		Evaluation      string   `json:"evaluation"`
		Source          string   `json:"source"`
		Implementations string   `json:"implementations,omitempty"`
		StandardLibrary string   `json:"standardLibrary,omitempty"`
		Externals       string   `json:"externals,omitempty"`
	}{
		Schema: SchemaVersion, Package: p.packagePattern, Mode: p.rootMode,
		Toolchain: p.buildProfile.ToolchainVersion(), GOOS: p.buildProfile.GOOS(),
		GoTool: p.goTool.Identity().String(), TSGoTool: p.tsgoTool.Identity().String(),
		GOARCH: p.buildProfile.GOARCH(), CGO: p.buildProfile.CgoEnabled(), Tags: p.buildProfile.Tags(),
		Integer: p.integer.String(), Evaluation: p.evaluation.String(),
		Source: evidence.Source, Implementations: evidence.SourceImplementations,
		StandardLibrary: evidence.StandardLibrary, Externals: evidence.Externals,
	}
	payload, err := json.Marshal(document)
	if err != nil {
		return "", projectError("digest", "", err.Error())
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func resolvePath(base string, selected string) (string, error) {
	if selected == "" {
		return "", projectError("resolve path", "", "value is empty")
	}
	if !filepath.IsAbs(selected) {
		selected = filepath.Join(base, selected)
	}
	return filepath.Clean(selected), nil
}

func resolveOptionalPath(base string, selected string) (string, error) {
	if selected == "" {
		return "", nil
	}
	return resolvePath(base, selected)
}

func resolvePaths(base string, selected []string) ([]string, error) {
	result := make([]string, len(selected))
	for index, value := range selected {
		resolved, err := resolvePath(base, value)
		if err != nil {
			return nil, err
		}
		result[index] = resolved
	}
	slices.Sort(result)
	for index := 1; index < len(result); index++ {
		if result[index] == result[index-1] {
			return nil, projectError("validate config", "implementations.bundles", "path is duplicated")
		}
	}
	return result, nil
}

func boolPointer(value bool) *bool       { return &value }
func stringPointer(value string) *string { return &value }
func defaultString(target **string, value string) {
	if *target == nil {
		*target = stringPointer(value)
	}
}
func applyString(source *string, target **string) {
	if source != nil {
		*target = source
	}
}

func projectError(operation string, subject string, reason string) error {
	return &Error{Operation: operation, Subject: subject, Reason: reason}
}
