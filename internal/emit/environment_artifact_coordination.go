package emit

import (
	"go/types"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	constantbinding "github.com/tsoniclang/gotots/internal/emit/constant"
	"github.com/tsoniclang/gotots/internal/emit/environmentcontract"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	canonicalsourcefact "github.com/tsoniclang/gotots/internal/emit/sourcefact"
	environmentsourcefact "github.com/tsoniclang/gotots/internal/emit/sourcefact/environment"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type environmentArtifact struct {
	placement       *targetplacement.Owner
	dependencies    []api.ArtifactDependency
	requestRoots    []api.RootRequest
	contract        artifactstate.Contract
	reconstructions uint64
}

func (s *programSession) validateProviderStatefulArtifact(
	artifact *api.GeneratedArtifact,
) error {
	source, sourceOK := artifact.ProviderStatefulRepresentationType()
	binding, found := s.registry.GeneratedArtifact(
		api.GeneratedArtifactProviderStatefulRepresentation,
		artifact.ArtifactKey(),
	)
	bound, boundOK := binding.ProviderStatefulRepresentationType()
	if !sourceOK || !found || binding != artifact || !boundOK ||
		!types.Identical(source, bound) {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "provider stateful-representation artifact has no exact canonical binding",
		}
	}
	return nil
}

type environmentDeclaration struct {
	object           types.Object
	name             string
	statements       []tsgo.Statement
	providerCoverage bool
	environmentArtifact
}

func (s *programSession) buildProviderCoverageDeclaration(
	object types.Object,
) (environmentDeclaration, error) {
	if object == nil || s.standardLibrary == nil ||
		!s.registry.HasProviderCoverageOwner(object) {
		return environmentDeclaration{}, &ScheduleError{
			Reason: "provider coverage owner is invalid",
		}
	}
	contract, err := artifactstate.ProjectCoverageContract(s.factory, nil)
	if err != nil {
		return environmentDeclaration{}, environmentContractError(object, err)
	}
	return environmentDeclaration{
		object:           object,
		name:             object.Name(),
		providerCoverage: true,
		environmentArtifact: environmentArtifact{
			placement: targetplacement.New(),
			contract:  contract,
		},
	}, nil
}

type environmentStateField struct {
	field tsgo.TypeElement
	environmentArtifact
}

type environmentConstantProjection struct {
	source     *types.Const
	projection types.BasicKind
	name       string
	statement  tsgo.Statement
	facts      []tsgo.Statement
	contract   artifactstate.Contract
}

func (s *programSession) buildEnvironmentDeclaration(
	builder *environmentContractBuilder,
	object types.Object,
	requirements []api.DeclarationRequirement,
) (environmentDeclaration, error) {
	target, err := environmentcontract.Declaration(
		builder.context,
		builder.emitter,
		object,
		requirements,
	)
	if err != nil {
		return environmentDeclaration{},
			environmentContractError(object, err)
	}
	owner := api.MustSourceArtifactOwner(object)
	statements := target.Declarations()
	requests := target.Requests()
	if target.Disposition() == api.DeclarationDispositionMaterialized {
		factOwner, ownerErr := environmentsourcefact.New(
			builder.context,
			builder.sourcePackage,
			builder.outputPath,
			s.source.SourceDigest(),
			s.registry,
			s.standardLibrary,
		)
		if ownerErr != nil {
			return environmentDeclaration{}, environmentContractError(object, ownerErr)
		}
		origin, originErr := factOwner.Origin(object)
		if originErr != nil {
			return environmentDeclaration{}, environmentContractError(object, originErr)
		}
		facts, factErr := canonicalsourcefact.Declaration(
			builder.context,
			object,
			origin,
			statements,
		)
		if factErr != nil {
			return environmentDeclaration{}, environmentContractError(object, factErr)
		}
		statements = append(statements, facts.Statements()...)
		requests = api.CombineRequests(requests, facts.Requests())
		if typeName, ok := object.(*types.TypeName); ok && !typeName.IsAlias() {
			origins, originErr := factOwner.MemberOrigins(typeName)
			if originErr != nil {
				return environmentDeclaration{}, environmentContractError(object, originErr)
			}
			members, memberErr := canonicalsourcefact.TypeMembers(
				builder.context,
				typeName,
				origins,
			)
			if memberErr != nil {
				return environmentDeclaration{}, environmentContractError(object, memberErr)
			}
			statements = append(statements, members.Statements()...)
			requests = api.CombineRequests(requests, members.Requests())
		}
		if function, ok := object.(*types.Func); ok {
			external, linked, externalErr := s.externalImplementationSourceFact(
				builder.context,
				function,
				statements,
			)
			if externalErr != nil {
				return environmentDeclaration{}, environmentContractError(object, externalErr)
			}
			if linked {
				statements = append(statements, external.Statements()...)
				requests = api.CombineRequests(requests, external.Requests())
			}
		}
	}
	placement, dependencies, requestedRequirements, err :=
		s.consumeArtifactRequests(owner, requests)
	if err != nil {
		return environmentDeclaration{}, environmentContractError(object, err)
	}
	contractTarget := target
	if target.Disposition() == api.DeclarationDispositionMaterialized {
		contractTarget, err = api.NewDeclarationEmission(statements, requests)
		if err != nil {
			return environmentDeclaration{}, environmentContractError(object, err)
		}
	}
	contract, err := environmentDeclarationContract(s.factory, contractTarget)
	if err != nil {
		return environmentDeclaration{}, environmentContractError(object, err)
	}
	return environmentDeclaration{
		object:     object,
		name:       object.Name(),
		statements: statements,
		environmentArtifact: environmentArtifact{
			placement:    placement,
			dependencies: dependencies,
			requestRoots: requestedRequirements,
			contract:     contract,
		},
	}, nil
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
	requirements, err := environmentcontract.SelectDeclarationRequirements(
		object,
		s.requirements.SelectedFor(api.MustSourceArtifactOwner(object)),
	)
	if err != nil {
		return err
	}
	var target environmentDeclaration
	if current.providerCoverage {
		target, err = s.buildProviderCoverageDeclaration(object)
	} else {
		target, err = s.buildEnvironmentDeclaration(
			builder,
			object,
			requirements,
		)
	}
	if err != nil {
		return err
	}
	owner := api.MustSourceArtifactOwner(object)
	if err := s.commitArtifactRevision(
		owner,
		target.contract,
		target.dependencies,
		target.requestRoots,
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
	if err := s.commitArtifactRevision(
		owner,
		target.contract,
		target.dependencies,
		target.requestRoots,
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
	placement, dependencies, requirements, err :=
		s.consumeArtifactRequests(owner, requests)
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
			requestRoots: requirements,
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
	if err := s.commitArtifactRevision(
		owner,
		target.contract,
		target.dependencies,
		target.requestRoots,
	); err != nil {
		return err
	}
	s.artifacts.DiscardDirty(owner)
	target.reconstructions = current.reconstructions + 1
	builder.stateFields[variable] = target
	return nil
}

func (s *programSession) replaceEnvironmentConstantProjections(
	builder *environmentContractBuilder,
	selected *types.Const,
	requirements []api.DeclarationRequirement,
) error {
	current, emitted := builder.declarations[selected]
	if !emitted {
		return &ScheduleError{
			Object: selected.Name(),
			Reason: "environment constant was not emitted before requirement replacement",
		}
	}
	base, ok := s.registry.Target(selected)
	if !ok {
		return &ScheduleError{
			Object: selected.Name(),
			Reason: "environment constant has no target binding",
		}
	}
	next := make(map[string]environmentConstantProjection, len(requirements))
	statements := make([]tsgo.Statement, 0, len(requirements))
	var requests []api.RootRequest
	for _, requirement := range requirements {
		constant, projection, ok := requirement.ConstantProjection()
		if !ok || constant != selected {
			return &ScheduleError{
				Object: selected.Name(),
				Reason: "environment constant projection requirement is invalid",
			}
		}
		name, err := api.ConstantProjectionName(base.Name, projection)
		if err != nil {
			return err
		}
		var statement tsgo.Statement
		var selectedRequests []api.RootRequest
		if s.standardLibrary != nil &&
			builder.sourcePackage.Kind() == load.PackageStandardLibraryContract {
			emission, projectionErr := constantbinding.EmitProjection(
				builder.context,
				builder.emitter,
				nil,
				selected,
				name,
				projection,
				api.RolePackageConstantType,
				api.RolePackageConstantValue,
			)
			if projectionErr != nil {
				return environmentContractError(selected, projectionErr)
			}
			statement = emission.ExportedStatement(s.factory)
			selectedRequests = emission.Requests()
		} else {
			var projectionErr error
			statement, selectedRequests, projectionErr =
				environmentcontract.ConstantProjection(
					builder.context,
					builder.emitter,
					selected,
					projection,
				)
			if projectionErr != nil {
				return projectionErr
			}
		}
		factOwner, ownerErr := environmentsourcefact.New(
			builder.context,
			builder.sourcePackage,
			builder.outputPath,
			s.source.SourceDigest(),
			s.registry,
			s.standardLibrary,
		)
		if ownerErr != nil {
			return environmentContractError(selected, ownerErr)
		}
		origin, originErr := factOwner.Origin(selected)
		if originErr != nil {
			return environmentContractError(selected, originErr)
		}
		fact, factErr := canonicalsourcefact.ConstantProjection(
			builder.context,
			selected,
			name,
			projection,
			origin,
			[]tsgo.Statement{statement},
		)
		if factErr != nil {
			return environmentContractError(selected, factErr)
		}
		selectedRequests = api.CombineRequests(
			selectedRequests,
			fact.Requests(),
		)
		projectionStatements := append(
			[]tsgo.Statement{statement},
			fact.Statements()...,
		)
		contract, err := artifactstate.ProjectContract(
			s.factory,
			projectionStatements,
		)
		if err != nil {
			return environmentContractError(selected, err)
		}
		next[name] = environmentConstantProjection{
			source:     selected,
			projection: projection,
			name:       name,
			statement:  statement,
			facts:      fact.Statements(),
			contract:   contract,
		}
		statements = append(statements, projectionStatements...)
		requests = append(requests, selectedRequests...)
	}
	owner := api.MustSourceArtifactOwner(selected)
	placement, dependencies, nestedRequirements, err :=
		s.consumeArtifactRequests(owner, requests)
	if err != nil {
		return err
	}
	var contract artifactstate.Contract
	if len(statements) == 0 {
		contract, err = artifactstate.ProjectCoverageContract(s.factory, nil)
	} else {
		contract, err = artifactstate.ProjectContract(s.factory, statements)
	}
	if err != nil {
		return environmentContractError(selected, err)
	}
	if err := s.commitArtifactRevision(
		owner,
		contract,
		dependencies,
		nestedRequirements,
	); err != nil {
		return err
	}
	for name, projection := range builder.projections {
		if projection.source == selected {
			delete(builder.projections, name)
		}
	}
	for name, projection := range next {
		builder.projections[name] = projection
	}
	s.artifacts.DiscardDirty(owner)
	current.placement = placement
	current.dependencies = dependencies
	current.requestRoots = nestedRequirements
	current.reconstructions++
	builder.declarations[selected] = current
	return nil
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
	if target, ok := builder.declarations[owner]; ok && target.providerCoverage {
		return s.reconstructEnvironmentDeclaration(builder, owner)
	}
	switch selected := owner.(type) {
	case *types.Var:
		return s.reconstructEnvironmentStateField(builder, selected)
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
	case *types.Func, *types.Const, *types.TypeName, *types.Var:
		return true
	default:
		return false
	}
}

func (s *programSession) externalImplementationSourceFact(
	context api.Context,
	function *types.Func,
	statements []tsgo.Statement,
) (api.StatementEmission, bool, error) {
	if s.externalProvider == nil || function == nil {
		return api.StatementEmission{}, false, nil
	}
	target, selected := s.externalFunctionBindings[function.Origin()]
	if !selected || target.Kind() != api.ExternalFunctionTargetModule {
		return api.StatementEmission{}, false, nil
	}
	emission, err := canonicalsourcefact.ExternalImplementation(
		context,
		s.externalProvider,
		target,
		function,
		statements,
	)
	return emission, true, err
}
