package emit

import (
	"fmt"
	"go/ast"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/contracts/sourceimplementation"
	"github.com/tsoniclang/gotots/internal/emit/api"
	packagevariable "github.com/tsoniclang/gotots/internal/emit/declaration/packagevariable"
	emitnaming "github.com/tsoniclang/gotots/internal/emit/naming"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	"github.com/tsoniclang/gotots/internal/emit/sourcepackage"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (s *programSession) replaceSourceImplementations(
	files []TargetFile,
	generatedContracts []sourceimplementation.Target,
) ([]TargetFile, sourceimplementation.GeneratedContractPlan, error) {
	if s.sourceImplementations == nil {
		return nil, sourceimplementation.GeneratedContractPlan{}, &ScheduleError{
			Reason: "source implementation replacement has no certificate",
		}
	}
	type selectedImplementation struct {
		implementation sourceimplementation.Implementation
		sourcePackage  *load.Package
		paths          sourcepackage.Paths
	}
	selected := make([]selectedImplementation, 0, len(s.sourceImplementations.Implementations()))
	contractPackages := make(
		[]sourceimplementation.PackageTarget,
		0,
		len(s.sourceImplementations.Implementations()),
	)
	ownedPaths := make(map[string]string)
	assemblyPaths := make(map[string]string)
	for _, implementation := range s.sourceImplementations.Implementations() {
		sourcePackage := s.source.PackageByPath(implementation.PackagePath())
		paths, err := sourcepackage.ResolvePaths(sourcePackage)
		if err != nil {
			return nil, sourceimplementation.GeneratedContractPlan{}, sourceImplementationError(implementation.PackagePath(), err)
		}
		contractPackage, err := sourceimplementation.NewPackageTarget(
			implementation.PackagePath(),
			paths.AssemblyPath(),
		)
		if err != nil {
			return nil, sourceimplementation.GeneratedContractPlan{}, sourceImplementationError(implementation.PackagePath(), err)
		}
		contractPackages = append(contractPackages, contractPackage)
		assemblyPaths[implementation.PackagePath()] = paths.AssemblyPath()
		selected = append(selected, selectedImplementation{
			implementation: implementation,
			sourcePackage:  sourcePackage,
			paths:          paths,
		})
		for _, outputPath := range paths.OwnedPaths() {
			if owner := ownedPaths[outputPath]; owner != "" {
				return nil, sourceimplementation.GeneratedContractPlan{}, sourceImplementationError(
					implementation.PackagePath(),
					fmt.Errorf("target path %q is also owned by %q", outputPath, owner),
				)
			}
			ownedPaths[outputPath] = implementation.PackagePath()
		}
	}
	replaced := slices.Clone(files)
	consumers := make([]sourcepackage.Consumer, len(replaced))
	for index, file := range replaced {
		consumers[index] = sourcepackage.Consumer{
			OutputPath: file.outputPath,
			SourceFile: file.sourceFile,
		}
	}
	for _, replacement := range selected {
		var err error
		consumers, err = sourcepackage.RebindConsumers(
			s.factory,
			replacement.paths,
			replacement.implementation,
			consumers,
		)
		if err != nil {
			return nil, sourceimplementation.GeneratedContractPlan{}, sourceImplementationError(
				replacement.implementation.PackagePath(),
				err,
			)
		}
	}
	for index := range replaced {
		replaced[index].sourceFile = consumers[index].SourceFile
	}
	assemblyFound := make(map[string]bool, len(selected))
	filtered := make([]TargetFile, 0, len(replaced))
	for _, file := range replaced {
		packagePath := ownedPaths[file.outputPath]
		if packagePath == "" {
			filtered = append(filtered, file)
			continue
		}
		if file.outputPath != assemblyPaths[packagePath] {
			continue
		}
		if assemblyFound[packagePath] || file.kind != TargetFilePackageAssembly {
			return nil, sourceimplementation.GeneratedContractPlan{}, sourceImplementationError(
				packagePath,
				fmt.Errorf("generated package assembly ownership is invalid"),
			)
		}
		assemblyFound[packagePath] = true
	}
	for _, replacement := range selected {
		implementation := replacement.implementation
		if !assemblyFound[implementation.PackagePath()] {
			return nil, sourceimplementation.GeneratedContractPlan{}, sourceImplementationError(
				implementation.PackagePath(),
				fmt.Errorf("generated package assembly is absent"),
			)
		}
		filtered = append(filtered, TargetFile{
			outputPath:  replacement.paths.AssemblyPath(),
			packageName: replacement.sourcePackage.Name(),
			sourceFile:  implementation.SourceFile(),
			kind:        TargetFileSourceImplementation,
		})
		for _, module := range implementation.PrivateModules() {
			outputPath, ok := replacement.paths.SourcePath(module.GoFile())
			if !ok {
				return nil, sourceimplementation.GeneratedContractPlan{}, sourceImplementationError(
					implementation.PackagePath(),
					fmt.Errorf("private module %q lost its source identity", module.GoFile()),
				)
			}
			filtered = append(filtered, TargetFile{
				outputPath:  outputPath,
				packageName: replacement.sourcePackage.Name(),
				sourceFile:  module.SourceFile(),
				kind:        TargetFileSourceImplementation,
			})
		}
	}
	replaced = filtered
	installedContracts, err := sourceImplementationTargets(replaced)
	if err != nil {
		return nil, sourceimplementation.GeneratedContractPlan{}, err
	}
	verification, err := s.sourceImplementations.PlanGeneratedContracts(
		generatedContracts,
		installedContracts,
		contractPackages,
	)
	if err != nil {
		return nil, sourceimplementation.GeneratedContractPlan{}, sourceImplementationError("generated contract", err)
	}
	return replaced, verification, nil
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
	)
	if err != nil {
		return api.Context{}, err
	}
	if e.providerScalar.Valid() {
		context = context.WithProviderScalarABI(e.providerScalar)
	}
	context = context.
		WithGenericCallableResolver(e.generic).
		WithDeclarationDemandResolver(e.declarationDemands).
		WithRecoveryCallableResolver(e.recovery).
		WithExternalFunctionResolver(e.external).
		WithCallableImplementationResolver(e.callableImplementations).
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
	if _, ok := e.source.FileForSyntax(sourceFile); !ok {
		return api.Context{}, &ScheduleError{
			Reason: "file context source file is not package-owned",
		}
	}
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
		call := s.factory.CallExpression(
			s.factory.Identifier(initFunction.name),
			nil,
			nil,
			nil,
			tsgo.NodeFlagsNone,
		)
		initialization = append(
			initialization,
			s.factory.ExpressionStatement(call),
		)
	}
	if len(initialization) != 0 || builder.implementationInit {
		modifiers := []tsgo.ModifierLike{s.factory.ExportKeyword()}
		result := s.factory.KeywordTypeNode(
			tsgo.KeywordTypeSyntaxKindVoidKeyword,
		)
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
	return b.implementationInit ||
		len(b.constantInitialization) != 0 ||
		len(b.storage) != 0 ||
		len(b.initialization) != 0 ||
		len(b.initFunctions) != 0
}
