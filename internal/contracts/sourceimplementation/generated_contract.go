package sourceimplementation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Target struct {
	outputPath string
	sourceFile tsgo.SourceFile
}

func NewTarget(outputPath string, sourceFile tsgo.SourceFile) (Target, error) {
	if !validTargetPath(outputPath) || sourceFile == nil {
		return Target{}, &Error{
			Operation: "join generated contract",
			Reason:    "target artifact is invalid",
		}
	}
	return Target{outputPath: outputPath, sourceFile: sourceFile}, nil
}

type PackageTarget struct {
	packagePath  string
	assemblyPath string
}

func NewPackageTarget(
	packagePath string,
	assemblyPath string,
) (PackageTarget, error) {
	if packagePath == "" || !validTargetPath(assemblyPath) {
		return PackageTarget{}, &Error{
			Operation: "join generated contract",
			Reason:    "package target is invalid",
		}
	}
	return PackageTarget{
		packagePath:  packagePath,
		assemblyPath: assemblyPath,
	}, nil
}

func (c *Certificate) VerifyGeneratedContracts(
	generated []Target,
	installed []Target,
	packages []PackageTarget,
) (resultErr error) {
	if !c.Valid() || c.repository == "" || c.scratch == "" ||
		len(generated) == 0 || len(installed) == 0 || len(packages) == 0 {
		return &Error{
			Operation: "join generated contract",
			Reason:    "contract evidence is incomplete",
		}
	}
	digest, err := targetSetDigest(generated, installed, packages)
	if err != nil {
		return err
	}
	root := filepath.Join(c.scratch, "generated-contract-"+digest)
	generatedRoot := filepath.Join(root, "generated")
	installedRoot := filepath.Join(root, "installed")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	client, err := tsgo.StartClient(c.repository, root)
	if err != nil {
		return err
	}
	defer func() {
		if err := client.Close(); resultErr == nil && err != nil {
			resultErr = err
		}
	}()
	generatedConfig, err := materializeTargetSet(
		client,
		c.repository,
		generatedRoot,
		generated,
	)
	if err != nil {
		return err
	}
	installedConfig, err := materializeTargetSet(
		client,
		c.repository,
		installedRoot,
		installed,
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
	for _, selected := range packages {
		implementation, ok := c.byPath[selected.packagePath]
		if !ok {
			return &Error{
				Operation: "join generated contract",
				Subject:   selected.packagePath,
				Reason:    "implementation owner is absent",
			}
		}
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
		if err := exactJoinPackageExports(
			implementation,
			generatedExports,
			installedExports,
		); err != nil {
			return err
		}
	}
	return nil
}

func exactJoinPackageExports(
	implementation Implementation,
	generated []tsgo.ProjectExport,
	installed []tsgo.ProjectExport,
) error {
	expected := implementation.Exports()
	expectedNames := make([]string, len(expected))
	for index, export := range expected {
		expectedNames[index] = export.Name()
	}
	generatedNames := projectExportNames(generated)
	installedNames := projectExportNames(installed)
	if !slices.Equal(generatedNames, expectedNames) ||
		!slices.Equal(installedNames, expectedNames) {
		return &Error{
			Operation: "join generated contract",
			Subject:   implementation.PackagePath(),
			Reason: fmt.Sprintf(
				"generated exports %v and installed exports %v differ from %v",
				generatedNames,
				installedNames,
				expectedNames,
			),
		}
	}
	return nil
}

func projectExportNames(selected []tsgo.ProjectExport) []string {
	result := make([]string, len(selected))
	for index, export := range selected {
		result[index] = export.Name()
	}
	sort.Strings(result)
	return result
}

func materializeTargetSet(
	client *tsgo.Client,
	distributionRoot string,
	root string,
	targets []Target,
) (string, error) {
	for _, target := range targets {
		if !validTargetPath(target.outputPath) || target.sourceFile == nil {
			return "", &Error{
				Operation: "join generated contract",
				Reason:    "target set contains an invalid artifact",
			}
		}
		printed, err := client.PrintNode(target.sourceFile, tsgo.PrintOptions{})
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

func targetSetDigest(
	generated []Target,
	installed []Target,
	packages []PackageTarget,
) (string, error) {
	digest := sha256.New()
	for _, set := range [][]Target{generated, installed} {
		ordered := slices.Clone(set)
		sort.Slice(ordered, func(left, right int) bool {
			return ordered[left].outputPath < ordered[right].outputPath
		})
		for _, target := range ordered {
			encoded, err := tsgo.EncodeSourceFile(target.sourceFile)
			if err != nil {
				return "", err
			}
			digest.Write([]byte(target.outputPath))
			digest.Write([]byte{0})
			digest.Write(encoded)
		}
		digest.Write([]byte{0xff})
	}
	for _, selected := range packages {
		digest.Write([]byte(selected.packagePath))
		digest.Write([]byte{0})
		digest.Write([]byte(selected.assemblyPath))
		digest.Write([]byte{0})
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

func validTargetPath(path string) bool {
	return path != "" && filepath.IsLocal(filepath.FromSlash(path)) &&
		filepath.ToSlash(filepath.Clean(filepath.FromSlash(path))) == path &&
		!strings.HasPrefix(path, ".")
}
