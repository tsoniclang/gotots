package sourcepackage

import (
	"fmt"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/contracts/sourceimplementation"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Consumer struct {
	OutputPath string
	SourceFile tsgo.SourceFile
}

func RebindConsumers(
	factory tsgo.Factory,
	paths Paths,
	implementation sourceimplementation.Implementation,
	consumers []Consumer,
) ([]Consumer, error) {
	public := make(map[string]struct{})
	for _, export := range implementation.Exports() {
		public[export.Name()] = struct{}{}
	}
	provided := make(map[string][]string)
	for _, module := range implementation.PrivateModules() {
		outputPath, ok := paths.SourcePath(module.GoFile())
		if !ok {
			return nil, fmt.Errorf("private module Go source %q is absent", module.GoFile())
		}
		exports := module.Exports()
		names := make([]string, len(exports))
		for index, export := range exports {
			names[index] = export.Name()
		}
		provided[outputPath] = names
	}
	return rebindConsumers(factory, paths, public, provided, consumers)
}

func rebindConsumers(
	factory tsgo.Factory,
	paths Paths,
	public map[string]struct{},
	provided map[string][]string,
	consumers []Consumer,
) ([]Consumer, error) {
	generated := make(map[string]struct{})
	for _, outputPath := range paths.SourcePaths() {
		generated[outputPath] = struct{}{}
	}
	required := make(map[string]map[string]struct{})
	result := slices.Clone(consumers)
	for index, consumer := range result {
		if paths.Owns(consumer.OutputPath) || consumer.SourceFile == nil {
			continue
		}
		statements := consumer.SourceFile.Statements()
		changed := false
		for statementIndex, statement := range statements {
			declaration, ok := statement.(tsgo.ImportDeclaration)
			if !ok {
				continue
			}
			literal, ok := declaration.ModuleSpecifier().(tsgo.StringLiteral)
			if !ok {
				continue
			}
			imported, ok := resolveGeneratedImport(consumer.OutputPath, literal.Text())
			if !ok {
				continue
			}
			if _, selected := generated[imported]; !selected {
				continue
			}
			names, err := importedNames(declaration.ImportClause())
			if err != nil {
				return nil, fmt.Errorf("generated file %q: %w", consumer.OutputPath, err)
			}
			allPublic := true
			for _, name := range names {
				if _, ok := public[name]; !ok {
					allPublic = false
					break
				}
			}
			if allPublic {
				module, err := targetoutput.ModuleSpecifier(
					consumer.OutputPath,
					paths.AssemblyPath(),
				)
				if err != nil {
					return nil, err
				}
				statements[statementIndex] = factory.ImportDeclaration(
					declaration.Modifiers(),
					declaration.ImportClause(),
					factory.StringLiteral(module, literal.TokenFlags()),
					declaration.Attributes(),
				)
				changed = true
				continue
			}
			if !typeOnlyImport(declaration.ImportClause()) {
				return nil, fmt.Errorf(
					"generated file %q has a private value dependency on replaced module %q",
					consumer.OutputPath,
					imported,
				)
			}
			if required[imported] == nil {
				required[imported] = make(map[string]struct{})
			}
			for _, name := range names {
				required[imported][name] = struct{}{}
			}
		}
		if changed {
			result[index].SourceFile = factory.SourceFile(
				statements,
				consumer.SourceFile.EndOfFileToken(),
				consumer.SourceFile.SourceData(),
			)
		}
	}
	if err := verifyPrivateModules(required, provided); err != nil {
		return nil, err
	}
	return result, nil
}

func importedNames(clause tsgo.ImportClause) ([]string, error) {
	if clause == nil || clause.Name() != nil {
		return nil, fmt.Errorf("dependency on replaced module is not a named import")
	}
	named, ok := clause.NamedBindings().(tsgo.NamedImports)
	if !ok || len(named.Elements()) == 0 {
		return nil, fmt.Errorf("dependency on replaced module is not a named import")
	}
	result := make([]string, 0, len(named.Elements()))
	for _, specifier := range named.Elements() {
		name := specifier.Name().Text()
		if specifier.PropertyName() != nil {
			var ok bool
			name, ok = moduleExportName(specifier.PropertyName())
			if !ok {
				return nil, fmt.Errorf("dependency on replaced module has a dynamic name")
			}
		}
		result = append(result, name)
	}
	sort.Strings(result)
	return result, nil
}

func typeOnlyImport(clause tsgo.ImportClause) bool {
	if clause == nil || clause.Name() != nil {
		return false
	}
	if clause.PhaseModifier() == tsgo.ImportPhaseModifierSyntaxKindTypeKeyword {
		return true
	}
	named, ok := clause.NamedBindings().(tsgo.NamedImports)
	if !ok || len(named.Elements()) == 0 {
		return false
	}
	for _, specifier := range named.Elements() {
		if !specifier.IsTypeOnly() {
			return false
		}
	}
	return true
}

func resolveGeneratedImport(fromPath string, specifier string) (string, bool) {
	if !strings.HasPrefix(specifier, ".") || !strings.HasSuffix(specifier, ".js") {
		return "", false
	}
	return path.Clean(path.Join(
		path.Dir(fromPath),
		strings.TrimSuffix(specifier, ".js")+".ts",
	)), true
}

func verifyPrivateModules(
	required map[string]map[string]struct{},
	provided map[string][]string,
) error {
	paths := make(map[string]struct{}, len(required)+len(provided))
	for outputPath := range required {
		paths[outputPath] = struct{}{}
	}
	for outputPath := range provided {
		paths[outputPath] = struct{}{}
	}
	ordered := make([]string, 0, len(paths))
	for outputPath := range paths {
		ordered = append(ordered, outputPath)
	}
	sort.Strings(ordered)
	for _, outputPath := range ordered {
		names := make([]string, 0, len(required[outputPath]))
		for name := range required[outputPath] {
			names = append(names, name)
		}
		sort.Strings(names)
		if !slices.Equal(names, provided[outputPath]) {
			return fmt.Errorf(
				"private module %q exports %v, want generated type dependencies %v",
				outputPath,
				provided[outputPath],
				names,
			)
		}
	}
	return nil
}
