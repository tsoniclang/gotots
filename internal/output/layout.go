package output

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/tsoniclang/gotots/internal/load"
)

const (
	ProgramInitializationPath  = "program.ts"
	RuntimePackageName         = "@gotots/runtime"
	RuntimePackageRootPath     = "runtime"
	RuntimePackageManifestPath = "runtime/package.json"
	ScalarSupportPath          = "runtime/scalars.ts"
	AnonymousStructSupportPath = "support/anonymous-structs.ts"
	InterfaceMethodSupportPath = "support/interface-methods.ts"
	InterfaceTypeSupportPath   = "support/interface-types.ts"
)

func EnvironmentContractPath(
	sourcePackage *load.Package,
) (string, error) {
	if sourcePackage == nil {
		return "", &PathError{Reason: "environment package is nil"}
	}
	if !sourcePackage.Kind().EnvironmentContract() ||
		sourcePackage.Path() == "" {
		return "", &PathError{
			Source: sourcePackage.Path(),
			Reason: "package is not an environment contract",
		}
	}
	importPath := sourcePackage.Path()
	if path.IsAbs(importPath) ||
		path.Clean(importPath) != importPath ||
		importPath == "." ||
		strings.HasPrefix(importPath, "../") {
		return "", &PathError{
			Source: importPath,
			Reason: "environment contract import path is not canonical",
		}
	}
	switch sourcePackage.Kind() {
	case load.PackageStandardLibraryContract:
		if sourcePackage.ModulePath() != "" ||
			sourcePackage.ToolchainKey() == "" {
			return "", &PathError{
				Source: importPath,
				Reason: "standard-library contract identity is incomplete",
			}
		}
		return path.Join(
			"gostdlib",
			sourcePackage.ToolchainKey(),
			importPath,
			"index.ts",
		), nil
	case load.PackageExternalContract:
		if sourcePackage.ExternalContractKey() == "" {
			return "", &PathError{
				Source: importPath,
				Reason: "external contract identity is incomplete",
			}
		}
		return path.Join(
			"externals",
			sourcePackage.ExternalContractKey(),
			importPath,
			"index.ts",
		), nil
	default:
		return "", &PathError{
			Source: importPath,
			Reason: "environment contract ownership is invalid",
		}
	}
}

func StandardLibraryConstantProjectionPath(
	sourcePackage *load.Package,
) (string, error) {
	if sourcePackage == nil ||
		sourcePackage.Kind() != load.PackageStandardLibraryContract ||
		sourcePackage.Path() == "" ||
		sourcePackage.ToolchainKey() == "" {
		return "", &PathError{Reason: "standard-library projection owner is invalid"}
	}
	importPath := sourcePackage.Path()
	if path.IsAbs(importPath) ||
		path.Clean(importPath) != importPath ||
		importPath == "." ||
		strings.HasPrefix(importPath, "../") {
		return "", &PathError{
			Source: importPath,
			Reason: "standard-library projection import path is not canonical",
		}
	}
	return path.Join(
		"support",
		"constant-projections",
		sourcePackage.ToolchainKey(),
		importPath,
		"index.ts",
	), nil
}

const (
	packageAssemblyFile = "package.ts"
	packageStateFile    = "state.ts"
)

type PathError struct {
	Source string
	Reason string
}

func (e *PathError) Error() string {
	if e.Source == "" {
		return "resolve target path: " + e.Reason
	}
	return fmt.Sprintf("resolve target path for %q: %s", e.Source, e.Reason)
}

func SourcePath(sourcePackage *load.Package, sourceFile load.File) (string, error) {
	if sourcePackage == nil {
		return "", &PathError{Reason: "source package is nil"}
	}
	if sourcePackage.ModulePath() == "" {
		return "", &PathError{
			Source: sourcePackage.Path(),
			Reason: "package has no source module identity",
		}
	}
	ownedFile, ok := sourcePackage.FileForSyntax(sourceFile.Syntax())
	if !ok || ownedFile.Path() != sourceFile.Path() {
		return "", &PathError{
			Source: sourceFile.Path(),
			Reason: "file does not belong to the source package",
		}
	}
	extension := filepath.Ext(sourceFile.Path())
	baseName := strings.TrimSuffix(filepath.Base(sourceFile.Path()), extension)
	if extension != ".go" || baseName == "" {
		return "", &PathError{
			Source: sourceFile.Path(),
			Reason: "source file is not a named Go file",
		}
	}
	packagePath, err := moduleRelativePackage(sourcePackage)
	if err != nil {
		return "", err
	}
	return path.Join(
		"modules",
		semanticModuleKey(sourcePackage),
		packagePath,
		baseName+".ts",
	), nil
}

func PackageAssemblyPath(sourcePackage *load.Package) (string, error) {
	return packageArtifactPath(sourcePackage, packageAssemblyFile)
}

func PackageStatePath(sourcePackage *load.Package) (string, error) {
	return packageArtifactPath(sourcePackage, packageStateFile)
}

func MapSpecializationPath(artifactKey string) (string, error) {
	return generatedArtifactPath("maps", artifactKey)
}

func InterfaceAdapterPath(artifactKey string) (string, error) {
	return generatedArtifactPath("interfaces/adapters", artifactKey)
}

func ProviderInterfaceBridgePath(artifactKey string) (string, error) {
	return generatedArtifactPath("interfaces/provider-bridges", artifactKey)
}

func ProviderStatefulRepresentationPath(artifactKey string) (string, error) {
	return generatedArtifactPath("providers/stateful-representations", artifactKey)
}

func GenericCapabilityPath(artifactKey string) (string, error) {
	return generatedArtifactPath("generics/capabilities", artifactKey)
}

func AnonymousInterfacePath(artifactKey string) (string, error) {
	return generatedArtifactPath("interfaces/contracts", artifactKey)
}

func generatedArtifactPath(
	directory string,
	artifactKey string,
) (string, error) {
	if len(artifactKey) != sha256.Size*2 {
		return "", &PathError{
			Source: artifactKey,
			Reason: "generated artifact key is invalid",
		}
	}
	for _, character := range artifactKey {
		if character < '0' || character > '9' &&
			character < 'a' || character > 'f' {
			return "", &PathError{
				Source: artifactKey,
				Reason: "generated artifact key is invalid",
			}
		}
	}
	return path.Join("support", directory, artifactKey+".ts"), nil
}

func packageArtifactPath(
	sourcePackage *load.Package,
	fileName string,
) (string, error) {
	if sourcePackage == nil {
		return "", &PathError{Reason: "source package is nil"}
	}
	if sourcePackage.ModulePath() == "" {
		return "", &PathError{
			Source: sourcePackage.Path(),
			Reason: "package has no source module identity",
		}
	}
	packagePath, err := moduleRelativePackage(sourcePackage)
	if err != nil {
		return "", err
	}
	return path.Join(
		"packages",
		semanticModuleKey(sourcePackage),
		packagePath,
		fileName,
	), nil
}

func semanticModuleKey(sourcePackage *load.Package) string {
	return moduleKey(sourcePackage.ModulePath(), sourcePackage.ModuleVersion())
}

func ModuleSpecifier(fromSourcePath string, toSourcePath string) (string, error) {
	if err := validateSourcePath(fromSourcePath); err != nil {
		return "", err
	}
	if err := validateSourcePath(toSourcePath); err != nil {
		return "", err
	}
	if fromSourcePath == toSourcePath {
		return "", &PathError{
			Source: fromSourcePath,
			Reason: "source module cannot import itself",
		}
	}
	relative, err := filepath.Rel(
		filepath.FromSlash(path.Dir(fromSourcePath)),
		filepath.FromSlash(toSourcePath),
	)
	if err != nil {
		return "", &PathError{
			Source: toSourcePath,
			Reason: err.Error(),
		}
	}
	relative = filepath.ToSlash(relative)
	relative = strings.TrimSuffix(relative, ".ts") + ".js"
	if !strings.HasPrefix(relative, ".") {
		relative = "./" + relative
	}
	return relative, nil
}

func moduleRelativePackage(sourcePackage *load.Package) (string, error) {
	if sourcePackage.Path() == sourcePackage.ModulePath() {
		return "_root", nil
	}
	relative, ok := strings.CutPrefix(
		sourcePackage.Path(),
		sourcePackage.ModulePath()+"/",
	)
	if !ok || relative == "" || path.Clean(relative) != relative ||
		strings.HasPrefix(relative, "../") {
		return "", &PathError{
			Source: sourcePackage.Path(),
			Reason: "package path is outside its semantic module",
		}
	}
	return relative, nil
}

func moduleKey(modulePath string, moduleVersion string) string {
	digest := sha256.Sum256([]byte(modulePath + "\x00" + moduleVersion))
	return hex.EncodeToString(digest[:])
}

func validateSourcePath(sourcePath string) error {
	if sourcePath == "" || path.IsAbs(sourcePath) ||
		path.Clean(sourcePath) != sourcePath ||
		path.Ext(sourcePath) != ".ts" ||
		strings.HasPrefix(sourcePath, "../") {
		return &PathError{
			Source: sourcePath,
			Reason: "target source path is not canonical",
		}
	}
	return nil
}
