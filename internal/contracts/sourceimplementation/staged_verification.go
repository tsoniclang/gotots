package sourceimplementation

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type StagedTarget struct {
	outputPath   string
	protocolPath string
	protocolHash [sha256.Size]byte
}

func NewStagedTarget(
	outputPath string,
	protocolPath string,
	protocolHash [sha256.Size]byte,
) (StagedTarget, error) {
	if !validTargetPath(outputPath) || !filepath.IsAbs(protocolPath) ||
		protocolHash == ([sha256.Size]byte{}) {
		return StagedTarget{}, &Error{
			Operation: "verify staged generated contract",
			Reason:    "staged target is invalid",
		}
	}
	return StagedTarget{
		outputPath:   outputPath,
		protocolPath: filepath.Clean(protocolPath),
		protocolHash: protocolHash,
	}, nil
}

type StagedVerificationConfig struct {
	RepositoryRoot string
	ScratchRoot    string
	TSGoTool       tsgo.Tool
	Generated      []StagedTarget
	Installed      []StagedTarget
	Packages       []ContractPackage
}

func VerifyStagedGeneratedContracts(
	config StagedVerificationConfig,
) (resultErr error) {
	if !filepath.IsAbs(config.RepositoryRoot) || !filepath.IsAbs(config.ScratchRoot) ||
		!config.TSGoTool.Valid() || len(config.Generated) == 0 ||
		len(config.Installed) == 0 || len(config.Packages) == 0 {
		return &Error{
			Operation: "verify staged generated contract",
			Reason:    "verification configuration is incomplete",
		}
	}
	generatedPaths, err := validateStagedTargets(config.Generated)
	if err != nil {
		return err
	}
	installedPaths, err := validateStagedTargets(config.Installed)
	if err != nil {
		return err
	}
	seenPackages := make(map[string]struct{}, len(config.Packages))
	for _, selected := range config.Packages {
		if !validContractPackage(selected) {
			return &Error{
				Operation: "verify staged generated contract",
				Reason:    "package contract is invalid",
			}
		}
		if _, duplicate := seenPackages[selected.packagePath]; duplicate {
			return &Error{
				Operation: "verify staged generated contract",
				Subject:   selected.packagePath,
				Reason:    "package contract is duplicated",
			}
		}
		if _, ok := generatedPaths[selected.assemblyPath]; !ok {
			return &Error{
				Operation: "verify staged generated contract",
				Subject:   selected.packagePath,
				Reason:    "ordinary package assembly is absent",
			}
		}
		if _, ok := installedPaths[selected.assemblyPath]; !ok {
			return &Error{
				Operation: "verify staged generated contract",
				Subject:   selected.packagePath,
				Reason:    "installed package assembly is absent",
			}
		}
		seenPackages[selected.packagePath] = struct{}{}
	}
	if err := os.Mkdir(config.ScratchRoot, 0o700); err != nil {
		return implementationError(
			"create generated-contract scratch",
			config.ScratchRoot,
			err,
		)
	}
	client, err := tsgo.StartClientWithTool(config.TSGoTool, config.ScratchRoot)
	if err != nil {
		return err
	}
	closed := false
	defer func() {
		if !closed {
			if err := client.Close(); resultErr == nil && err != nil {
				resultErr = err
			}
		}
	}()
	generatedRoot := filepath.Join(config.ScratchRoot, "generated")
	installedRoot := filepath.Join(config.ScratchRoot, "installed")
	generatedConfig, err := materializeStagedTargetSet(
		client,
		config.RepositoryRoot,
		generatedRoot,
		config.Generated,
	)
	if err != nil {
		return err
	}
	installedConfig, err := materializeStagedTargetSet(
		client,
		config.RepositoryRoot,
		installedRoot,
		config.Installed,
	)
	if err != nil {
		return err
	}
	generatedProject, err := client.OpenProject(generatedConfig)
	if err != nil {
		return err
	}
	installedProject, err := client.OpenProject(installedConfig)
	if err != nil {
		return err
	}
	for _, selected := range config.Packages {
		generatedExports, err := generatedProject.Exports(filepath.Join(
			generatedRoot,
			filepath.FromSlash(selected.assemblyPath),
		))
		if err != nil {
			return err
		}
		installedExports, err := installedProject.Exports(filepath.Join(
			installedRoot,
			filepath.FromSlash(selected.assemblyPath),
		))
		if err != nil {
			return err
		}
		if err := joinPackageExportIdentities(
			selected,
			generatedExports,
			installedExports,
		); err != nil {
			return err
		}
	}
	if err := client.Close(); err != nil {
		return err
	}
	closed = true
	if err := os.RemoveAll(config.ScratchRoot); err != nil {
		return implementationError(
			"remove generated-contract scratch",
			config.ScratchRoot,
			err,
		)
	}
	return nil
}

func validateStagedTargets(
	targets []StagedTarget,
) (map[string]struct{}, error) {
	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		if !validTargetPath(target.outputPath) || !filepath.IsAbs(target.protocolPath) ||
			target.protocolHash == ([sha256.Size]byte{}) {
			return nil, &Error{
				Operation: "verify staged generated contract",
				Reason:    "staged target set contains an invalid artifact",
			}
		}
		if _, duplicate := seen[target.outputPath]; duplicate {
			return nil, &Error{
				Operation: "verify staged generated contract",
				Subject:   target.outputPath,
				Reason:    "staged target is duplicated",
			}
		}
		seen[target.outputPath] = struct{}{}
	}
	return seen, nil
}

func materializeStagedTargetSet(
	client *tsgo.Client,
	distributionRoot string,
	root string,
	targets []StagedTarget,
) (string, error) {
	for _, target := range targets {
		payload, err := os.ReadFile(target.protocolPath)
		if err != nil {
			return "", implementationError(
				"read staged generated contract",
				target.protocolPath,
				err,
			)
		}
		if sha256.Sum256(payload) != target.protocolHash {
			return "", &Error{
				Operation: "verify staged generated contract",
				Subject:   target.outputPath,
				Reason:    "protocol payload digest changed",
			}
		}
		printed, err := client.PrintEncodedSourceFile(payload, tsgo.PrintOptions{})
		if err != nil {
			return "", err
		}
		path := filepath.Join(root, filepath.FromSlash(target.outputPath))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(path, []byte(printed), 0o644); err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(
		filepath.Join(root, "package.json"),
		[]byte("{\"type\":\"module\"}\n"),
		0o644,
	); err != nil {
		return "", err
	}
	config := filepath.Join(root, "tsconfig.json")
	payload, err := tsgo.EncodeStrictProjectConfig(distributionRoot, root)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(config, payload, 0o644); err != nil {
		return "", err
	}
	return config, nil
}

func joinPackageExportIdentities(
	contract ContractPackage,
	generated []tsgo.ProjectExport,
	installed []tsgo.ProjectExport,
) error {
	generatedIdentities := projectExportIdentities(generated)
	installedIdentities := projectExportIdentities(installed)
	if !slices.Equal(generatedIdentities, contract.exports) ||
		!slices.Equal(installedIdentities, contract.exports) {
		return &Error{
			Operation: "join generated contract",
			Subject:   contract.packagePath,
			Reason: fmt.Sprintf(
				"generated export identities %v and installed export identities %v differ from %v",
				generatedIdentities,
				installedIdentities,
				contract.exports,
			),
		}
	}
	return nil
}

func projectExportIdentities(selected []tsgo.ProjectExport) []string {
	result := make([]string, len(selected))
	for index, export := range selected {
		result[index] = export.Name()
	}
	sort.Strings(result)
	return result
}
