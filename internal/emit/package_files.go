package emit

import (
	"fmt"
	"go/ast"
	"go/types"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/contracts/sourceimplementation"
	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	declarationindex "github.com/tsoniclang/gotots/internal/emit/declaration/index"
	packagevariable "github.com/tsoniclang/gotots/internal/emit/declaration/packagevariable"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	"github.com/tsoniclang/gotots/internal/emit/sourcepackage"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (s *programSession) replaceSourceImplementations(
	files []TargetFile,
	generatedContracts []sourceimplementation.Target,
) ([]TargetFile, error) {
	if s.sourceImplementations == nil {
		return nil, &ScheduleError{
			Reason: "source implementation replacement has no certificate",
		}
	}
	replaced := slices.Clone(files)
	contractPackages := make(
		[]sourceimplementation.PackageTarget,
		0,
		len(s.sourceImplementations.Implementations()),
	)
	for _, implementation := range s.sourceImplementations.Implementations() {
		sourcePackage := s.source.PackageByPath(implementation.PackagePath())
		paths, err := sourcepackage.ResolvePaths(sourcePackage)
		if err != nil {
			return nil, sourceImplementationError(implementation.PackagePath(), err)
		}
		contractPackage, err := sourceimplementation.NewPackageTarget(
			implementation.PackagePath(),
			paths.AssemblyPath(),
		)
		if err != nil {
			return nil, sourceImplementationError(implementation.PackagePath(), err)
		}
		contractPackages = append(contractPackages, contractPackage)
		consumers := make([]sourcepackage.Consumer, len(replaced))
		for index, file := range replaced {
			consumers[index] = sourcepackage.Consumer{
				OutputPath: file.outputPath,
				SourceFile: file.sourceFile,
			}
		}
		consumers, err = sourcepackage.RebindConsumers(
			s.factory,
			paths,
			implementation,
			consumers,
		)
		if err != nil {
			return nil, sourceImplementationError(implementation.PackagePath(), err)
		}
		for index := range replaced {
			replaced[index].sourceFile = consumers[index].SourceFile
		}
		filtered := make([]TargetFile, 0, len(replaced))
		assemblyFound := false
		for _, file := range replaced {
			if !paths.Owns(file.outputPath) {
				filtered = append(filtered, file)
				continue
			}
			if file.outputPath != paths.AssemblyPath() {
				continue
			}
			if assemblyFound || file.kind != TargetFilePackageAssembly {
				return nil, sourceImplementationError(
					implementation.PackagePath(),
					fmt.Errorf("generated package assembly ownership is invalid"),
				)
			}
			assemblyFound = true
		}
		if !assemblyFound {
			return nil, sourceImplementationError(
				implementation.PackagePath(),
				fmt.Errorf("generated package assembly is absent"),
			)
		}
		filtered = append(filtered, TargetFile{
			outputPath:  paths.AssemblyPath(),
			packageName: sourcePackage.Name(),
			sourceFile:  implementation.SourceFile(),
			kind:        TargetFileSourceImplementation,
		})
		for _, module := range implementation.PrivateModules() {
			outputPath, _ := paths.SourcePath(module.GoFile())
			filtered = append(filtered, TargetFile{
				outputPath:  outputPath,
				packageName: sourcePackage.Name(),
				sourceFile:  module.SourceFile(),
				kind:        TargetFileSourceImplementation,
			})
		}
		replaced = filtered
	}
	installedContracts, err := sourceImplementationTargets(replaced)
	if err != nil {
		return nil, err
	}
	if err := s.sourceImplementations.VerifyGeneratedContracts(
		generatedContracts,
		installedContracts,
		contractPackages,
	); err != nil {
		return nil, sourceImplementationError("generated contract", err)
	}
	return replaced, nil
}

func sourceImplementationTargets(
	files []TargetFile,
) ([]sourceimplementation.Target, error) {
	result := make([]sourceimplementation.Target, len(files))
	for index, file := range files {
		var err error
		result[index], err = sourceimplementation.NewTarget(
			file.outputPath,
			file.sourceFile,
		)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func sourceImplementationError(packagePath string, cause error) error {
	return &ScheduleError{
		Object: packagePath,
		Reason: "install source implementation: " + cause.Error(),
	}
}

func sourceArtifactOwnerOrder(
	sites map[types.Object]declarationSite,
) func(api.ArtifactOwner, api.ArtifactOwner) int {
	return func(left api.ArtifactOwner, right api.ArtifactOwner) int {
		leftObject, leftSource := left.Source()
		rightObject, rightSource := right.Source()
		if leftSource && rightSource {
			leftSite, leftIndexed := sites[leftObject]
			rightSite, rightIndexed := sites[rightObject]
			if leftIndexed && rightIndexed {
				if order := declarationindex.CompareSites(
					leftSite,
					rightSite,
				); order != 0 {
					return order
				}
			}
		}
		return emitordering.CompareArtifactOwners(left, right)
	}
}

func (e *emitter) context(names api.Names) (api.Context, error) {
	byteOrder, err := e.source.Program().BuildProfile().ByteOrder()
	if err != nil {
		return api.Context{}, err
	}
	context, err := api.NewContext(
		api.RoleFileDeclaration,
		e.source.FileSet(),
		e.source.Types(),
		e.source.TypesInfo(),
		e.source.TypesSizes(),
		byteOrder,
		e.factory,
		names,
		e.values,
		e.scalar.IntegerRepresentation(),
		e.order,
		e.concurrency,
	)
	if err != nil {
		return api.Context{}, err
	}
	if e.providerScalar.Valid() {
		context = context.WithProviderScalarABI(e.providerScalar)
	}
	context = context.
		WithGenericCallableResolver(e.generic).
		WithCooperativeCallableResolver(e.cooperative).
		WithRecoveryCallableResolver(e.recovery).
		WithExternalFunctionResolver(e.external).
		WithGoRuntimeContract(e.goRuntime)
	if e.implementationContract {
		context = context.WithSourceImplementationContract()
	}
	return context, nil
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
	names, err := e.names.ForFile(
		sourceFile,
		e.source.Types().Scope(),
		e.factory,
		targetPath,
		e.observer,
	)
	if err != nil {
		return api.Context{}, err
	}
	return e.context(names)
}

func (e *emitter) generatedContext(
	targetPath string,
	registry *emitnaming.Registry,
) (api.Context, error) {
	names, err := emitnaming.NewOwner(
		nil,
		nil,
		registry,
	).ForFile(
		nil,
		nil,
		e.factory,
		targetPath,
		e.observer,
	)
	if err != nil {
		return api.Context{}, err
	}
	return e.context(names)
}

func (s *programSession) packageTargetFiles(
	requirements *targetRequirements,
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
		if len(builder.storage) != 0 {
			stateFile, err := s.packageStateFile(builder, requirements)
			if err != nil {
				return nil, err
			}
			files = append(files, stateFile)
		}
		assemblyFile, err := s.packageAssemblyFile(builder, requirements)
		if err != nil {
			return nil, err
		}
		files = append(files, assemblyFile)
	}
	return files, nil
}

func (s *programSession) packageStateFile(
	builder *packageTargetBuilder,
	requirements *targetRequirements,
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
	requirements.observe(placement)
	storage := slices.Clone(builder.storage)
	sort.Slice(storage, func(left, right int) bool {
		return storage[left].variable.Name() <
			storage[right].variable.Name()
	})
	fields := make([]tsgo.PropertyDeclaration, 0, len(storage))
	for _, item := range storage {
		fields = append(fields, item.field)
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
	requirements *targetRequirements,
) (TargetFile, error) {
	placement := targetplacement.New()
	if err := placement.Apply(builder.assemblyPlacement.Requests()); err != nil {
		return TargetFile{}, err
	}
	if err := placement.Apply(builder.exportPlacement.Requests()); err != nil {
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
	requirements.observe(placement)
	statements := placement.Statements(s.factory)
	initialization := slices.Clone(builder.constantInitialization)
	for _, storage := range builder.storage {
		initialization = append(
			initialization,
			storage.initializationStatements...,
		)
	}
	for _, artifact := range builder.initialization {
		if len(artifact.statements) == 0 {
			continue
		}
		initialization = append(
			initialization,
			s.factory.Block(artifact.statements, true),
		)
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
	if len(initialization) != 0 || builder.implementationInit {
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

func deferredConstantPackageExport(
	factory tsgo.Factory,
	publicName string,
	deferredName string,
) (tsgo.VariableStatement, tsgo.ExpressionStatement) {
	declaration := factory.VariableStatement(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{factory.VariableDeclaration(
				factory.Identifier(publicName),
				nil,
				factory.TypeReferenceNode(
					factory.Identifier("ReturnType"),
					[]tsgo.TypeNode{factory.TypeQueryNode(
						factory.Identifier(deferredName),
						nil,
					)},
				),
				nil,
			)},
			tsgo.NodeFlagsLet,
		),
	)
	initialization := factory.ExpressionStatement(factory.BinaryExpression(
		nil,
		factory.Identifier(publicName),
		nil,
		factory.BinaryOperatorToken(tsgo.BinaryOperatorEqualsToken),
		factory.CallExpression(
			factory.Identifier(deferredName),
			nil,
			nil,
			nil,
			tsgo.NodeFlagsNone,
		),
	))
	return declaration, initialization
}

func (b *packageTargetBuilder) hasInitializationWork() bool {
	return b.implementationInit ||
		len(b.constantInitialization) != 0 ||
		len(b.storage) != 0 ||
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
	observation, err := s.observeCooperativeCallable(
		facet.Owner(),
		nil,
		facet,
	)
	if err != nil {
		return false, err
	}
	return observation.Cooperative(), nil
}

func hasExportedPackageVariable(storage []packageStorage) bool {
	for _, item := range storage {
		if item.variable.Exported() {
			return true
		}
	}
	return false
}
