package certify

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
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
