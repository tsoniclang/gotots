package emit

import (
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	"github.com/tsoniclang/gotots/internal/emit/environmentcontract"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type environmentArtifact struct {
	placement       *targetplacement.Owner
	dependencies    []api.ArtifactDependency
	contract        artifactstate.Contract
	reconstructions uint64
}

type environmentDeclaration struct {
	object     types.Object
	name       string
	statements []tsgo.Statement
	environmentArtifact
}

type environmentStateField struct {
	field tsgo.TypeElement
	environmentArtifact
}

type environmentBuiltin struct {
	emitted    bool
	signatures []*types.Signature
	statements []tsgo.Statement
	environmentArtifact
}

type environmentConstantProjection struct {
	source     *types.Const
	projection types.BasicKind
	name       string
	statement  tsgo.Statement
	contract   artifactstate.Contract
}

func (s *programSession) buildEnvironmentDeclaration(
	builder *environmentContractBuilder,
	object types.Object,
	requirements []api.DeclarationRequirement,
) (environmentDeclaration, error) {
	for {
		builder.building[object] = true
		target, err := environmentcontract.Declaration(
			builder.context,
			builder.emitter,
			object,
			requirements,
		)
		if err != nil {
			delete(builder.building, object)
			return environmentDeclaration{},
				environmentContractError(object, err)
		}
		owner := api.MustSourceArtifactOwner(object)
		placement, dependencies, err := s.consumeArtifactRequests(
			owner,
			target.Requests(),
		)
		if err != nil {
			delete(builder.building, object)
			return environmentDeclaration{},
				environmentContractError(object, err)
		}
		current := builder.environmentRequirements(object)
		delete(builder.building, object)
		if len(current) != len(requirements) {
			requirements = current
			continue
		}
		contract, err := environmentDeclarationContract(s.factory, target)
		if err != nil {
			return environmentDeclaration{},
				environmentContractError(object, err)
		}
		return environmentDeclaration{
			object:     object,
			name:       object.Name(),
			statements: target.Declarations(),
			environmentArtifact: environmentArtifact{
				placement:    placement,
				dependencies: dependencies,
				contract:     contract,
			},
		}, nil
	}
}

func environmentDeclarationContract(
	factory tsgo.Factory,
	target api.DeclarationEmission,
) (artifactstate.Contract, error) {
	switch target.Disposition() {
	case api.DeclarationDispositionMaterialized:
		return artifactstate.ProjectContract(factory, target.Declarations())
	case api.DeclarationDispositionCoverageOnly:
		return artifactstate.ProjectCoverageContract(
			factory,
			target.Declarations(),
		)
	default:
		return artifactstate.Contract{}, &ScheduleError{
			Reason: "environment declaration disposition is invalid",
		}
	}
}

func (s *programSession) reconstructEnvironmentDeclaration(
	builder *environmentContractBuilder,
	object types.Object,
) error {
	current, emitted := builder.declarations[object]
	if !emitted {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "environment declaration was not emitted before reconstruction",
		}
	}
	target, err := s.buildEnvironmentDeclaration(
		builder,
		object,
		builder.environmentRequirements(object),
	)
	if err != nil {
		return err
	}
	owner := api.MustSourceArtifactOwner(object)
	if err := s.artifacts.Commit(
		owner,
		target.contract,
		target.dependencies,
	); err != nil {
		return err
	}
	s.artifacts.DiscardDirty(owner)
	target.reconstructions = current.reconstructions + 1
	builder.declarations[object] = target
	return nil
}

func (s *programSession) emitEnvironmentStateField(
	builder *environmentContractBuilder,
	variable *types.Var,
) error {
	target, err := s.buildEnvironmentStateField(builder, variable)
	if err != nil {
		return err
	}
	owner := api.MustSourceArtifactOwner(variable)
	if err := s.artifacts.Commit(
		owner,
		target.contract,
		target.dependencies,
	); err != nil {
		return err
	}
	builder.stateFields[variable] = target
	return nil
}

func (s *programSession) buildEnvironmentStateField(
	builder *environmentContractBuilder,
	variable *types.Var,
) (environmentStateField, error) {
	field, requests, err := environmentcontract.StateField(
		builder.context,
		builder.emitter,
		variable,
	)
	if err != nil {
		return environmentStateField{},
			environmentContractError(variable, err)
	}
	owner := api.MustSourceArtifactOwner(variable)
	placement, dependencies, err := s.consumeArtifactRequests(owner, requests)
	if err != nil {
		return environmentStateField{},
			environmentContractError(variable, err)
	}
	contract, err := artifactstate.ProjectFacet(
		api.ArtifactFacetValueSurface,
		field,
	)
	if err != nil {
		return environmentStateField{},
			environmentContractError(variable, err)
	}
	return environmentStateField{
		field: field,
		environmentArtifact: environmentArtifact{
			placement:    placement,
			dependencies: dependencies,
			contract:     contract,
		},
	}, nil
}

func (s *programSession) reconstructEnvironmentStateField(
	builder *environmentContractBuilder,
	variable *types.Var,
) error {
	current, emitted := builder.stateFields[variable]
	if !emitted {
		return &ScheduleError{
			Object: variable.Name(),
			Reason: "environment state field was not emitted before reconstruction",
		}
	}
	target, err := s.buildEnvironmentStateField(builder, variable)
	if err != nil {
		return err
	}
	owner := api.MustSourceArtifactOwner(variable)
	if err := s.artifacts.Commit(
		owner,
		target.contract,
		target.dependencies,
	); err != nil {
		return err
	}
	s.artifacts.DiscardDirty(owner)
	target.reconstructions = current.reconstructions + 1
	builder.stateFields[variable] = target
	return nil
}

func (s *programSession) reconstructEnvironmentBuiltin(
	builder *environmentContractBuilder,
	builtin *types.Builtin,
) error {
	current, emitted := builder.builtins[builtin]
	if !emitted || len(current.signatures) == 0 {
		return &ScheduleError{
			Object: builtin.Name(),
			Reason: "environment builtin has no selected overload",
		}
	}
	target, err := s.buildEnvironmentBuiltin(
		builder,
		builtin,
		current.signatures,
	)
	if err != nil {
		return err
	}
	owner := api.MustSourceArtifactOwner(builtin)
	if err := s.artifacts.Commit(
		owner,
		target.contract,
		target.dependencies,
	); err != nil {
		return err
	}
	s.artifacts.DiscardDirty(owner)
	if len(current.statements) != 0 {
		target.reconstructions = current.reconstructions + 1
	}
	target.emitted = current.emitted
	builder.builtins[builtin] = target
	return nil
}

func (s *programSession) buildEnvironmentBuiltin(
	builder *environmentContractBuilder,
	builtin *types.Builtin,
	signatures []*types.Signature,
) (environmentBuiltin, error) {
	ordered := append([]*types.Signature(nil), signatures...)
	sort.Slice(ordered, func(left, right int) bool {
		return emitordering.StableTypeString(ordered[left]) <
			emitordering.StableTypeString(ordered[right])
	})
	var statements []tsgo.Statement
	var requests []api.RootRequest
	for _, signature := range ordered {
		target, err := environmentcontract.BuiltinDeclaration(
			builder.context,
			builder.emitter,
			builtin,
			signature,
		)
		if err != nil {
			return environmentBuiltin{},
				environmentContractError(builtin, err)
		}
		statements = append(statements, target.Declarations()...)
		requests = append(requests, target.Requests()...)
	}
	owner := api.MustSourceArtifactOwner(builtin)
	placement, dependencies, err := s.consumeArtifactRequests(owner, requests)
	if err != nil {
		return environmentBuiltin{},
			environmentContractError(builtin, err)
	}
	nodes := make([]tsgo.Node, len(statements))
	for index, statement := range statements {
		nodes[index] = statement
	}
	contract, err := artifactstate.ProjectFacet(
		api.ArtifactFacetCallableSignature,
		builder.context.Factory().SyntaxList(nodes),
	)
	if err != nil {
		return environmentBuiltin{},
			environmentContractError(builtin, err)
	}
	return environmentBuiltin{
		signatures: ordered,
		statements: statements,
		environmentArtifact: environmentArtifact{
			placement:    placement,
			dependencies: dependencies,
			contract:     contract,
		},
	}, nil
}

func (b *environmentContractBuilder) committedPlacement() (
	*targetplacement.Owner,
	error,
) {
	placement := targetplacement.New()
	if err := placement.Apply(b.placement.Requests()); err != nil {
		return nil, err
	}
	for _, target := range b.declarations {
		if err := placement.Apply(target.placement.Requests()); err != nil {
			return nil, err
		}
	}
	for _, target := range b.stateFields {
		if err := placement.Apply(target.placement.Requests()); err != nil {
			return nil, err
		}
	}
	for _, target := range b.builtins {
		if target.placement == nil {
			continue
		}
		if err := placement.Apply(target.placement.Requests()); err != nil {
			return nil, err
		}
	}
	return placement, nil
}

func (s *programSession) reconstructEnvironmentArtifact(
	owner types.Object,
) error {
	sourcePackage := s.source.EnvironmentForTypes(owner.Pkg())
	if sourcePackage == nil {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "environment artifact lost its source package",
		}
	}
	builder := s.environmentBuilders[sourcePackage]
	if builder == nil {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "environment artifact lost its target builder",
		}
	}
	switch selected := owner.(type) {
	case *types.Var:
		return s.reconstructEnvironmentStateField(builder, selected)
	case *types.Builtin:
		return s.reconstructEnvironmentBuiltin(builder, selected)
	default:
		return s.reconstructEnvironmentDeclaration(builder, owner)
	}
}

func (s *programSession) environmentArtifactSource(
	object types.Object,
) bool {
	if object == nil ||
		object.Pkg() == nil ||
		s.source.EnvironmentForTypes(object.Pkg()) == nil {
		return false
	}
	switch object.(type) {
	case *types.Func, *types.Const, *types.TypeName, *types.Var, *types.Builtin:
		return true
	default:
		return false
	}
}
