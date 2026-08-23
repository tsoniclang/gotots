package certify

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib"
	runtimecontract "github.com/tsoniclang/gotots/internal/contracts/runtime"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	gotool "github.com/tsoniclang/gotots/internal/toolchain"
)

type Config struct {
	RepositoryRoot      string
	ProviderRoot        string
	ManifestPath        string
	ModuleMapPath       string
	FacetMapPath        string
	RuntimeContractPath string
	TSConfigPath        string
	ScratchDirectory    string
	GoTool              gotool.Go
	TSGoTool            tsgo.Tool
	BuildProfile        environmentcontract.BuildProfile
	Backend             string
	MinimumGoVersion    string
	MaximumGoVersion    string
}

type resolvedConfig struct {
	repositoryRoot      string
	providerRoot        string
	manifestPath        string
	moduleMapPath       string
	facetMapPath        string
	runtimeContractPath string
	tsConfigPath        string
	scratchDirectory    string
	goTool              gotool.Go
	tsgoTool            tsgo.Tool
	buildProfile        environmentcontract.BuildProfile
	backend             string
	minimumGoVersion    string
	maximumGoVersion    string
}

type Error struct {
	Operation string
	Subject   string
	Reason    string
}

func (e *Error) Error() string {
	if e.Subject == "" {
		return fmt.Sprintf("certify gostdlib %s: %s", e.Operation, e.Reason)
	}
	return fmt.Sprintf(
		"certify gostdlib %s %q: %s",
		e.Operation,
		e.Subject,
		e.Reason,
	)
}

type toolchain struct {
	root    string
	version string
	profile environmentcontract.BuildProfile
	key     string
}

func resolveConfig(source Config) (resolvedConfig, error) {
	result := resolvedConfig{
		buildProfile:     source.BuildProfile,
		backend:          source.Backend,
		minimumGoVersion: source.MinimumGoVersion,
		maximumGoVersion: source.MaximumGoVersion,
		goTool:           source.GoTool,
		tsgoTool:         source.TSGoTool,
	}
	var err error
	for name, value := range map[string]*string{
		"repository root":  &result.repositoryRoot,
		"provider root":    &result.providerRoot,
		"manifest":         &result.manifestPath,
		"module map":       &result.moduleMapPath,
		"facet map":        &result.facetMapPath,
		"runtime contract": &result.runtimeContractPath,
		"tsconfig":         &result.tsConfigPath,
		"scratch":          &result.scratchDirectory,
	} {
		selected := ""
		switch name {
		case "repository root":
			selected = source.RepositoryRoot
		case "provider root":
			selected = source.ProviderRoot
		case "manifest":
			selected = source.ManifestPath
		case "module map":
			selected = source.ModuleMapPath
		case "facet map":
			selected = source.FacetMapPath
		case "runtime contract":
			selected = source.RuntimeContractPath
		case "tsconfig":
			selected = source.TSConfigPath
		case "scratch":
			selected = source.ScratchDirectory
		}
		if selected == "" {
			return resolvedConfig{}, certifyError("configure", name, "path is empty")
		}
		*value, err = filepath.Abs(selected)
		if err != nil {
			return resolvedConfig{}, certifyError("configure", name, err.Error())
		}
	}
	if result.backend == "" ||
		!result.buildProfile.Valid() ||
		!result.goTool.Valid() || !result.tsgoTool.Valid() ||
		result.goTool.Version() != result.buildProfile.ToolchainVersion() ||
		!strings.HasPrefix(result.minimumGoVersion, "go") ||
		!strings.HasPrefix(result.maximumGoVersion, "go") {
		return resolvedConfig{}, certifyError(
			"configure",
			"provider policy",
			"build profile, backend, or Go version bounds are invalid",
		)
	}
	for _, directory := range []string{
		result.repositoryRoot,
		result.providerRoot,
	} {
		info, statErr := os.Stat(directory)
		if statErr != nil || !info.IsDir() {
			if statErr == nil {
				statErr = fmt.Errorf("not a directory")
			}
			return resolvedConfig{}, certifyError("configure", directory, statErr.Error())
		}
	}
	if err := os.MkdirAll(result.scratchDirectory, 0o755); err != nil {
		return resolvedConfig{}, certifyError(
			"configure",
			result.scratchDirectory,
			err.Error(),
		)
	}
	return result, nil
}

func inspectToolchain(config resolvedConfig) (toolchain, error) {
	if !config.goTool.Valid() || config.goTool.Version() != config.buildProfile.ToolchainVersion() {
		return toolchain{}, certifyError(
			"inspect toolchain",
			config.goTool.Path(),
			"binary version does not match the selected build profile",
		)
	}
	key, err := environmentcontract.ToolchainKey(config.buildProfile)
	if err != nil {
		return toolchain{}, err
	}
	return toolchain{
		root:    config.goTool.Root(),
		version: config.goTool.Version(),
		profile: config.buildProfile,
		key:     key,
	}, nil
}

func certifyError(operation string, subject string, reason string) error {
	return &Error{Operation: operation, Subject: subject, Reason: reason}
}

func commandError(operation string, subject string, err error) error {
	reason := err.Error()
	if exit, ok := err.(*exec.ExitError); ok {
		if stderr := strings.TrimSpace(string(exit.Stderr)); stderr != "" {
			reason += ": " + stderr
		}
	}
	return certifyError(operation, subject, reason)
}

type Certificate struct {
	manifest     gostdlib.Manifest
	toolchainKey string
	providerRoot string
	runtime      runtimecontract.Requirements
}

func Verify(config Config) (*Certificate, error) {
	resolved, err := resolveConfig(config)
	if err != nil {
		return nil, err
	}
	checkedBytes, checked, err := readManifest(resolved.manifestPath)
	if err != nil {
		return nil, err
	}
	generated, err := Generate(config)
	if err != nil {
		return nil, err
	}
	if err := compareCanonical(checkedBytes, generated); err != nil {
		return nil, err
	}
	runtimeRequirements, runtimeDigest, err := readRuntimeContract(
		resolved.runtimeContractPath,
	)
	if err != nil {
		return nil, err
	}
	if runtimeDigest != checked.RuntimeDigest() {
		return nil, certifyError(
			"verify runtime contract",
			resolved.runtimeContractPath,
			"content digest does not match the checked manifest",
		)
	}
	selectedToolchain, err := inspectToolchain(resolved)
	if err != nil {
		return nil, err
	}
	return &Certificate{
		manifest:     checked,
		toolchainKey: selectedToolchain.key,
		providerRoot: resolved.providerRoot,
		runtime:      runtimeRequirements,
	}, nil
}

func (c *Certificate) Valid() bool {
	return c != nil && c.manifest.Digest() != "" &&
		c.toolchainKey != "" && c.providerRoot != "" && c.runtime.Valid()
}

func (c *Certificate) ManifestDigest() string {
	if c == nil {
		return ""
	}
	return c.manifest.Digest()
}

func (c *Certificate) ToolchainKey() string {
	if c == nil {
		return ""
	}
	return c.toolchainKey
}

func (c *Certificate) ProviderModules() []string {
	if !c.Valid() {
		return nil
	}
	seen := make(map[string]struct{})
	for _, module := range c.manifest.Modules() {
		seen[module.Specifier()] = struct{}{}
	}
	for _, module := range c.manifest.FacetModules() {
		seen[module.Specifier()] = struct{}{}
	}
	seen[c.ProviderScalarModule()] = struct{}{}
	seen[c.ProviderPointerModule()] = struct{}{}
	modules := make([]string, 0, len(seen))
	for module := range seen {
		modules = append(modules, module)
	}
	sort.Strings(modules)
	return modules
}

func (c *Certificate) ProviderScalarModule() string {
	if !c.Valid() {
		return ""
	}
	return strings.TrimSuffix(c.manifest.PackageName(), "/") +
		strings.TrimPrefix(c.runtime.ProviderScalarModule(), ".")
}

func (c *Certificate) ProviderPointerModule() string {
	if !c.Valid() {
		return ""
	}
	return strings.TrimSuffix(c.manifest.PackageName(), "/") +
		strings.TrimPrefix(c.runtime.ProviderPointerModule(), ".")
}

func (c *Certificate) RuntimeRequirements() (
	runtimecontract.Requirements,
	bool,
) {
	if !c.Valid() {
		return runtimecontract.Requirements{}, false
	}
	return c.runtime, true
}

func (c *Certificate) Binding(identity string) (gostdlib.Binding, bool) {
	if !c.Valid() {
		return gostdlib.Binding{}, false
	}
	return c.manifest.Binding(identity)
}

// FacetModules lists the certified facet modules.
func (c *Certificate) FacetModules() []gostdlib.FacetModule {
	if !c.Valid() {
		return nil
	}
	return c.manifest.FacetModules()
}

// Implementations lists the certified private implementation documents.
func (c *Certificate) Implementations() []gostdlib.ImplementationDocument {
	if !c.Valid() {
		return nil
	}
	return c.manifest.Implementations()
}

func (c *Certificate) Modules() []gostdlib.Module {
	if !c.Valid() {
		return nil
	}
	return c.manifest.Modules()
}

func (c *Certificate) Facet(
	sourceIdentity string,
	kind gostdlib.FacetKind,
	capability gostdlib.FacetCapability,
) (gostdlib.Facet, bool) {
	if !c.Valid() {
		return gostdlib.Facet{}, false
	}
	return c.manifest.Facet(sourceIdentity, kind, capability)
}

func (c *Certificate) ProviderRepresentation(
	module string,
	export string,
) (gostdlib.ProviderRepresentation, bool) {
	if !c.Valid() {
		return gostdlib.ProviderRepresentation{}, false
	}
	return c.manifest.ProviderRepresentation(module, export)
}

func (c *Certificate) ProviderInterface(
	sourceIdentity string,
) (gostdlib.ProviderInterfaceBinding, bool) {
	if !c.Valid() {
		return gostdlib.ProviderInterfaceBinding{}, false
	}
	return c.manifest.ProviderInterface(sourceIdentity)
}

func (c *Certificate) ProviderInterfaceCapabilities(
	baseSourceIdentity string,
) []gostdlib.ProviderInterfaceCapability {
	if !c.Valid() {
		return nil
	}
	return c.manifest.ProviderInterfaceCapabilities(baseSourceIdentity)
}

func (c *Certificate) ProviderCallableProfile(
	sourceIdentity string,
	profileKey string,
) (gostdlib.ProviderCallableProfile, bool) {
	if !c.Valid() {
		return gostdlib.ProviderCallableProfile{}, false
	}
	return c.manifest.ProviderCallableProfile(sourceIdentity, profileKey)
}

func (c *Certificate) ProviderCallableProfiles(
	sourceIdentity string,
) []gostdlib.ProviderCallableProfile {
	if !c.Valid() {
		return nil
	}
	return c.manifest.ProviderCallableProfiles(sourceIdentity)
}

func (c *Certificate) ProviderStatefulProfile(
	sourceIdentity string,
	profileKey string,
) (gostdlib.ProviderStatefulProfile, bool) {
	if !c.Valid() {
		return gostdlib.ProviderStatefulProfile{}, false
	}
	return c.manifest.ProviderStatefulProfile(sourceIdentity, profileKey)
}

func (c *Certificate) ProviderStatefulProfiles(
	sourceIdentity string,
) []gostdlib.ProviderStatefulProfile {
	if !c.Valid() {
		return nil
	}
	return c.manifest.ProviderStatefulProfiles(sourceIdentity)
}
