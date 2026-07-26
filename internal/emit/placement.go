package emit

import (
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type placementOwner struct {
	requests map[api.PlacementOwner]api.PlacementRequest
}

func (p *placementOwner) PrimitiveAliases() []api.PrimitiveAlias {
	aliases := make([]api.PrimitiveAlias, 0)
	seen := make(map[api.PrimitiveAlias]struct{})
	for _, request := range p.requests {
		alias, ok := request.PrimitiveAlias()
		if !ok {
			continue
		}
		if _, duplicate := seen[alias]; duplicate {
			continue
		}
		seen[alias] = struct{}{}
		aliases = append(aliases, alias)
	}
	slices.Sort(aliases)
	return aliases
}

func newPlacementOwner() *placementOwner {
	return &placementOwner{
		requests: make(map[api.PlacementOwner]api.PlacementRequest),
	}
}

func (p *placementOwner) Apply(requests []api.PlacementRequest) error {
	for _, request := range requests {
		if request.Kind() != api.PlacementImport ||
			request.LegalScope() != api.ScopeFileImports ||
			request.PreferredScope() != api.ScopeFileImports ||
			request.Execution() != api.ExecutionStatic {
			return &api.PlacementError{
				ModulePath:   request.ModulePath(),
				ExportedName: request.ExportedName(),
				Reason:       "request is not a static file import",
			}
		}
		owner := request.Owner()
		if existing, ok := p.requests[owner]; ok {
			if existing.LocalName() != request.LocalName() {
				return &api.PlacementError{
					ModulePath:   request.ModulePath(),
					ExportedName: request.ExportedName(),
					Reason:       "one import owner requested multiple local names",
				}
			}
			if existing.ImportPhase() == api.ImportPhaseType &&
				request.ImportPhase() == api.ImportPhaseValue {
				p.requests[owner] = request
			}
			continue
		}
		p.requests[owner] = request
	}
	return nil
}

func (p *placementOwner) RequireTypeOnly() error {
	for _, request := range p.requests {
		if request.Kind() != api.PlacementImport ||
			request.ImportPhase() != api.ImportPhaseType {
			return &api.PlacementError{
				ModulePath:   request.ModulePath(),
				ExportedName: request.ExportedName(),
				Reason:       "package state requires type-only imports",
			}
		}
	}
	return nil
}

func (p *placementOwner) Statements(factory tsgo.Factory) []tsgo.Statement {
	type importGroup struct {
		phase      api.ImportPhase
		modulePath string
	}
	byGroup := make(map[importGroup][]api.PlacementRequest)
	for _, request := range p.requests {
		group := importGroup{
			phase:      request.ImportPhase(),
			modulePath: request.ModulePath(),
		}
		byGroup[group] = append(byGroup[group], request)
	}
	groups := make([]importGroup, 0, len(byGroup))
	for group := range byGroup {
		groups = append(groups, group)
	}
	sort.Slice(groups, func(left, right int) bool {
		if groups[left].phase != groups[right].phase {
			return groups[left].phase < groups[right].phase
		}
		return groups[left].modulePath < groups[right].modulePath
	})

	statements := make([]tsgo.Statement, 0, len(groups))
	for _, group := range groups {
		requests := byGroup[group]
		sort.Slice(requests, func(left, right int) bool {
			return requests[left].LocalName() < requests[right].LocalName()
		})
		specifiers := make([]tsgo.ImportSpecifier, 0, len(requests))
		for _, request := range requests {
			specifiers = append(specifiers, request.Specifier())
		}
		var phase tsgo.ImportPhaseModifierSyntaxKind
		if group.phase == api.ImportPhaseType {
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
			requests[0].ModuleSpecifier(),
			nil,
		))
	}
	return statements
}
