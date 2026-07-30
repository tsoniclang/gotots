package emit

import (
	"go/ast"
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	packagevariable "github.com/tsoniclang/gotots/internal/emit/declaration/packagevariable"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	"github.com/tsoniclang/gotots/internal/emit/storage"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (e *emitter) context(names api.Names) (api.Context, error) {
	context, err := api.NewContext(
		api.RoleFileDeclaration,
		e.source.FileSet(),
		e.source.Types(),
		e.source.TypesInfo(),
		e.source.TypesSizes(),
		e.factory,
		names,
		e.values,
		storage.Owner{},
		e.integer,
		e.order,
		e.concurrency,
	)
	if err != nil {
		return api.Context{}, err
	}
	return context.
		WithGenericCallableResolver(e.generic).
		WithCooperativeCallableResolver(e.cooperative).
		WithGoRuntimeContract(e.goRuntime), nil
}

func (e *emitter) fileContext(
	sourceFile *ast.File,
	targetPath string,
) (api.Context, error) {
	return e.targetContext(sourceFile, targetPath)
}

func (e *emitter) targetContext(
	sourceFile *ast.File,
	targetPath string,
) (api.Context, error) {
	names := e.names.ForFile(
		sourceFile,
		e.source.Types().Scope(),
		e.factory,
		targetPath,
		e.require,
	)
	return e.context(names)
}

func (e *emitter) generatedContext(
	targetPath string,
	registry *emitnaming.Registry,
) (api.Context, error) {
	names := emitnaming.NewOwner(
		nil,
		nil,
		registry,
	).ForFile(
		nil,
		nil,
		e.factory,
		targetPath,
		e.require,
	)
	return e.context(names)
}

func (s *programSession) packageTargetFiles(
	primitiveAliases map[api.PrimitiveAlias]struct{},
) ([]TargetFile, error) {
	builders := make([]*packageTargetBuilder, 0, len(s.packageBuilders))
	for _, builder := range s.packageBuilders {
		builders = append(builders, builder)
	}
	sort.Slice(builders, func(left, right int) bool {
		return builders[left].assemblyPath < builders[right].assemblyPath
	})
	files := make([]TargetFile, 0, len(builders)*2)
	for _, builder := range builders {
		if _, initialized := s.packageInitializations.emitted[builder.sourcePackage]; !initialized {
			return nil, &ScheduleError{
				Object: builder.sourcePackage.Path(),
				Reason: "package assembly was sealed before initialization",
			}
		}
		for _, alias := range builder.statePlacement.PrimitiveAliases() {
			primitiveAliases[alias] = struct{}{}
		}
		for _, alias := range builder.assemblyPlacement.PrimitiveAliases() {
			primitiveAliases[alias] = struct{}{}
		}
		for _, storage := range builder.storage {
			for _, alias := range storage.statePlacement.PrimitiveAliases() {
				primitiveAliases[alias] = struct{}{}
			}
			for _, alias := range storage.assemblyPlacement.PrimitiveAliases() {
				primitiveAliases[alias] = struct{}{}
			}
		}
		if len(builder.storage) != 0 {
			stateFile, err := s.packageStateFile(builder)
			if err != nil {
				return nil, err
			}
			files = append(files, stateFile)
		}
		assemblyFile, err := s.packageAssemblyFile(builder)
		if err != nil {
			return nil, err
		}
		files = append(files, assemblyFile)
	}
	return files, nil
}

func (s *programSession) packageStateFile(
	builder *packageTargetBuilder,
) (TargetFile, error) {
	placement := targetplacement.New()
	if err := placement.Apply(builder.statePlacement.Requests()); err != nil {
		return TargetFile{}, err
	}
	for _, storage := range builder.storage {
		if storage.statePlacement == nil {
			return TargetFile{}, &ScheduleError{
				Object: storage.owner.Name(),
				Reason: "package storage has no committed state placement",
			}
		}
		if err := placement.Apply(storage.statePlacement.Requests()); err != nil {
			return TargetFile{}, err
		}
	}
	if err := placement.RequireTypeOnly(); err != nil {
		return TargetFile{}, err
	}
	sort.Slice(builder.storage, func(left, right int) bool {
		return builder.storage[left].variable.Name() <
			builder.storage[right].variable.Name()
	})
	fields := make([]tsgo.PropertyDeclaration, 0, len(builder.storage))
	for _, storage := range builder.storage {
		fields = append(fields, storage.field)
	}
	declarations, err := packagevariable.StateDeclarations(s.factory, fields)
	if err != nil {
		return TargetFile{}, err
	}
	statements := append(
		placement.Statements(s.factory),
		declarations...,
	)
	return s.sourceFile(
		builder.statePath,
		builder.sourcePackage.Name(),
		TargetFilePackageState,
		statements,
	)
}

func (s *programSession) packageAssemblyFile(
	builder *packageTargetBuilder,
) (TargetFile, error) {
	placement := targetplacement.New()
	if err := placement.Apply(builder.assemblyPlacement.Requests()); err != nil {
		return TargetFile{}, err
	}
	for _, storage := range builder.storage {
		if storage.assemblyPlacement == nil {
			return TargetFile{}, &ScheduleError{
				Object: storage.owner.Name(),
				Reason: "package storage has no committed assembly placement",
			}
		}
		if err := placement.Apply(
			storage.assemblyPlacement.Requests(),
		); err != nil {
			return TargetFile{}, err
		}
	}
	for _, artifact := range builder.initialization {
		if artifact.placement == nil {
			return TargetFile{}, &ScheduleError{
				Object: artifact.owner.Name(),
				Reason: "package initializer has no committed placement",
			}
		}
		if err := placement.Apply(artifact.placement.Requests()); err != nil {
			return TargetFile{}, err
		}
	}
	statements := placement.Statements(s.factory)
	var initialization []tsgo.Statement
	for _, storage := range builder.storage {
		initialization = append(initialization, storage.zeroStatements...)
	}
	for _, artifact := range builder.initialization {
		initialization = append(initialization, artifact.statements...)
	}
	for _, initFunction := range builder.initFunctions {
		cooperative, err := s.sourceCallableIsCooperative(
			initFunction.function,
		)
		if err != nil {
			return TargetFile{}, err
		}
		var call tsgo.Expression = s.factory.CallExpression(
			s.factory.Identifier(initFunction.name),
			nil,
			nil,
			nil,
			tsgo.NodeFlagsNone,
		)
		if cooperative {
			call = s.factory.AwaitExpression(call)
		}
		initialization = append(
			initialization,
			s.factory.ExpressionStatement(call),
		)
	}
	if len(initialization) != 0 {
		cooperative, err := s.packageInitializationIsCooperative(builder)
		if err != nil {
			return TargetFile{}, err
		}
		modifiers := []tsgo.ModifierLike{s.factory.ExportKeyword()}
		var result tsgo.TypeNode = s.factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindVoidKeyword,
		)
		if cooperative {
			modifiers = append(modifiers, s.factory.AsyncKeyword())
			result = callable.PromiseResult(s.factory, result)
		}
		statements = append(statements, s.factory.FunctionDeclaration(
			modifiers,
			nil,
			s.factory.Identifier(packageInitializeName),
			nil,
			nil,
			result,
			s.factory.Block(initialization, true),
		))
	}
	statements = append(statements, builder.exportStatements...)
	if hasExportedPackageVariable(builder.storage) {
		statements = append(statements, s.factory.ExportDeclaration(
			nil,
			false,
			s.factory.NamedExports([]tsgo.ExportSpecifier{
				s.factory.ExportSpecifier(
					false,
					nil,
					s.factory.Identifier(packagevariable.StateValueName),
				),
			}),
			nil,
			nil,
		))
	}
	return s.sourceFile(
		builder.assemblyPath,
		builder.sourcePackage.Name(),
		TargetFilePackageAssembly,
		statements,
	)
}

func (b *packageTargetBuilder) hasInitializationWork() bool {
	return len(b.storage) != 0 ||
		len(b.initialization) != 0 ||
		len(b.initFunctions) != 0
}

func (s *programSession) packageInitializationIsCooperative(
	builder *packageTargetBuilder,
) (bool, error) {
	if builder == nil {
		return false, &ScheduleError{
			Reason: "package initialization owner is nil",
		}
	}
	for _, artifact := range builder.initialization {
		facet, err := api.NewPackageInitializerCallableFacet(artifact.owner)
		if err != nil {
			return false, err
		}
		cooperative, err := s.callableFacetIsCooperative(facet)
		if err != nil {
			return false, err
		}
		if cooperative {
			return true, nil
		}
	}
	for _, initFunction := range builder.initFunctions {
		cooperative, err := s.sourceCallableIsCooperative(
			initFunction.function,
		)
		if err != nil {
			return false, err
		}
		if cooperative {
			return true, nil
		}
	}
	return false, nil
}

func (s *programSession) sourceCallableIsCooperative(
	function *types.Func,
) (bool, error) {
	facet, err := api.NewSourceCallableFacet(function)
	if err != nil {
		return false, err
	}
	return s.callableFacetIsCooperative(facet)
}

func (s *programSession) callableFacetIsCooperative(
	facet api.CallableFacet,
) (bool, error) {
	observation, err := s.ObserveCooperativeCallable(facet.Owner(), facet)
	if err != nil {
		return false, err
	}
	return observation.Cooperative(), nil
}

func (s *programSession) recordPackageExport(
	builder *packageTargetBuilder,
	object types.Object,
) error {
	if object == nil || !object.Exported() {
		return nil
	}
	if method, ok := object.(*types.Func); ok &&
		method.Signature().Recv() != nil {
		return nil
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
	return s.publishPackageExports(builder)
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
	dependencies := make([]api.ArtifactDependency, 0, len(objects))
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
	}
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
			return err
		}
		exports = append(exports, s.factory.ExportDeclaration(
			nil,
			false,
			s.factory.NamedExports(specifiers),
			s.factory.StringLiteral(modulePath, tsgo.TokenFlagsNone),
			nil,
		))
	}
	nodes := make([]tsgo.Node, len(exports))
	for index, statement := range exports {
		nodes[index] = statement
	}
	contract, err := artifactstate.ProjectFacet(
		api.ArtifactFacetImplementation,
		s.factory.SyntaxList(nodes),
	)
	if err != nil {
		return err
	}
	if err := s.artifacts.Commit(
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
	builder.exportStatements = exports
	return nil
}

func hasExportedPackageVariable(storage []packageStorage) bool {
	for _, item := range storage {
		if item.variable.Exported() {
			return true
		}
	}
	return false
}
