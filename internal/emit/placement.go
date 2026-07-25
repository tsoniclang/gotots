package emit

import (
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type typeImport struct {
	modulePath   string
	exportedName string
}

type placementOwner struct {
	typeImports map[typeImport]struct{}
}

func newPlacementOwner() *placementOwner {
	return &placementOwner{typeImports: make(map[typeImport]struct{})}
}

func (p *placementOwner) TypeImport(modulePath string, exportedName string) (string, error) {
	if modulePath == "" || exportedName == "" {
		return "", &api.PlacementError{
			ModulePath:   modulePath,
			ExportedName: exportedName,
			Reason:       "module path and exported name are required",
		}
	}
	p.typeImports[typeImport{modulePath: modulePath, exportedName: exportedName}] = struct{}{}
	return exportedName, nil
}

func (p *placementOwner) Statements(factory tsgo.Factory) []tsgo.Statement {
	byModule := make(map[string][]string)
	for request := range p.typeImports {
		byModule[request.modulePath] = append(byModule[request.modulePath], request.exportedName)
	}
	modules := make([]string, 0, len(byModule))
	for modulePath := range byModule {
		modules = append(modules, modulePath)
	}
	sort.Strings(modules)

	statements := make([]tsgo.Statement, 0, len(modules))
	for _, modulePath := range modules {
		names := byModule[modulePath]
		sort.Strings(names)
		specifiers := make([]tsgo.ImportSpecifier, 0, len(names))
		for _, name := range names {
			specifiers = append(specifiers, factory.ImportSpecifier(
				false,
				nil,
				factory.Identifier(name),
			))
		}
		clause := factory.ImportClause(
			tsgo.ImportPhaseModifierSyntaxKindTypeKeyword,
			nil,
			factory.NamedImports(specifiers),
		)
		statements = append(statements, factory.ImportDeclaration(
			nil,
			clause,
			factory.StringLiteral(modulePath, tsgo.TokenFlagsNone),
			nil,
		))
	}
	return statements
}
