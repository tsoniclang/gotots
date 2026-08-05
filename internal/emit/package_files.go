package emit

import (
	"go/ast"
	"go/types"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	"github.com/tsoniclang/gotots/internal/emit/callable"
	declarationindex "github.com/tsoniclang/gotots/internal/emit/declaration/index"
	packagevariable "github.com/tsoniclang/gotots/internal/emit/declaration/packagevariable"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	"github.com/tsoniclang/gotots/internal/emit/storage"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

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
		storage.Owner{},
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
	return context.
		WithGenericCallableResolver(e.generic).
		WithCooperativeCallableResolver(e.cooperative).
		WithRecoveryCallableResolver(e.recovery).
		WithExternalFunctionResolver(e.external).
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
	return len(b.constantInitialization) != 0 ||
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
