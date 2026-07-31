package placement

import (
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type Owner struct {
	requests map[api.RootRequestOwner]api.RootRequest
}

func (p *Owner) RuntimeSymbols() []api.RuntimeSymbol {
	symbols := make([]api.RuntimeSymbol, 0)
	seen := make(map[api.RuntimeSymbol]struct{})
	for _, request := range p.requests {
		symbol, ok := request.RuntimeSymbol()
		if !ok {
			continue
		}
		if _, duplicate := seen[symbol]; duplicate {
			continue
		}
		seen[symbol] = struct{}{}
		symbols = append(symbols, symbol)
	}
	slices.Sort(symbols)
	return symbols
}

func (p *Owner) PrimitiveAliases() []api.PrimitiveAlias {
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

func New() *Owner {
	return &Owner{
		requests: make(map[api.RootRequestOwner]api.RootRequest),
	}
}

func (p *Owner) Requests() []api.RootRequest {
	requests := make([]api.RootRequest, 0, len(p.requests))
	for _, request := range p.requests {
		requests = append(requests, request)
	}
	return requests
}

func (p *Owner) Apply(requests []api.RootRequest) error {
	return api.WalkRootRequests(requests, func(request api.RootRequest) error {
		if request.Kind() != api.RootRequestImport ||
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
			return nil
		}
		p.requests[owner] = request
		return nil
	})
}

func (p *Owner) RequireTypeOnly() error {
	for _, request := range p.requests {
		if request.Kind() != api.RootRequestImport ||
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

func (p *Owner) Statements(factory tsgo.Factory) []tsgo.Statement {
	type importGroup struct {
		phase      api.ImportPhase
		binding    api.ImportBindingKind
		modulePath string
	}
	byGroup := make(map[importGroup][]api.RootRequest)
	for _, request := range p.requests {
		group := importGroup{
			phase:      request.ImportPhase(),
			binding:    request.ImportBinding(),
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
		if groups[left].modulePath != groups[right].modulePath {
			return groups[left].modulePath < groups[right].modulePath
		}
		if groups[left].binding != groups[right].binding {
			return groups[left].binding < groups[right].binding
		}
		return false
	})

	statements := make([]tsgo.Statement, 0, len(groups))
	for _, group := range groups {
		requests := byGroup[group]
		sort.Slice(requests, func(left, right int) bool {
			return requests[left].LocalName() < requests[right].LocalName()
		})
		var phase tsgo.ImportPhaseModifierSyntaxKind
		if group.phase == api.ImportPhaseType {
			phase = tsgo.ImportPhaseModifierSyntaxKindTypeKeyword
		}
		var bindings tsgo.NamedImportBindings
		switch group.binding {
		case api.ImportBindingNamed:
			specifiers := make([]tsgo.ImportSpecifier, 0, len(requests))
			for _, request := range requests {
				specifiers = append(specifiers, request.Specifier())
			}
			bindings = factory.NamedImports(specifiers)
		case api.ImportBindingNamespace:
			if len(requests) != 1 {
				panic("namespace import group has multiple owners")
			}
			bindings = requests[0].NamespaceSpecifier()
		default:
			panic("import group has invalid binding kind")
		}
		clause := factory.ImportClause(
			phase,
			nil,
			bindings,
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
