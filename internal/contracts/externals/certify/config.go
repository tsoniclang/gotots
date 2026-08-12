package certify

import (
	"fmt"
	"os"
	"path/filepath"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"github.com/tsoniclang/gotots/internal/toolchain"
)

type Config struct {
	RepositoryRoot              string
	ProviderRoot                string
	ManifestPath                string
	BindingMapPath              string
	TSConfigPath                string
	StandardLibraryManifestPath string
	StandardLibraryRuntimePath  string
	BuildProfile                environmentcontract.BuildProfile
	Backend                     string
	GoTool                      toolchain.Go
	TSGoTool                    tsgo.Tool
}

type resolvedConfig struct {
	repositoryRoot              string
	providerRoot                string
	manifestPath                string
	bindingMapPath              string
	tsConfigPath                string
	standardLibraryManifestPath string
	standardLibraryRuntimePath  string
	buildProfile                environmentcontract.BuildProfile
	backend                     string
	goTool                      toolchain.Go
	tsgoTool                    tsgo.Tool
}

type Error struct {
	Operation string
	Subject   string
	Reason    string
}

func (e *Error) Error() string {
	if e.Subject == "" {
		return fmt.Sprintf("certify external provider %s: %s", e.Operation, e.Reason)
	}
	return fmt.Sprintf(
		"certify external provider %s %q: %s",
		e.Operation,
		e.Subject,
		e.Reason,
	)
}

func resolveConfig(source Config) (resolvedConfig, error) {
	result := resolvedConfig{
		buildProfile: source.BuildProfile,
		backend:      source.Backend,
		goTool:       source.GoTool,
		tsgoTool:     source.TSGoTool,
	}
	paths := []struct {
		name   string
		source string
		target *string
	}{
		{"repository root", source.RepositoryRoot, &result.repositoryRoot},
		{"provider root", source.ProviderRoot, &result.providerRoot},
		{"manifest", source.ManifestPath, &result.manifestPath},
		{"binding map", source.BindingMapPath, &result.bindingMapPath},
		{"tsconfig", source.TSConfigPath, &result.tsConfigPath},
		{
			"standard-library manifest",
			source.StandardLibraryManifestPath,
			&result.standardLibraryManifestPath,
		},
		{
			"standard-library runtime contract",
			source.StandardLibraryRuntimePath,
			&result.standardLibraryRuntimePath,
		},
	}
	for _, selected := range paths {
		if selected.source == "" {
			return resolvedConfig{}, certifyError(
				"configure",
				selected.name,
				"path is empty",
			)
		}
		absolute, err := filepath.Abs(selected.source)
		if err != nil {
			return resolvedConfig{}, certifyError(
				"configure",
				selected.name,
				err.Error(),
			)
		}
		*selected.target = absolute
	}
	if !result.buildProfile.Valid() || result.backend == "" ||
		!result.goTool.Valid() || !result.tsgoTool.Valid() ||
		result.goTool.Version() != result.buildProfile.ToolchainVersion() {
		return resolvedConfig{}, certifyError(
			"configure",
			"provider profile",
			"build or target profile is invalid",
		)
	}
	for _, directory := range []string{result.repositoryRoot, result.providerRoot} {
		info, err := os.Stat(directory)
		if err != nil || !info.IsDir() {
			if err == nil {
				err = fmt.Errorf("not a directory")
			}
			return resolvedConfig{}, certifyError(
				"configure",
				directory,
				err.Error(),
			)
		}
	}
	return result, nil
}

func certifyError(operation string, subject string, reason string) error {
	return &Error{Operation: operation, Subject: subject, Reason: reason}
}
