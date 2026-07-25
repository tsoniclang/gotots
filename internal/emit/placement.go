package emit

import (
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type importRequest struct {
	typeOnly     bool
	modulePath   string
	exportedName string
}

type placementOwner struct {
	imports map[importRequest]struct{}
}

func newPlacementOwner() *placementOwner {
	return &placementOwner{imports: make(map[importRequest]struct{})}
}

func (p *placementOwner) TypeImport(modulePath string, exportedName string) (string, error) {
	return p.importName(true, modulePath, exportedName)
}

func (p *placementOwner) ValueImport(modulePath string, exportedName string) (string, error) {
	return p.importName(false, modulePath, exportedName)
}

func (p *placementOwner) importName(
	typeOnly bool,
	modulePath string,
	exportedName string,
) (string, error) {
	if modulePath == "" || exportedName == "" {
		return "", &api.PlacementError{
			ModulePath:   modulePath,
			ExportedName: exportedName,
			Reason:       "module path and exported name are required",
		}
	}
	p.imports[importRequest{
		typeOnly:     typeOnly,
		modulePath:   modulePath,
		exportedName: exportedName,
	}] = struct{}{}
	return exportedName, nil
}

func (p *placementOwner) Statements(factory tsgo.Factory) []tsgo.Statement {
	type importGroup struct {
		typeOnly   bool
		modulePath string
	}
	byGroup := make(map[importGroup][]string)
	for request := range p.imports {
		group := importGroup{typeOnly: request.typeOnly, modulePath: request.modulePath}
		byGroup[group] = append(byGroup[group], request.exportedName)
	}
	groups := make([]importGroup, 0, len(byGroup))
	for group := range byGroup {
		groups = append(groups, group)
	}
	sort.Slice(groups, func(left, right int) bool {
		if groups[left].typeOnly != groups[right].typeOnly {
			return groups[left].typeOnly
		}
		return groups[left].modulePath < groups[right].modulePath
	})

	statements := make([]tsgo.Statement, 0, len(groups))
	for _, group := range groups {
		names := byGroup[group]
		sort.Strings(names)
		specifiers := make([]tsgo.ImportSpecifier, 0, len(names))
		for _, name := range names {
			specifiers = append(specifiers, factory.ImportSpecifier(
				false,
				nil,
				factory.Identifier(name),
			))
		}
		var phase tsgo.ImportPhaseModifierSyntaxKind
		if group.typeOnly {
			phase = tsgo.ImportPhaseModifierSyntaxKindTypeKeyword
		}
		clause := factory.ImportClause(
			phase,
			nil,
			factory.NamedImports(specifiers),
		)
		statements = append(statements, factory.ImportDeclaration(
			nil,
			clause,
			factory.StringLiteral(group.modulePath, tsgo.TokenFlagsNone),
			nil,
		))
	}
	return statements
}
