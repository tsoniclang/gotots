package output

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/tsoniclang/gotots/internal/load"
	"golang.org/x/mod/module"
)

const (
	ProgramInitializationPath                 = "program.ts"
	RuntimePackageName                        = "@gotots/runtime"
	RuntimePackageVersion                     = "0.0.0"
	RuntimePackageRootPath                    = "runtime"
	RuntimePackageManifestPath                = "runtime/package.json"
	ScalarSupportPath                         = "runtime/scalars.ts"
	AnonymousStructSupportPath                = "support/anonymous-structs.ts"
	MapSpecializationSupportPath              = "support/maps.ts"
	InterfaceAdapterSupportPath               = "support/interface-adapters.ts"
	AnonymousInterfaceSupportPath             = "support/interface-contracts.ts"
	InterfaceMethodSupportPath                = "support/interface-methods.ts"
	InterfaceTypeSupportPath                  = "support/interface-types.ts"
	ReflectionTypeSupportPath                 = "support/reflection-types.ts"
	ProviderInterfaceBridgeSupportPath        = "support/provider-interface-bridges.ts"
	ProviderStatefulRepresentationSupportPath = "support/provider-stateful-representations.ts"
	DeferredCallableRegistrySupportPath       = "support/deferred-callables.ts"
	CallableImplementationRootPath            = "implementations"
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
	moduleOwner, err := semanticModuleKey(sourcePackage)
	if err != nil {
		return "", err
	}
	return path.Join(
		"modules",
		moduleOwner,
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

func GenericCapabilityPath(module string) (string, error) {
	return semanticGeneratedArtifactPath("generics/capabilities", module)
}

func GenericConcretizationPath(module string) (string, error) {
	return semanticGeneratedArtifactPath("generics/concretizations", module)
}

func semanticGeneratedArtifactPath(
	directory string,
	module string,
) (string, error) {
	if module == "" || path.IsAbs(module) {
		return "", &PathError{
			Source: module,
			Reason: "semantic generated artifact module is invalid",
		}
	}
	for _, segment := range strings.Split(module, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return "", &PathError{
				Source: module,
				Reason: "semantic generated artifact module is invalid",
			}
		}
		for _, character := range segment {
			if character >= 'a' && character <= 'z' ||
				character >= 'A' && character <= 'Z' ||
				character >= '0' && character <= '9' ||
				character == '_' || character == '$' {
				continue
			}
			return "", &PathError{
				Source: module,
				Reason: "semantic generated artifact module is invalid",
			}
		}
	}
	return path.Join("support", directory, module+".ts"), nil
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
	moduleOwner, err := semanticModuleKey(sourcePackage)
	if err != nil {
		return "", err
	}
	return path.Join(
		"packages",
		moduleOwner,
		packagePath,
		fileName,
	), nil
}

func semanticModuleKey(sourcePackage *load.Package) (string, error) {
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

func CallableImplementationPath(selected string) (string, error) {
	if err := validateSourcePath(selected); err != nil {
		return "", err
	}
	prefix := CallableImplementationRootPath + "/"
	if !strings.HasPrefix(selected, prefix) || len(selected) == len(prefix) {
		return "", &PathError{
			Source: selected,
			Reason: "callable implementation is outside its owned root",
		}
	}
	return selected, nil
}

func RuntimeModuleSpecifier(runtimeSourcePath string) (string, error) {
	if err := validateSourcePath(runtimeSourcePath); err != nil {
		return "", err
	}
	relative, ok := strings.CutPrefix(
		runtimeSourcePath,
		RuntimePackageRootPath+"/",
	)
	if !ok || relative == "" {
		return "", &PathError{
			Source: runtimeSourcePath,
			Reason: "source module is outside the runtime package",
		}
	}
	return RuntimePackageName + "/" +
		strings.TrimSuffix(relative, ".ts") + ".js", nil
}

func moduleRelativePackage(sourcePackage *load.Package) (string, error) {
	escapedPackage, err := module.EscapePath(sourcePackage.Path())
	if err != nil {
		return "", &PathError{
			Source: sourcePackage.Path(),
			Reason: "package path is not a valid semantic path",
		}
	}
	escapedModule, err := module.EscapePath(sourcePackage.ModulePath())
	if err != nil {
		return "", &PathError{
			Source: sourcePackage.ModulePath(),
			Reason: "module path is not a valid semantic path",
		}
	}
	if escapedPackage == escapedModule {
		return "_root", nil
	}
	relative, ok := strings.CutPrefix(
		escapedPackage,
		escapedModule+"/",
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

func moduleKey(modulePath string, moduleVersion string) (string, error) {
	escapedPath, err := module.EscapePath(modulePath)
	if err != nil {
		return "", &PathError{
			Source: modulePath,
			Reason: "module path is not a valid semantic path",
		}
	}
	if moduleVersion == "" {
		return escapedPath, nil
	}
	escapedVersion, err := module.EscapeVersion(moduleVersion)
	if err != nil {
		return "", &PathError{
			Source: moduleVersion,
			Reason: "module version is not canonical",
		}
	}
	return escapedPath + "@" + escapedVersion, nil
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
