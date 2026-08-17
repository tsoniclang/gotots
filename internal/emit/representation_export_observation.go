package emit

import (
	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	constantbinding "github.com/tsoniclang/gotots/internal/emit/constant"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"go/ast"
	"go/types"
	"sort"
)

func compareGenericRepresentationRequirements(
	left api.DeclarationRequirement,
	right api.DeclarationRequirement,
) int {
	leftOwner, leftParameter, leftFacet, leftOK :=
		left.GenericRepresentation()
	rightOwner, rightParameter, rightFacet, rightOK :=
		right.GenericRepresentation()
	switch {
	case !leftOK && rightOK:
		return -1
	case leftOK && !rightOK:
		return 1
	case !leftOK:
		return 0
	}
	leftIndex, leftIndexed :=
		api.GenericDeclarationParameterIndex(leftOwner, leftParameter)
	rightIndex, rightIndexed :=
		api.GenericDeclarationParameterIndex(rightOwner, rightParameter)
	switch {
	case !leftIndexed && rightIndexed:
		return -1
	case leftIndexed && !rightIndexed:
		return 1
	case leftIndex < rightIndex:
		return -1
	case leftIndex > rightIndex:
		return 1
	case leftFacet < rightFacet:
		return -1
	case leftFacet > rightFacet:
		return 1
	default:
		return 0
	}
}

func (s *programSession) ResolveGenericRepresentationProfile(
	declaration types.Object,
) (api.GenericRepresentationProfile, bool, error) {
	owner := api.GenericDeclarationOrigin(declaration)
	if owner == nil || len(api.GenericDeclarationParameters(owner)) == 0 {
		return api.GenericRepresentationProfile{}, false, nil
	}
	if _, ok := s.sites[owner]; ok {
		profile, err := api.SelectGenericRepresentationProfile(
			owner,
			s.requirements.SelectedFor(api.MustSourceArtifactOwner(owner)),
		)
		return profile, err == nil, err
	}
	sourcePackage := s.source.EnvironmentForTypes(owner.Pkg())
	if sourcePackage == nil {
		return api.GenericRepresentationProfile{}, false, &ScheduleError{
			Object: owner.Name(),
			Reason: "generic representation owner has no declaration",
		}
	}
	var requirements []api.DeclarationRequirement
	if builder := s.environmentBuilders[sourcePackage]; builder != nil {
		requirements = s.requirements.SelectedFor(
			api.MustSourceArtifactOwner(owner),
		)
	}
	profile, err := api.SelectGenericRepresentationProfile(
		owner,
		requirements,
	)
	return profile, err == nil, err
}

func (s *programSession) recordPackageExport(
	builder *packageTargetBuilder,
	object types.Object,
) error {
	if object == nil || !object.Exported() {
		return nil
	}
	if method, ok := object.(*types.Func); ok && method.Signature().Recv() != nil {
		target, found := s.registry.SourceMethodTargetKind(method)
		if !found {
			return &ScheduleError{
				Object: object.Name(),
				Reason: "source method export has no declaration target",
			}
		}
		if target != api.MethodTargetSourceFunction {
			return nil
		}
	}
	if builder == nil ||
		object.Pkg() != builder.sourcePackage.Types() {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "package export has no exact assembly owner",
		}
	}
	if _, exists := builder.exportObjects[object]; exists {
		return nil
	}
	builder.exportObjects[object] = struct{}{}
	s.packageExports.enqueue(builder)
	return nil
}

func (s *programSession) reconstructPackageExports(
	owner api.ArtifactOwner,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "package exports reconstructed after target files were sealed",
		}
	}
	sourcePackage, ok := owner.PackageAssembly()
	if !ok {
		return &ScheduleError{Reason: "package export owner is invalid"}
	}
	loaded := s.source.PackageForTypes(sourcePackage)
	builder := s.packageBuilders[loaded]
	if builder == nil ||
		builder.assemblyOwner != owner ||
		!builder.exportPublished {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "package export owner has no published assembly",
		}
	}
	return s.publishPackageExports(builder)
}

func (s *programSession) publishPackageExports(
	builder *packageTargetBuilder,
) error {
	objects := make([]types.Object, 0, len(builder.exportObjects))
	for object := range builder.exportObjects {
		objects = append(objects, object)
	}
	sort.Slice(objects, func(left, right int) bool {
		return emitordering.CompareObjects(objects[left], objects[right]) < 0
	})
	byPath := make(map[string]map[string]struct{})
	dependencies := make([]api.ArtifactDependency, 0, len(objects)*2)
	exportPlacement := targetplacement.New()
	var constantExports []tsgo.Statement
	var constantInitialization []tsgo.Statement
	for _, object := range objects {
		owner := api.MustSourceArtifactOwner(object)
		names, ok := s.artifacts.ExportedBindings(owner)
		if !ok {
			return &ScheduleError{
				Object: object.Name(),
				Reason: "assembly export provider has no committed export surface",
			}
		}
		binding, ok := s.registry.Target(object)
		if !ok {
			return &ScheduleError{
				Object: object.Name(),
				Reason: "assembly export has no target binding",
			}
		}
		if byPath[binding.SourcePath] == nil {
			byPath[binding.SourcePath] = make(map[string]struct{})
		}
		selected, deferred := object.(*types.Const)
		deferred = deferred && constantbinding.RequiresDeferredBinding(selected)
		if deferred {
			deferredName, err := constantbinding.DeferredBindingName(binding.Name)
			if err != nil {
				return err
			}
			if len(names) != 1 || names[0] != deferredName {
				return &ScheduleError{
					Object: object.Name(),
					Reason: "deferred constant export surface is invalid",
				}
			}
			modulePath, err := targetoutput.ModuleSpecifier(
				builder.assemblyPath,
				binding.SourcePath,
			)
			if err != nil {
				return err
			}
			request, err := api.NewImportRequest(
				s.factory,
				api.ImportPhaseValue,
				modulePath,
				deferredName,
				deferredName,
			)
			if err != nil {
				return err
			}
			if err := exportPlacement.Apply([]api.RootRequest{request}); err != nil {
				return err
			}
			declaration, initialization := deferredConstantPackageExport(
				s.factory,
				binding.Name,
				deferredName,
			)
			constantExports = append(constantExports, declaration)
			constantInitialization = append(
				constantInitialization,
				initialization,
			)
		}
		for _, name := range names {
			if _, duplicate := byPath[binding.SourcePath][name]; duplicate {
				return &ScheduleError{
					Object: name,
					Reason: "assembly export binding is duplicated",
				}
			}
			byPath[binding.SourcePath][name] = struct{}{}
		}
		dependency, err := api.NewArtifactDependency(
			owner,
			api.ArtifactFacetExportSurface,
		)
		if err != nil {
			return err
		}
		dependencies = append(dependencies, dependency)
		if deferred {
			dependency, err := api.NewArtifactDependency(
				owner,
				api.ArtifactFacetCallableSignature,
			)
			if err != nil {
				return err
			}
			dependencies = append(dependencies, dependency)
		}
	}
	exports, err := s.packageReexports(builder, byPath)
	if err != nil {
		return err
	}
	exports = append(exports, constantExports...)
	nodes := make([]tsgo.Node, 0, len(exports)+len(constantInitialization))
	for _, statement := range exports {
		nodes = append(nodes, statement)
	}
	for _, statement := range constantInitialization {
		nodes = append(nodes, statement)
	}
	contract, err := artifactstate.ProjectFacet(
		api.ArtifactFacetImplementation,
		s.factory.SyntaxList(nodes),
	)
	if err != nil {
		return err
	}
	if err := s.commitArtifactContract(
		builder.assemblyOwner,
		contract,
		dependencies,
	); err != nil {
		return err
	}
	s.artifacts.DiscardDirty(builder.assemblyOwner)
	if builder.exportPublished {
		builder.exportRevisions++
	}
	builder.exportPublished = true
	builder.exportPlacement = exportPlacement
	builder.exportStatements = exports
	builder.constantInitialization = constantInitialization
	return nil
}

func (s *programSession) packageReexports(
	builder *packageTargetBuilder,
	byPath map[string]map[string]struct{},
) ([]tsgo.Statement, error) {
	paths := make([]string, 0, len(byPath))
	for sourcePath := range byPath {
		paths = append(paths, sourcePath)
	}
	sort.Strings(paths)
	exports := make([]tsgo.Statement, 0, len(paths))
	for _, sourcePath := range paths {
		names := make([]string, 0, len(byPath[sourcePath]))
		for name := range byPath[sourcePath] {
			names = append(names, name)
		}
		sort.Strings(names)
		specifiers := make([]tsgo.ExportSpecifier, 0, len(names))
		for _, name := range names {
			specifiers = append(specifiers, s.factory.ExportSpecifier(
				false,
				nil,
				s.factory.Identifier(name),
			))
		}
		modulePath, err := targetoutput.ModuleSpecifier(
			builder.assemblyPath,
			sourcePath,
		)
		if err != nil {
			return nil, err
		}
		exports = append(exports, s.factory.ExportDeclaration(
			nil,
			false,
			s.factory.NamedExports(specifiers),
			s.factory.StringLiteral(modulePath, tsgo.TokenFlagsNone),
			nil,
		))
	}
	return exports, nil
}

func (s *programSession) ObserveRecoveryCallable(
	context api.Context,
	facet api.CallableFacet,
) (api.RecoveryCallableObservation, error) {
	if !facet.Valid() || !context.ArtifactOwner().Valid() {
		return api.RecoveryCallableObservation{}, &ScheduleError{
			Reason: "recovery callable observation is invalid",
		}
	}
	function, ok := recoveryFacetSource(facet)
	if !ok {
		return api.RecoveryCallableObservation{}, &ScheduleError{
			Object: facet.Owner().Name(),
			Reason: "recovery observation has no exact source callable",
		}
	}
	if s.source.EnvironmentForTypes(function.Pkg()) != nil {
		_, selected, err := context.Names().RecoveryCallable(function)
		if err != nil {
			return api.RecoveryCallableObservation{}, err
		}
		return api.NewRecoveryCallableObservation(selected)
	}
	recovery, err := sourceCallableRecoveryRequirement(
		function,
		s.requirements.SelectedFor(api.MustSourceArtifactOwner(function)),
	)
	if err != nil {
		return api.RecoveryCallableObservation{}, err
	}
	var requests []api.RootRequest
	if context.ArtifactOwner() != api.MustSourceArtifactOwner(function) {
		request, requestErr := api.NewOwnedArtifactDependencyRequest(
			api.MustSourceArtifactOwner(function),
			api.ArtifactFacetCallableRecovery,
		)
		if requestErr != nil {
			return api.RecoveryCallableObservation{}, requestErr
		}
		requests = []api.RootRequest{request}
	}
	return api.NewRecoveryCallableObservation(recovery, requests...)
}

func recoveryFacetSource(
	facet api.CallableFacet,
) (*types.Func, bool) {
	if function, ok := facet.SourceFunction(); ok {
		return function.Origin(), true
	}
	return nil, false
}

func sourceCallableRecoveryRequirement(
	function *types.Func,
	requirements []api.DeclarationRequirement,
) (bool, error) {
	owner := api.MustSourceArtifactOwner(function)
	recovery := false
	for _, requirement := range requirements {
		requirementOwner, _, callable, control, ok :=
			requirement.CallableControl()
		if !ok {
			continue
		}
		if requirementOwner != owner {
			return false, &ScheduleError{
				Object: function.Name(),
				Reason: "recovery requirement has foreign ownership",
			}
		}
		if control != api.CallableControlRecovery {
			continue
		}
		if callable == nil {
			recovery = true
			continue
		}
		if _, direct := callable.(*ast.FuncDecl); direct {
			recovery = true
		}
	}
	return recovery, nil
}

type packageExportScheduler struct {
	pending map[*packageTargetBuilder]struct{}
}

func newPackageExportScheduler() *packageExportScheduler {
	return &packageExportScheduler{
		pending: make(map[*packageTargetBuilder]struct{}),
	}
}

func (s *packageExportScheduler) enqueue(builder *packageTargetBuilder) {
	if builder == nil {
		panic("package export builder is nil")
	}
	s.pending[builder] = struct{}{}
}

func (s *packageExportScheduler) nextBatch() []*packageTargetBuilder {
	if len(s.pending) == 0 {
		return nil
	}
	builders := make([]*packageTargetBuilder, 0, len(s.pending))
	for builder := range s.pending {
		builders = append(builders, builder)
	}
	clear(s.pending)
	sort.Slice(builders, func(left, right int) bool {
		return builders[left].assemblyPath < builders[right].assemblyPath
	})
	return builders
}

func (s *packageExportScheduler) hasPending() bool {
	return len(s.pending) != 0
}
