package emit

import (
	"go/token"
	"go/types"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	anonymousstructdeclaration "github.com/tsoniclang/gotots/internal/emit/declaration/namedstruct"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	"github.com/tsoniclang/gotots/internal/emit/typescriptclass"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (s *programSession) validateAnonymousStructArtifact(
	artifact *api.GeneratedArtifact,
) error {
	if !artifact.Valid() {
		return &ScheduleError{Reason: "anonymous-struct artifact is invalid"}
	}
	structType, structural := artifact.StructType()
	if !structural {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generated artifact is not an anonymous struct",
		}
	}
	binding, ok := s.registry.GeneratedArtifact(
		api.GeneratedArtifactAnonymousStruct,
		artifact.ArtifactKey(),
	)
	if !ok ||
		binding != artifact ||
		!types.Identical(
			binding.SourceType(),
			structType,
		) {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "anonymous-struct artifact has no exact canonical binding",
		}
	}
	return nil
}

func (s *programSession) reconstructAnonymousStruct(
	artifact *api.GeneratedArtifact,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "anonymous struct reconstructed after target files were sealed",
		}
	}
	if err := s.validateAnonymousStructArtifact(artifact); err != nil {
		return err
	}
	if artifact.Placement() !=
		api.GeneratedArtifactPlacementCompilation {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "lexical anonymous struct must reconstruct through its source artifact",
		}
	}
	builder, err := s.anonymousStructBuilder()
	if err != nil {
		return err
	}
	owner := api.MustGeneratedArtifactOwner(artifact)
	index, exists := builder.indexByOwner[owner]
	var temporaryStart emitnaming.TemporarySnapshot
	if exists {
		temporaryStart = builder.declarations[index].temporaryStart
	}
	revision, err := s.buildAnonymousStructRevision(
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

func (s *programSession) buildAnonymousStructRevision(
	builder *targetFileBuilder,
	artifact *api.GeneratedArtifact,
	temporaryStart emitnaming.TemporarySnapshot,
	reconstruction bool,
) (artifactRevision, error) {
	names, ok := builder.context.Names().(*emitnaming.File)
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "anonymous struct has no concrete name owner",
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

	operations, representationFacets, err :=
		anonymousstructdeclaration.SelectAnonymousRequirements(
			context.Role(),
			artifact,
			s.requirements.SelectedFor(owner),
		)
	if err != nil {
		return artifactRevision{}, err
	}
	structType, ok := artifact.StructType()
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generated artifact is not an anonymous struct",
		}
	}
	emission, err := anonymousstructdeclaration.EmitAnonymous(
		context,
		builder.emitter,
		structType,
		artifact.TargetName(),
		operations,
		representationFacets,
		true,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	placement, dependencies, requirements, err :=
		s.consumeArtifactRequests(
			owner,
			emission.Requests(),
		)
	if err != nil {
		return artifactRevision{}, err
	}
	statements := emission.Declarations()
	contract, err := artifactstate.ProjectContract(
		s.factory,
		statements,
	)
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

func (s *programSession) anonymousStructBuilder() (
	*targetFileBuilder,
	error,
) {
	if existing := s.builders[output.AnonymousStructSupportPath]; existing != nil {
		return existing, nil
	}
	sourcePackage, ok := deterministicSupportPackage(s.source.Packages())
	if !ok {
		return nil, &ScheduleError{
			Reason: "anonymous-struct support has no deterministic source context",
		}
	}
	emitter := s.emitters[sourcePackage]
	if emitter == nil {
		return nil, &ScheduleError{
			Reason: "anonymous-struct support package has no emitter",
		}
	}
	context, err := emitter.generatedContext(
		output.AnonymousStructSupportPath,
		s.registry,
	)
	if err != nil {
		return nil, err
	}
	builder := &targetFileBuilder{
		sourcePackage: sourcePackage,
		outputPath:    output.AnonymousStructSupportPath,
		emitter:       emitter,
		context:       context,
		placement:     targetplacement.New(),
		byOwner:       make(map[api.ArtifactOwner]struct{}),
		indexByOwner:  make(map[api.ArtifactOwner]int),
	}
	s.builders[output.AnonymousStructSupportPath] = builder
	return builder, nil
}

func deterministicSupportPackage(
	sourcePackages []*load.Package,
) (*load.Package, bool) {
	packages := append([]*load.Package(nil), sourcePackages...)
	sort.Slice(packages, func(left, right int) bool {
		return packages[left].Path() < packages[right].Path()
	})
	for _, sourcePackage := range packages {
		if len(sourcePackage.Files()) != 0 {
			return sourcePackage, true
		}
	}
	return nil, false
}

type classMemberContribution struct {
	owner   *types.TypeName
	method  *types.Func
	members []tsgo.ClassElement
}

func (s *programSession) artifactTargetSite(
	site declarationSite,
) (declarationSite, error) {
	method, ok := site.Object.(*types.Func)
	if !ok || method.Signature().Recv() == nil {
		return site, nil
	}
	owner := api.MethodReceiverTypeName(method)
	target, ok := s.sites[owner]
	if owner == nil || !ok {
		return declarationSite{}, &ScheduleError{
			Object: site.Object.Name(),
			Reason: "receiver method has no target class declaration",
		}
	}
	return target, nil
}

func (s *programSession) commitClassMemberContribution(
	owner types.Object,
	contribution *classMemberContribution,
) {
	method, ok := owner.(*types.Func)
	if !ok {
		return
	}
	method = method.Origin()
	if contribution == nil {
		delete(s.classMembers, method)
		return
	}
	s.classMembers[method] = classMemberContribution{
		owner:   contribution.owner,
		method:  contribution.method,
		members: slices.Clone(contribution.members),
	}
}

func (s *programSession) partitionClassMethodRequirements(
	owner types.Object,
	requirements []api.DeclarationRequirement,
) ([]api.DeclarationRequirement, []*types.Func, error) {
	ordinary := make([]api.DeclarationRequirement, 0, len(requirements))
	methods := make([]*types.Func, 0, len(requirements))
	typeName, typeOwner := owner.(*types.TypeName)
	for _, requirement := range requirements {
		if requirement.Kind() != api.DeclarationRequirementClassMethod {
			ordinary = append(ordinary, requirement)
			continue
		}
		selectedOwner, method, ok := requirement.ClassMethod()
		if !ok || !typeOwner || selectedOwner != typeName {
			return nil, nil, &ScheduleError{
				Object: owner.Name(),
				Reason: "class-method requirement has a foreign class owner",
			}
		}
		methods = append(methods, method)
	}
	sort.Slice(methods, func(left, right int) bool {
		return emitordering.CompareObjects(methods[left], methods[right]) < 0
	})
	return ordinary, methods, nil
}

func (s *programSession) classArtifactRequests(
	owner types.Object,
	methods []*types.Func,
	requests []api.RootRequest,
) ([]api.RootRequest, error) {
	if _, ok := owner.(*types.TypeName); !ok {
		if len(methods) != 0 {
			return nil, &ScheduleError{
				Object: owner.Name(),
				Reason: "non-class artifact selected class methods",
			}
		}
		return requests, nil
	}
	result := slices.Clone(requests)
	for _, method := range methods {
		request, err := api.NewArtifactDependencyRequest(
			method,
			api.ArtifactFacetImplementation,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, request)
	}
	return result, nil
}

func (s *programSession) attachClassMemberContributions(
	builder *targetFileBuilder,
	owner types.Object,
	statements []tsgo.Statement,
	methods []*types.Func,
) ([]tsgo.Statement, error) {
	typeName, ok := owner.(*types.TypeName)
	if !ok {
		return nil, &ScheduleError{
			Object: owner.Name(),
			Reason: "class-member attachment owner is not a type",
		}
	}
	className, err := builder.context.Names().Declare(typeName)
	if err != nil {
		return nil, err
	}
	members := make([]tsgo.ClassElement, 0, len(methods))
	for _, method := range methods {
		contribution, ok := s.classMembers[method]
		if !ok ||
			contribution.owner != typeName ||
			contribution.method != method ||
			len(contribution.members) == 0 {
			return nil, &ScheduleError{
				Object: method.Name(),
				Reason: "selected class method has no exact contribution",
			}
		}
		members = append(members, contribution.members...)
	}
	result := slices.Clone(statements)
	found := false
	for index, statement := range result {
		class, ok := statement.(tsgo.ClassDeclaration)
		if !ok || class.Name().Text() != className {
			continue
		}
		if found {
			return nil, &ScheduleError{
				Object: owner.Name(),
				Reason: "target class declaration is duplicated",
			}
		}
		found = true
		targetMembers := append(class.Members(), members...)
		result[index] = typescriptclass.Declaration(s.factory,
			class.Modifiers(),
			class.Name(),
			class.TypeParameters(),
			class.HeritageClauses(),
			targetMembers,
		)
	}
	if !found {
		return nil, &ScheduleError{
			Object: owner.Name(),
			Reason: "class-member owner emitted no target class",
		}
	}
	return result, nil
}
