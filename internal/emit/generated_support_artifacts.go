package emit

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	providerstatefulrepresentation "github.com/tsoniclang/gotots/internal/emit/declaration/providerstatefulrepresentation"
	"github.com/tsoniclang/gotots/internal/emit/deferredregistry"
	genericconcretization "github.com/tsoniclang/gotots/internal/emit/generic/concretization"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	canonicalsourcefact "github.com/tsoniclang/gotots/internal/emit/sourcefact"
	providerboundary "github.com/tsoniclang/gotots/internal/emit/value/providerboundary"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"go/token"
	"go/types"
)

func (s *programSession) reconstructProviderStatefulArtifact(
	artifact *api.GeneratedArtifact,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "provider stateful representation reconstructed after target files were sealed",
		}
	}
	if err := s.validateProviderStatefulArtifact(artifact); err != nil {
		return err
	}
	builder, err := s.compilationGeneratedArtifactBuilder(
		artifact,
		"provider stateful representation",
	)
	if err != nil {
		return err
	}
	owner := api.MustGeneratedArtifactOwner(artifact)
	index, exists := builder.indexByOwner[owner]
	var temporaryStart emitnaming.TemporarySnapshot
	if exists {
		temporaryStart = builder.declarations[index].temporaryStart
	}
	revision, err := s.buildProviderStatefulRevision(
		builder,
		artifact,
		temporaryStart,
		exists,
	)
	if err != nil {
		return err
	}
	if err := s.commitArtifactRevision(
		owner,
		revision.contract,
		revision.dependencies,
		revision.requestRoots,
	); err != nil {
		return err
	}
	s.artifacts.DiscardDirty(owner)
	declaration := targetDeclaration{
		owner:          owner,
		name:           artifact.TargetName(),
		position:       token.NoPos,
		statements:     revision.statements,
		placement:      revision.placement,
		temporaryStart: revision.temporaryStart,
	}
	if exists {
		declaration.reconstructions =
			builder.declarations[index].reconstructions + 1
		builder.declarations[index] = declaration
		return nil
	}
	builder.byOwner[owner] = struct{}{}
	builder.indexByOwner[owner] = len(builder.declarations)
	builder.declarations = append(builder.declarations, declaration)
	return nil
}

func (s *programSession) buildProviderStatefulRevision(
	builder *targetFileBuilder,
	artifact *api.GeneratedArtifact,
	temporaryStart emitnaming.TemporarySnapshot,
	reconstruction bool,
) (artifactRevision, error) {
	names, ok := builder.context.Names().(*emitnaming.File)
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "provider stateful representation has no concrete name owner",
		}
	}
	if !reconstruction {
		temporaryStart = names.SnapshotTemporaries()
	} else {
		current := names.SnapshotTemporaries()
		names.RestoreTemporaries(temporaryStart)
		defer names.FinishTemporaryReplay(current)
	}
	owner := api.MustGeneratedArtifactOwner(artifact)
	finish, err := names.BeginArtifact(owner, nil, nil, "")
	if err != nil {
		return artifactRevision{}, err
	}
	defer finish()
	requirements := s.requirements.SelectedFor(owner)
	if len(requirements) != 1 {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "provider stateful representation requires exactly one definition request",
		}
	}
	selectedRequirement, ok :=
		requirements[0].ProviderStatefulRepresentation()
	if !ok || selectedRequirement != artifact {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "provider stateful representation received a foreign requirement",
		}
	}
	source, ok := artifact.ProviderStatefulRepresentationType()
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "provider stateful representation source is invalid",
		}
	}
	context := builder.context.WithArtifactOwner(owner)
	selection, profiled, err := providerboundary.ResolveStatefulProfile(
		context,
		source,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	profileKey := ""
	var typeArguments []tsgo.TypeNode
	var requests []api.RootRequest
	if profiled {
		profileKey = selection.Profile().ProfileKey()
		profileContext, profileErr := context.WithProviderProfile(
			selection.Profile().Interfaces(),
		)
		if profileErr != nil {
			return artifactRevision{}, profileErr
		}
		for _, selected := range selection.TypeArguments() {
			represented, handled, representErr :=
				providerboundary.EmitProfileInterfaceType(
					profileContext.WithRole(api.RoleDefinedTypeArgument),
					builder.emitter,
					nil,
					selected,
					true,
				)
			if representErr != nil {
				return artifactRevision{}, representErr
			}
			if !handled {
				return artifactRevision{}, &ScheduleError{
					Object: artifact.TargetName(),
					Reason: "provider stateful-profile type argument is not certified",
				}
			}
			typeArguments = append(typeArguments, represented.Value())
			requests = append(requests, represented.Requests()...)
		}
	}
	typeTarget, err := names.ProviderStatefulProfileTarget(
		source.Obj(),
		profileKey,
		api.ImportPhaseType,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	valueTarget, err := names.ProviderStatefulProfileTarget(
		source.Obj(),
		profileKey,
		api.ImportPhaseValue,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	statements, buildRequests, err := providerstatefulrepresentation.Build(
		context.Factory(),
		artifact.TargetName(),
		typeTarget,
		valueTarget,
		typeArguments,
		[]tsgo.ModifierLike{context.Factory().ExportKeyword()},
	)
	if err != nil {
		return artifactRevision{}, err
	}
	statements, buildRequests, err = canonicalsourcefact.IncludeGeneratedArtifact(
		context,
		artifact,
		statements,
		buildRequests,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	placement, dependencies, nextRequirements, err :=
		s.consumeArtifactRequests(
			owner,
			api.CombineRequests(
				requests,
				selection.Requests(),
				buildRequests,
			),
		)
	if err != nil {
		return artifactRevision{}, err
	}
	contract, err := artifactstate.ProjectContract(s.factory, statements)
	if err != nil {
		return artifactRevision{}, err
	}
	return artifactRevision{
		statements:     statements,
		placement:      placement,
		dependencies:   dependencies,
		requestRoots:   nextRequirements,
		contract:       contract,
		temporaryStart: temporaryStart,
	}, nil
}

func (s *programSession) validateGenericConcretizationArtifact(
	artifact *api.GeneratedArtifact,
) error {
	concretization, ok := artifact.GenericConcretization()
	binding, found := s.registry.GeneratedArtifact(
		api.GeneratedArtifactGenericConcretization,
		artifact.ArtifactKey(),
	)
	bound, boundOK := binding.GenericConcretization()
	if !ok || !found || binding != artifact || !boundOK ||
		bound != concretization ||
		!types.Identical(bound.Signature(), concretization.Signature()) {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generic concretization has no exact canonical binding",
		}
	}
	return nil
}

func (s *programSession) reconstructGenericConcretizationArtifact(
	artifact *api.GeneratedArtifact,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generic concretization reconstructed after target files were sealed",
		}
	}
	if err := s.validateGenericConcretizationArtifact(artifact); err != nil {
		return err
	}
	if artifact.Placement() != api.GeneratedArtifactPlacementCompilation {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "lexical generic concretization must reconstruct through its source artifact",
		}
	}
	builder, err := s.compilationGeneratedArtifactBuilder(
		artifact,
		"generic concretization",
	)
	if err != nil {
		return err
	}
	owner := api.MustGeneratedArtifactOwner(artifact)
	index, exists := builder.indexByOwner[owner]
	var temporaryStart emitnaming.TemporarySnapshot
	if exists {
		temporaryStart = builder.declarations[index].temporaryStart
	}
	revision, err := s.buildGenericConcretizationRevision(
		builder,
		artifact,
		temporaryStart,
		exists,
	)
	if err != nil {
		return err
	}
	if err := s.commitArtifactRevision(
		owner,
		revision.contract,
		revision.dependencies,
		revision.requestRoots,
	); err != nil {
		return err
	}
	s.artifacts.DiscardDirty(owner)
	declaration := targetDeclaration{
		owner:          owner,
		name:           artifact.TargetName(),
		position:       token.NoPos,
		statements:     revision.statements,
		placement:      revision.placement,
		temporaryStart: revision.temporaryStart,
	}
	if exists {
		declaration.reconstructions =
			builder.declarations[index].reconstructions + 1
		builder.declarations[index] = declaration
		return nil
	}
	builder.byOwner[owner] = struct{}{}
	builder.indexByOwner[owner] = len(builder.declarations)
	builder.declarations = append(builder.declarations, declaration)
	return nil
}

func (s *programSession) buildGenericConcretizationRevision(
	builder *targetFileBuilder,
	artifact *api.GeneratedArtifact,
	temporaryStart emitnaming.TemporarySnapshot,
	reconstruction bool,
) (artifactRevision, error) {
	names, ok := builder.context.Names().(*emitnaming.File)
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generic concretization has no concrete name owner",
		}
	}
	if !reconstruction {
		temporaryStart = names.SnapshotTemporaries()
	} else {
		current := names.SnapshotTemporaries()
		names.RestoreTemporaries(temporaryStart)
		defer names.FinishTemporaryReplay(current)
	}
	owner := api.MustGeneratedArtifactOwner(artifact)
	finish, err := names.BeginArtifact(owner, nil, nil, "")
	if err != nil {
		return artifactRevision{}, err
	}
	defer finish()
	context := builder.context.WithArtifactOwner(owner)
	deferred, err := genericconcretization.ExactRequirement(
		artifact,
		s.requirements.SelectedFor(owner),
	)
	if err != nil {
		return artifactRevision{}, err
	}
	statements, requests, err := genericconcretization.Build(
		context,
		builder.emitter,
		artifact,
		[]tsgo.ModifierLike{context.Factory().ExportKeyword()},
		deferred,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	statements, requests, err = canonicalsourcefact.IncludeGeneratedArtifact(
		context,
		artifact,
		statements,
		requests,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	placement, dependencies, requirements, err :=
		s.consumeArtifactRequests(owner, requests)
	if err != nil {
		return artifactRevision{}, err
	}
	contract, err := artifactstate.ProjectContract(s.factory, statements)
	if err != nil {
		return artifactRevision{}, err
	}
	return artifactRevision{
		statements:     statements,
		placement:      placement,
		dependencies:   dependencies,
		requestRoots:   requirements,
		contract:       contract,
		temporaryStart: temporaryStart,
	}, nil
}

func (s *programSession) validateDeferredCallableRegistry(
	artifact *api.GeneratedArtifact,
) error {
	signature, sourceOK := artifact.DeferredCallableRegistry()
	binding, found := s.registry.GeneratedArtifact(
		api.GeneratedArtifactDeferredCallableRegistry,
		artifact.ArtifactKey(),
	)
	bound, boundOK := binding.DeferredCallableRegistry()
	if !sourceOK || !found || binding != artifact || !boundOK ||
		!types.Identical(signature, bound) {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "deferred-callable registry has no exact canonical binding",
		}
	}
	return nil
}

func (s *programSession) reconstructDeferredCallableRegistry(
	artifact *api.GeneratedArtifact,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "deferred-callable registry reconstructed after sealing",
		}
	}
	if err := s.validateDeferredCallableRegistry(artifact); err != nil {
		return err
	}
	builder, err := s.compilationGeneratedArtifactBuilder(
		artifact,
		"deferred-callable registry",
	)
	if err != nil {
		return err
	}
	owner := api.MustGeneratedArtifactOwner(artifact)
	index, exists := builder.indexByOwner[owner]
	var temporaryStart emitnaming.TemporarySnapshot
	if exists {
		temporaryStart = builder.declarations[index].temporaryStart
	}
	revision, err := s.buildDeferredCallableRegistryRevision(
		builder,
		artifact,
		temporaryStart,
		exists,
	)
	if err != nil {
		return err
	}
	if err := s.commitArtifactRevision(
		owner,
		revision.contract,
		revision.dependencies,
		revision.requestRoots,
	); err != nil {
		return err
	}
	s.artifacts.DiscardDirty(owner)
	declaration := targetDeclaration{
		owner:          owner,
		name:           artifact.TargetName(),
		position:       token.NoPos,
		statements:     revision.statements,
		placement:      revision.placement,
		temporaryStart: revision.temporaryStart,
	}
	if exists {
		declaration.reconstructions =
			builder.declarations[index].reconstructions + 1
		builder.declarations[index] = declaration
		return nil
	}
	builder.byOwner[owner] = struct{}{}
	builder.indexByOwner[owner] = len(builder.declarations)
	builder.declarations = append(builder.declarations, declaration)
	return nil
}

func (s *programSession) buildDeferredCallableRegistryRevision(
	builder *targetFileBuilder,
	artifact *api.GeneratedArtifact,
	temporaryStart emitnaming.TemporarySnapshot,
	reconstruction bool,
) (artifactRevision, error) {
	names, ok := builder.context.Names().(*emitnaming.File)
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "deferred-callable registry has no concrete name owner",
		}
	}
	if !reconstruction {
		temporaryStart = names.SnapshotTemporaries()
	} else {
		current := names.SnapshotTemporaries()
		names.RestoreTemporaries(temporaryStart)
		defer names.FinishTemporaryReplay(current)
	}
	owner := api.MustGeneratedArtifactOwner(artifact)
	finish, err := names.BeginArtifact(owner, nil, nil, "")
	if err != nil {
		return artifactRevision{}, err
	}
	defer finish()
	requirements := s.requirements.SelectedFor(owner)
	definitions := 0
	for _, requirement := range requirements {
		if selected, ok := requirement.DeferredCallableRegistry(); ok {
			if selected != artifact {
				return artifactRevision{}, &ScheduleError{
					Object: artifact.TargetName(),
					Reason: "deferred-callable registry received a foreign definition",
				}
			}
			definitions++
			continue
		}
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "deferred-callable registry received a foreign requirement",
		}
	}
	if definitions != 1 {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "deferred-callable registry requires one definition",
		}
	}
	context := builder.context.WithArtifactOwner(owner)
	statement, valueType, requests, err := deferredregistry.Build(
		context,
		builder.emitter,
		artifact,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	statements := []tsgo.Statement{statement}
	statements, requests, err = canonicalsourcefact.IncludeGeneratedArtifact(
		context,
		artifact,
		statements,
		requests,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	placement, dependencies, nextRequirements, err :=
		s.consumeArtifactRequests(owner, requests)
	if err != nil {
		return artifactRevision{}, err
	}
	contract, err := artifactstate.ProjectExplicitValueContract(
		s.factory,
		statement,
		valueType,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	return artifactRevision{
		statements:     statements,
		placement:      placement,
		dependencies:   dependencies,
		requestRoots:   nextRequirements,
		contract:       contract,
		temporaryStart: temporaryStart,
	}, nil
}
