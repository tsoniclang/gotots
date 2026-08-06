package sourcepackage

import (
	"fmt"
	"path/filepath"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/contracts/sourceimplementation"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Paths struct {
	assembly     string
	owned        map[string]struct{}
	bySourceFile map[string]string
}

func ResolvePaths(sourcePackage *load.Package) (Paths, error) {
	if sourcePackage == nil || sourcePackage.Kind() != load.PackageSource {
		return Paths{}, fmt.Errorf("source package is absent")
	}
	assemblyPath, err := output.PackageAssemblyPath(sourcePackage)
	if err != nil {
		return Paths{}, err
	}
	statePath, err := output.PackageStatePath(sourcePackage)
	if err != nil {
		return Paths{}, err
	}
	result := Paths{
		assembly:     assemblyPath,
		bySourceFile: make(map[string]string),
		owned: map[string]struct{}{
			assemblyPath: {},
			statePath:    {},
		},
	}
	for _, sourceFile := range sourcePackage.Files() {
		outputPath, err := output.SourcePath(sourcePackage, sourceFile)
		if err != nil {
			return Paths{}, err
		}
		result.owned[outputPath] = struct{}{}
		name := filepath.Base(sourceFile.Path())
		if _, duplicate := result.bySourceFile[name]; duplicate {
			return Paths{}, fmt.Errorf("Go source filename %q is duplicated", name)
		}
		result.bySourceFile[name] = outputPath
	}
	return result, nil
}

func (p Paths) AssemblyPath() string {
	return p.assembly
}

func (p Paths) Owns(outputPath string) bool {
	_, ok := p.owned[outputPath]
	return ok
}

func (p Paths) SourcePath(goFile string) (string, bool) {
	selected, ok := p.bySourceFile[goFile]
	return selected, ok
}

func (p Paths) SourcePaths() []string {
	result := make([]string, 0, len(p.bySourceFile))
	for _, outputPath := range p.bySourceFile {
		result = append(result, outputPath)
	}
	sort.Strings(result)
	return result
}

func VerifyExports(
	implementation sourceimplementation.Implementation,
	generated tsgo.SourceFile,
) error {
	actual, err := generatedExportNames(generated)
	if err != nil {
		return err
	}
	exports := implementation.Exports()
	expected := make([]string, len(exports))
	for index, export := range exports {
		expected[index] = export.Name()
	}
	if !slices.Equal(actual, expected) {
		return fmt.Errorf(
			"exports %v differ from generated surface %v",
			expected,
			actual,
		)
	}
	return nil
}

func generatedExportNames(source tsgo.SourceFile) ([]string, error) {
	if source == nil {
		return nil, fmt.Errorf("generated package assembly is nil")
	}
	names := make(map[string]struct{})
	for _, statement := range source.Statements() {
		switch declaration := statement.(type) {
		case tsgo.FunctionDeclaration:
			if hasExportModifier(declaration.Modifiers()) {
				if declaration.Name() == nil {
					return nil, fmt.Errorf("exported function has no name")
				}
				names[declaration.Name().Text()] = struct{}{}
			}
		case tsgo.ClassDeclaration:
			if hasExportModifier(declaration.Modifiers()) {
				if declaration.Name() == nil {
					return nil, fmt.Errorf("exported class has no name")
				}
				names[declaration.Name().Text()] = struct{}{}
			}
		case tsgo.InterfaceDeclaration:
			if hasExportModifier(declaration.Modifiers()) {
				names[declaration.Name().Text()] = struct{}{}
			}
		case tsgo.TypeAliasDeclaration:
			if hasExportModifier(declaration.Modifiers()) {
				names[declaration.Name().Text()] = struct{}{}
			}
		case tsgo.EnumDeclaration:
			if hasExportModifier(declaration.Modifiers()) {
				names[declaration.Name().Text()] = struct{}{}
			}
		case tsgo.VariableStatement:
			if !hasExportModifier(declaration.Modifiers()) {
				continue
			}
			for _, variable := range declaration.DeclarationList().Declarations() {
				identifier, ok := variable.Name().(tsgo.Identifier)
				if !ok {
					return nil, fmt.Errorf("exported variable uses a binding pattern")
				}
				names[identifier.Text()] = struct{}{}
			}
		case tsgo.ExportDeclaration:
			named, ok := declaration.ExportClause().(tsgo.NamedExports)
			if !ok {
				return nil, fmt.Errorf("generated package uses a non-named export")
			}
			for _, specifier := range named.Elements() {
				name, ok := moduleExportName(specifier.Name())
				if !ok {
					return nil, fmt.Errorf("generated package export name is not static")
				}
				names[name] = struct{}{}
			}
		case tsgo.ExportAssignment, tsgo.NamespaceExportDeclaration:
			return nil, fmt.Errorf("generated package uses an unsupported export form")
		}
	}
	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	sort.Strings(result)
	return result, nil
}

func hasExportModifier(modifiers []tsgo.ModifierLike) bool {
	for _, modifier := range modifiers {
		if modifier.Kind() == tsgo.SyntaxKindExportKeyword {
			return true
		}
	}
	return false
}

func moduleExportName(name tsgo.ModuleExportName) (string, bool) {
	switch selected := name.(type) {
	case tsgo.Identifier:
		return selected.Text(), true
	case tsgo.StringLiteral:
		return selected.Text(), true
	default:
		return "", false
	}
}
