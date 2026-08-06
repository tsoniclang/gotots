package certify

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
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
	GoBinary            string
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
	goBinary            string
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

type goEnvironment struct {
	Root    string `json:"GOROOT"`
	Version string `json:"GOVERSION"`
}

func resolveConfig(source Config) (resolvedConfig, error) {
	result := resolvedConfig{
		buildProfile:     source.BuildProfile,
		backend:          source.Backend,
		minimumGoVersion: source.MinimumGoVersion,
		maximumGoVersion: source.MaximumGoVersion,
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
	if source.GoBinary == "" {
		return resolvedConfig{}, certifyError("configure", "Go binary", "path is empty")
	}
	result.goBinary, err = exec.LookPath(source.GoBinary)
	if err != nil {
		return resolvedConfig{}, certifyError("configure", source.GoBinary, err.Error())
	}
	result.goBinary, err = filepath.Abs(result.goBinary)
	if err != nil {
		return resolvedConfig{}, certifyError("configure", source.GoBinary, err.Error())
	}
	if result.backend == "" ||
		!result.buildProfile.Valid() ||
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
	command := exec.Command(
		config.goBinary,
		"env",
		"-json",
		"GOROOT",
		"GOVERSION",
	)
	command.Dir = config.repositoryRoot
	command.Env = environmentcontract.HostEnvironment()
	payload, err := command.Output()
	if err != nil {
		return toolchain{}, commandError("inspect toolchain", config.goBinary, err)
	}
	var selected goEnvironment
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&selected); err != nil {
		return toolchain{}, certifyError("inspect toolchain", config.goBinary, err.Error())
	}
	if selected.Root == "" || selected.Version == "" {
		return toolchain{}, certifyError(
			"inspect toolchain",
			config.goBinary,
			"identity is incomplete",
		)
	}
	if selected.Version != config.buildProfile.ToolchainVersion() {
		return toolchain{}, certifyError(
			"inspect toolchain",
			config.goBinary,
			"binary version does not match the selected build profile",
		)
	}
	key, err := environmentcontract.ToolchainKey(config.buildProfile)
	if err != nil {
		return toolchain{}, err
	}
	return toolchain{
		root:    filepath.Clean(selected.Root),
		version: selected.Version,
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
