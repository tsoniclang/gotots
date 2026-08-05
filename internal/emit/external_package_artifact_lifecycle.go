package emit

import (
	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/externals"
	externalcertify "github.com/tsoniclang/gotots/internal/contracts/externals/certify"
	gostdlibcertify "github.com/tsoniclang/gotots/internal/contracts/gostdlib/certify"
	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	packagevariable "github.com/tsoniclang/gotots/internal/emit/declaration/packagevariable"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	emitstorage "github.com/tsoniclang/gotots/internal/emit/storage"
	"github.com/tsoniclang/gotots/internal/load"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
	"go/ast"
	"go/types"
	"slices"
	"sort"
)

type indexedExternalFunction struct {
	function *types.Func
	site     declarationSite
	contract environmentcontract.ObjectContract
}

func resolveExternalFunctionProvider(
	source *load.Program,
	sites map[types.Object]declarationSite,
	provider *externalcertify.Certificate,
	standardLibrary *gostdlibcertify.Certificate,
	integer api.IntegerRepresentation,
	concurrency api.ConcurrencySemantics,
) (map[*types.Func]api.ExternalFunctionTarget, []string, error) {
	resolved := make(map[*types.Func]api.ExternalFunctionTarget)
	if provider == nil {
		return resolved, nil, nil
	}
	if err := validateExternalProviderProfile(
		source,
		provider,
		standardLibrary,
		integer,
		concurrency,
	); err != nil {
		return nil, nil, err
	}
	bindings := provider.Bindings()
	requested := externalFunctionIdentities(bindings)
	byIdentity, err := indexExternalFunctions(sites, requested)
	if err != nil {
		return nil, nil, err
	}
	modules := make([]string, 0)
	for _, binding := range bindings {
		owner, selected := byIdentity[binding.SourceIdentity()]
		if !selected {
			continue
		}
		if err := validateExternalSourceBinding(owner, binding); err != nil {
			return nil, nil, err
		}
		target, module, err := externalFunctionTarget(
			owner,
			binding,
			byIdentity,
		)
		if err != nil {
			return nil, nil, err
		}
		if _, duplicate := resolved[owner.function]; duplicate {
			return nil, nil, &ExternalFunctionBindingError{
				Identity: binding.SourceIdentity(),
				Reason:   "certificate binding is duplicated",
			}
		}
		resolved[owner.function] = target
		if module != "" {
			modules = append(modules, module)
		}
	}
	sort.Strings(modules)
	return resolved, slices.Compact(modules), nil
}

func validateExternalProviderProfile(
	source *load.Program,
	provider *externalcertify.Certificate,
	standardLibrary *gostdlibcertify.Certificate,
	integer api.IntegerRepresentation,
	concurrency api.ConcurrencySemantics,
) error {
	if source == nil || !provider.Valid() {
		return &ExternalFunctionBindingError{Reason: "provider certificate is invalid"}
	}
	if standardLibrary == nil || !standardLibrary.Valid() ||
		provider.StandardLibraryDigest() != standardLibrary.ManifestDigest() {
		return &ExternalFunctionBindingError{
			Reason: "provider and standard-library certificates do not exact-join",
		}
	}
	selectedProfile, ok := provider.BuildProfile()
	if !ok {
		return &ExternalFunctionBindingError{Reason: "provider build profile is absent"}
	}
	selectedKey, err := environmentcontract.ToolchainKey(selectedProfile)
	if err != nil {
		return err
	}
	sourceKey, err := environmentcontract.ToolchainKey(source.BuildProfile())
	if err != nil {
		return err
	}
	if selectedKey != sourceKey || provider.Backend() != "node" ||
		provider.IntegerRepresentation() != integer.String() ||
		provider.ConcurrencySemantics() != concurrency.String() {
		return &ExternalFunctionBindingError{
			Reason: "provider target profile does not match compilation",
		}
	}
	return nil
}

func indexExternalFunctions(
	sites map[types.Object]declarationSite,
	requested map[string]struct{},
) (map[string]indexedExternalFunction, error) {
	result := make(map[string]indexedExternalFunction)
	for object, site := range sites {
		function, ok := object.(*types.Func)
		if !ok || function != function.Origin() {
			continue
		}
		contract, err := environmentcontract.Describe(function)
		if err != nil {
			return nil, err
		}
		if _, selected := requested[contract.Identity()]; !selected {
			continue
		}
		if _, duplicate := result[contract.Identity()]; duplicate {
			return nil, &ExternalFunctionBindingError{
				Identity: contract.Identity(),
				Reason:   "selected source identity is duplicated",
			}
		}
		result[contract.Identity()] = indexedExternalFunction{
			function: function,
			site:     site,
			contract: contract,
		}
	}
	return result, nil
}

func externalFunctionIdentities(bindings []externals.Binding) map[string]struct{} {
	result := make(map[string]struct{}, len(bindings)*2)
	for _, binding := range bindings {
		result[binding.SourceIdentity()] = struct{}{}
		if identity, _, _, ok := binding.SourceTarget(); ok {
			result[identity] = struct{}{}
		}
	}
	return result
}

func validateExternalSourceBinding(
	owner indexedExternalFunction,
	binding externals.Binding,
) error {
	declaration, ok := owner.site.Declaration.(*ast.FuncDecl)
	if !ok || declaration.Body != nil ||
		owner.contract.Signature() != binding.SourceSignature() ||
		owner.site.Source.ModulePath() != binding.SourceModulePath() ||
		owner.site.Source.ModuleVersion() != binding.SourceModuleVersion() {
		return &ExternalFunctionBindingError{
			Identity: binding.SourceIdentity(),
			Reason:   "certificate does not match the selected bodyless declaration",
		}
	}
	return nil
}

func externalFunctionTarget(
	owner indexedExternalFunction,
	binding externals.Binding,
	byIdentity map[string]indexedExternalFunction,
) (api.ExternalFunctionTarget, string, error) {
	switch binding.TargetKind() {
	case externals.TargetModule:
		module, export, _, _, ok := binding.ModuleTarget()
		if !ok {
			break
		}
		target, err := api.NewExternalModuleFunctionTarget(module, export)
		return target, module, err
	case externals.TargetSource:
		identity, signature, _, ok := binding.SourceTarget()
		implementation, found := byIdentity[identity]
		declaration, bodyOwner := implementation.site.Declaration.(*ast.FuncDecl)
		if !ok || !found || !bodyOwner || declaration.Body == nil ||
			implementation.contract.Signature() != signature ||
			implementation.function.Pkg() != owner.function.Pkg() ||
			!types.Identical(implementation.function.Type(), owner.function.Type()) {
			return api.ExternalFunctionTarget{}, "", &ExternalFunctionBindingError{
				Identity: binding.SourceIdentity(),
				Reason:   "portable source implementation is absent or incompatible",
			}
		}
		target, err := api.NewExternalSourceFunctionTarget(implementation.function)
		return target, "", err
	}
	return api.ExternalFunctionTarget{}, "", &ExternalFunctionBindingError{
		Identity: binding.SourceIdentity(),
		Reason:   "certificate target is invalid",
	}
}

func (s *programSession) emitPackageInitializer(
	builder *packageTargetBuilder,
	initializer *types.Initializer,
) error {
	anchor, anchored := packageInitializerAnchor(initializer)
	if !anchored {
		return &ScheduleError{
			Object: builder.sourcePackage.Path(),
			Reason: "package initializer has no source declaration anchor",
		}
	}
	site, ok := s.sites[anchor]
	if !ok || site.Source != builder.sourcePackage {
		return &ScheduleError{
			Object: anchor.Name(),
			Reason: "package initializer has no exact source declaration",
		}
	}
	artifactOwner, err := api.PackageInitializerArtifactOwner(
		builder.sourcePackage.Types(),
		initializer,
	)
	if err != nil {
		return err
	}
	if _, duplicate := builder.initializerByOwner[artifactOwner]; duplicate {
		return &ScheduleError{
			Object: artifactOwner.Name(),
			Reason: "package initializer artifact was emitted more than once",
		}
	}
	revision, err := s.buildPackageInitializerRevision(
		builder,
		initializer,
		site,
		artifactOwner,
		nil,
		false,
	)
	if err != nil {
		return err
	}
	if err := s.commitArtifactRevision(
		artifactOwner,
		revision.contract,
		revision.dependencies,
		revision.requirements,
	); err != nil {
		return err
	}
	builder.initializerByOwner[artifactOwner] = len(builder.initialization)
	builder.initialization = append(
		builder.initialization,
		packageInitializationArtifact{
			owner:          artifactOwner,
			initializer:    initializer,
			site:           site,
			statements:     revision.statements,
			placement:      revision.placement,
			temporaryStart: revision.temporaryStart,
		},
	)
	return nil
}

func packageInitializerAnchor(
	initializer *types.Initializer,
) (*types.Var, bool) {
	if initializer == nil {
		return nil, false
	}
	for _, variable := range initializer.Lhs {
		if variable != nil {
			return variable, true
		}
	}
	return nil, false
}

func (s *programSession) buildPackageInitializerRevision(
	builder *packageTargetBuilder,
	initializer *types.Initializer,
	site declarationSite,
	owner api.ArtifactOwner,
	temporaryStart emitnaming.TemporarySnapshot,
	reconstruction bool,
) (artifactRevision, error) {
	names, ok := builder.assemblyContext.Names().(*emitnaming.File)
	if !ok {
		return artifactRevision{}, &ScheduleError{
			Object: owner.Name(),
			Reason: "package initializer has no concrete name owner",
		}
	}
	if !reconstruction {
		temporaryStart = names.SnapshotTemporaries()
	} else {
		current := names.SnapshotTemporaries()
		names.RestoreTemporaries(temporaryStart)
		defer names.RestoreTemporaries(current)
	}
	sourcePath, err := targetoutput.SourcePath(
		site.Source,
		site.SourceFile,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	finish, err := names.BeginArtifact(
		owner,
		site.Declaration,
		site.SourceFile.Syntax(),
		sourcePath,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	defer finish()
	requirements := s.requirements.appliedFor(owner)
	context, err := emitnaming.WithLexicalTypeRequirements(
		builder.assemblyContext.WithArtifactOwner(owner),
		site.Declaration,
		owner,
		requirements,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	context, err = emitstorage.ApplyRequirements(
		context,
		initializer.Rhs,
		owner,
		requirements,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	context, err = context.WithCallableControls(
		owner,
		initializer.Rhs,
		requirements,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	callableFacet, err := api.NewPackageInitializerCallableFacet(owner)
	if err != nil {
		return artifactRevision{}, err
	}
	observation, err := context.ObserveCooperativeCallable(callableFacet)
	if err != nil {
		return artifactRevision{}, err
	}
	context = context.WithCooperativeCallable(
		callableFacet,
		observation.Cooperative(),
	)
	emission, err := packagevariable.EmitInitializer(
		context,
		builder.emitter,
		initializer,
	)
	if err != nil {
		return artifactRevision{}, err
	}
	placement, dependencies, requirements, err :=
		s.consumeArtifactRequests(
			owner,
			api.CombineRequests(
				emission.Requests(),
				observation.Requests(),
			),
		)
	if err != nil {
		return artifactRevision{}, err
	}
	contract, err := artifactstate.ProjectCoverageContract(s.factory, nil)
	if err != nil {
		return artifactRevision{}, err
	}
	return artifactRevision{
		statements:     emission.Statements(),
		placement:      placement,
		dependencies:   dependencies,
		requirements:   requirements,
		contract:       contract,
		temporaryStart: temporaryStart,
	}, nil
}

func (s *programSession) reconstructPackageInitializer(
	owner api.ArtifactOwner,
) error {
	sourcePackage, initializer, owned := owner.PackageInitializer()
	if !owned {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "dirty package initializer owner is invalid",
		}
	}
	if s.sealed {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "package initializer reconstructed after target files were sealed",
		}
	}
	loaded := s.source.PackageForTypes(sourcePackage)
	builder := s.packageBuilders[loaded]
	if builder == nil {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "dirty package initializer lost its package builder",
		}
	}
	index, ok := builder.initializerByOwner[owner]
	if !ok {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "dirty package initializer was not emitted first",
		}
	}
	artifact := &builder.initialization[index]
	if artifact.initializer != initializer {
		return &ScheduleError{
			Object: owner.Name(),
			Reason: "dirty package initializer identity changed",
		}
	}
	revision, err := s.buildPackageInitializerRevision(
		builder,
		artifact.initializer,
		artifact.site,
		owner,
		artifact.temporaryStart,
		true,
	)
	if err != nil {
		return err
	}
	if err := s.commitArtifactRevision(
		owner,
		revision.contract,
		revision.dependencies,
		revision.requirements,
	); err != nil {
		return err
	}
	s.artifacts.DiscardDirty(owner)
	artifact.statements = revision.statements
	artifact.placement = revision.placement
	artifact.reconstructions++
	return nil
}

func (s *programSession) retireCompilationGeneratedArtifact(
	artifact *api.GeneratedArtifact,
) error {
	if s.sealed {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "generated artifact retired after target files were sealed",
		}
	}
	if err := s.validateGeneratedArtifact(artifact); err != nil {
		return err
	}
	if artifact.Placement() != api.GeneratedArtifactPlacementCompilation {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "non-compilation generated artifact cannot be retired as a target file",
		}
	}
	owner := api.MustGeneratedArtifactOwner(artifact)
	builder := s.builders[artifact.OutputPath()]
	if builder == nil {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "retired generated artifact has no materialized target file",
		}
	}
	index, exists := builder.indexByOwner[owner]
	if !exists || index != 0 ||
		len(builder.declarations) != 1 ||
		builder.declarations[0].owner != owner {
		return &ScheduleError{
			Object: artifact.TargetName(),
			Reason: "retired generated artifact has no exact materialized target file",
		}
	}
	contract, err := artifactstate.ProjectCoverageContract(s.factory, nil)
	if err != nil {
		return err
	}
	if err := s.commitArtifactRevision(owner, contract, nil, nil); err != nil {
		return err
	}
	s.artifacts.DiscardDirty(owner)
	delete(s.builders, artifact.OutputPath())
	return nil
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
