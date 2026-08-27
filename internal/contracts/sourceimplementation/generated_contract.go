package sourceimplementation

import (
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

func (t Target) OutputPath() string          { return t.outputPath }
func (t Target) SourceFile() tsgo.SourceFile { return t.sourceFile }

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

type ContractPackage struct {
	packagePath  string
	assemblyPath string
	exports      []string
}

func NewContractPackage(
	packagePath string,
	assemblyPath string,
	exports []string,
) (ContractPackage, error) {
	selected := slices.Clone(exports)
	if packagePath == "" || !validTargetPath(assemblyPath) || len(selected) == 0 {
		return ContractPackage{}, &Error{
			Operation: "join generated contract",
			Reason:    "package contract is invalid",
		}
	}
	sort.Strings(selected)
	for index, name := range selected {
		if name == "" || index > 0 && selected[index-1] == name {
			return ContractPackage{}, &Error{
				Operation: "join generated contract",
				Reason:    "package export identities are invalid",
			}
		}
	}
	return ContractPackage{
		packagePath:  packagePath,
		assemblyPath: assemblyPath,
		exports:      selected,
	}, nil
}

func (p ContractPackage) PackagePath() string  { return p.packagePath }
func (p ContractPackage) AssemblyPath() string { return p.assemblyPath }
func (p ContractPackage) Exports() []string    { return slices.Clone(p.exports) }

type GeneratedContractPlan struct {
	generated []Target
	packages  []ContractPackage
}

func (p GeneratedContractPlan) Valid() bool {
	if len(p.generated) == 0 || len(p.packages) == 0 {
		return false
	}
	if _, err := targetPaths(p.generated); err != nil {
		return false
	}
	seen := make(map[string]struct{}, len(p.packages))
	for _, selected := range p.packages {
		if !validContractPackage(selected) {
			return false
		}
		if _, duplicate := seen[selected.packagePath]; duplicate {
			return false
		}
		seen[selected.packagePath] = struct{}{}
	}
	return true
}

func (p GeneratedContractPlan) Generated() []Target {
	return slices.Clone(p.generated)
}

func (p GeneratedContractPlan) Packages() []ContractPackage {
	result := make([]ContractPackage, len(p.packages))
	for index, selected := range p.packages {
		result[index] = selected
		result[index].exports = slices.Clone(selected.exports)
	}
	return result
}

func (c *Certificate) PlanGeneratedContracts(
	generated []Target,
	installed []Target,
	packages []PackageTarget,
) (GeneratedContractPlan, error) {
	if !c.Valid() || len(generated) == 0 || len(installed) == 0 ||
		len(packages) != len(c.byPath) {
		return GeneratedContractPlan{}, &Error{
			Operation: "join generated contract",
			Reason:    "contract evidence is incomplete",
		}
	}
	generatedPaths, err := targetPaths(generated)
	if err != nil {
		return GeneratedContractPlan{}, err
	}
	installedPaths, err := targetPaths(installed)
	if err != nil {
		return GeneratedContractPlan{}, err
	}
	contractPackages := make([]ContractPackage, 0, len(packages))
	seenPackages := make(map[string]struct{}, len(packages))
	for _, selected := range packages {
		implementation, ok := c.byPath[selected.packagePath]
		if !ok || !validTargetPath(selected.assemblyPath) {
			return GeneratedContractPlan{}, &Error{
				Operation: "join generated contract",
				Subject:   selected.packagePath,
				Reason:    "implementation owner is absent",
			}
		}
		if _, duplicate := seenPackages[selected.packagePath]; duplicate {
			return GeneratedContractPlan{}, &Error{
				Operation: "join generated contract",
				Subject:   selected.packagePath,
				Reason:    "package contract is duplicated",
			}
		}
		if _, ok := generatedPaths[selected.assemblyPath]; !ok {
			return GeneratedContractPlan{}, &Error{
				Operation: "join generated contract",
				Subject:   selected.packagePath,
				Reason:    "ordinary generated package assembly is absent",
			}
		}
		if _, ok := installedPaths[selected.assemblyPath]; !ok {
			return GeneratedContractPlan{}, &Error{
				Operation: "join generated contract",
				Subject:   selected.packagePath,
				Reason:    "installed package assembly is absent",
			}
		}
		exports := implementation.Exports()
		exportNames := make([]string, len(exports))
		for index, selectedExport := range exports {
			exportNames[index] = selectedExport.Name()
		}
		contractPackage, err := NewContractPackage(
			selected.packagePath,
			selected.assemblyPath,
			exportNames,
		)
		if err != nil {
			return GeneratedContractPlan{}, err
		}
		seenPackages[selected.packagePath] = struct{}{}
		contractPackages = append(contractPackages, contractPackage)
	}
	sort.Slice(contractPackages, func(left, right int) bool {
		return contractPackages[left].packagePath < contractPackages[right].packagePath
	})
	return GeneratedContractPlan{
		generated: slices.Clone(generated),
		packages:  contractPackages,
	}, nil
}

func targetPaths(targets []Target) (map[string]struct{}, error) {
	result := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		if !validTargetPath(target.outputPath) || target.sourceFile == nil {
			return nil, &Error{
				Operation: "join generated contract",
				Reason:    "target set contains an invalid artifact",
			}
		}
		if _, duplicate := result[target.outputPath]; duplicate {
			return nil, &Error{
				Operation: "join generated contract",
				Subject:   target.outputPath,
				Reason:    "target artifact is duplicated",
			}
		}
		result[target.outputPath] = struct{}{}
	}
	return result, nil
}

func validContractPackage(selected ContractPackage) bool {
	if selected.packagePath == "" || !validTargetPath(selected.assemblyPath) ||
		len(selected.exports) == 0 || !sort.StringsAreSorted(selected.exports) {
		return false
	}
	for index, name := range selected.exports {
		if name == "" || index > 0 && selected.exports[index-1] == name {
			return false
		}
	}
	return true
}

func validTargetPath(path string) bool {
	return path != "" && filepath.IsLocal(filepath.FromSlash(path)) &&
		filepath.ToSlash(filepath.Clean(filepath.FromSlash(path))) == path &&
		!strings.HasPrefix(path, ".")
}
