package emit

import (
	"go/ast"
	"go/types"
	"slices"
	"sort"
	"strconv"

	"github.com/tsoniclang/gotots/internal/emit/api"
	packageinit "github.com/tsoniclang/gotots/internal/emit/declaration/packageinit"
	packagevariable "github.com/tsoniclang/gotots/internal/emit/declaration/packagevariable"
	"github.com/tsoniclang/gotots/internal/load"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

const packageInitializeName = "$initialize"

type packageInitializationScheduler struct {
	queue   []*load.Package
	pending map[*load.Package]struct{}
	emitted map[*load.Package]struct{}
}

type packageStorage struct {
	variable       *types.Var
	field          tsgo.PropertyDeclaration
	zeroStatements []tsgo.Statement
}

type packageTargetBuilder struct {
	sourcePackage     *load.Package
	statePath         string
	assemblyPath      string
	emitter           *emitter
	stateContext      api.Context
	assemblyContext   api.Context
	statePlacement    *placementOwner
	assemblyPlacement *placementOwner
	storage           []packageStorage
	storageByObject   map[*types.Var]struct{}
	initialization    []tsgo.Statement
}

func newPackageInitializationScheduler() *packageInitializationScheduler {
	return &packageInitializationScheduler{
		pending: make(map[*load.Package]struct{}),
		emitted: make(map[*load.Package]struct{}),
	}
}

func (s *packageInitializationScheduler) enqueue(sourcePackage *load.Package) {
	if _, done := s.emitted[sourcePackage]; done {
		return
	}
	if _, queued := s.pending[sourcePackage]; queued {
		return
	}
	s.pending[sourcePackage] = struct{}{}
	s.queue = append(s.queue, sourcePackage)
}

func (s *packageInitializationScheduler) next() (*load.Package, bool) {
	if len(s.queue) == 0 {
		return nil, false
	}
	sourcePackage := s.queue[0]
	s.queue = s.queue[1:]
	delete(s.pending, sourcePackage)
	s.emitted[sourcePackage] = struct{}{}
	return sourcePackage, true
}

func (s *packageInitializationScheduler) hasPending() bool {
	return len(s.queue) != 0 || len(s.pending) != 0
}

func (s *programSession) requirePackage(sourcePackage *load.Package) error {
	if sourcePackage == nil || sourcePackage.Types() == nil {
		return &ScheduleError{Reason: "required package is nil"}
	}
	if s.packageBuilders[sourcePackage] != nil {
		return nil
	}
	emitter := s.emitters[sourcePackage]
	if emitter == nil {
		return &ScheduleError{
			Object: sourcePackage.Path(),
			Reason: "required package has no source emitter",
		}
	}
	statePath, err := targetoutput.PackageStatePath(sourcePackage)
	if err != nil {
		return err
	}
	assemblyPath, err := targetoutput.PackageAssemblyPath(sourcePackage)
	if err != nil {
		return err
	}
	stateContext, err := emitter.targetContext(nil, statePath)
	if err != nil {
		return err
	}
	assemblyContext, err := emitter.targetContext(nil, assemblyPath)
	if err != nil {
		return err
	}
	builder := &packageTargetBuilder{
		sourcePackage:     sourcePackage,
		statePath:         statePath,
		assemblyPath:      assemblyPath,
		emitter:           emitter,
		stateContext:      stateContext,
		assemblyContext:   assemblyContext,
		statePlacement:    newPlacementOwner(),
		assemblyPlacement: newPlacementOwner(),
		storageByObject:   make(map[*types.Var]struct{}),
	}
	s.packageBuilders[sourcePackage] = builder

	imports := slices.Clone(sourcePackage.Types().Imports())
	sort.Slice(imports, func(left, right int) bool {
		return imports[left].Path() < imports[right].Path()
	})
	for _, imported := range imports {
		dependency := s.source.PackageForTypes(imported)
		if dependency == nil || dependency.ModulePath() == "" {
			return &ScheduleError{
				Object: imported.Path(),
				Reason: "package dependency has no source-available assembly",
			}
		}
		if err := s.requirePackage(dependency); err != nil {
			return err
		}
	}

	scope := sourcePackage.Types().Scope()
	names := scope.Names()
	sort.Strings(names)
	for _, name := range names {
		variable, ok := scope.Lookup(name).(*types.Var)
		if !ok {
			continue
		}
		if _, exists := s.sites[variable]; !exists {
			return &ScheduleError{
				Object: variable.Name(),
				Reason: "package variable has no indexed declaration",
			}
		}
		s.scheduler.enqueue(variable)
	}
	s.packageInitializations.enqueue(sourcePackage)
	return nil
}

func (s *programSession) emitPackageStorage(
	variable *types.Var,
	site declarationSite,
) error {
	builder := s.packageBuilders[site.source]
	if builder == nil {
		return &ScheduleError{
			Object: variable.Name(),
			Reason: "package variable owner was not reached",
		}
	}
	if _, duplicate := builder.storageByObject[variable]; duplicate {
		return &ScheduleError{
			Object: variable.Name(),
			Reason: "package variable storage was emitted more than once",
		}
	}
	source, err := packageVariableSyntax(site, variable)
	if err != nil {
		return err
	}
	emission, err := packagevariable.EmitStorage(
		builder.stateContext,
		builder.assemblyContext,
		builder.emitter,
		source,
		variable,
	)
	if err != nil {
		return err
	}
	if err := s.applyPlacementRequests(
		builder.statePlacement,
		emission.StateRequests(),
	); err != nil {
		return err
	}
	if err := s.applyPlacementRequests(
		builder.assemblyPlacement,
		emission.AssemblyRequests(),
	); err != nil {
		return err
	}
	builder.storageByObject[variable] = struct{}{}
	builder.storage = append(builder.storage, packageStorage{
		variable:       variable,
		field:          emission.Field(),
		zeroStatements: emission.ZeroStatements(),
	})
	return nil
}

func packageVariableSyntax(
	site declarationSite,
	variable *types.Var,
) (ast.Node, error) {
	declaration, ok := site.declaration.(*ast.GenDecl)
	if !ok {
		return nil, &ScheduleError{
			Object: variable.Name(),
			Reason: "package variable declaration is not general",
		}
	}
	for _, sourceSpec := range declaration.Specs {
		spec, ok := sourceSpec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for _, name := range spec.Names {
			if site.source.TypesInfo().Defs[name] == variable {
				return name, nil
			}
		}
	}
	return nil, &ScheduleError{
		Object: variable.Name(),
		Reason: "package variable syntax is absent from its declaration",
	}
}

func (s *programSession) emitPackageInitialization(
	sourcePackage *load.Package,
) error {
	builder := s.packageBuilders[sourcePackage]
	if builder == nil {
		return &ScheduleError{
			Object: sourcePackage.Path(),
			Reason: "package initialization has no assembly owner",
		}
	}
	expectedStorage := 0
	scope := sourcePackage.Types().Scope()
	for _, name := range scope.Names() {
		variable, ok := scope.Lookup(name).(*types.Var)
		if !ok {
			continue
		}
		expectedStorage++
		if _, emitted := builder.storageByObject[variable]; !emitted {
			return &ScheduleError{
				Object: variable.Name(),
				Reason: "package variable has no state storage",
			}
		}
	}
	if len(builder.storageByObject) != expectedStorage {
		return &ScheduleError{
			Object: sourcePackage.Path(),
			Reason: "package initialization ran before complete state storage",
		}
	}
	for _, initializer := range sourcePackage.TypesInfo().InitOrder {
		emission, err := packagevariable.EmitInitializer(
			builder.assemblyContext,
			builder.emitter,
			initializer,
		)
		if err != nil {
			return err
		}
		if err := s.applyPlacementRequests(
			builder.assemblyPlacement,
			emission.Requests(),
		); err != nil {
			return err
		}
		builder.initialization = append(
			builder.initialization,
			emission.Statements()...,
		)
	}
	return s.emitPackageInitFunctions(builder)
}

func (s *programSession) emitPackageInitFunctions(
	packageBuilder *packageTargetBuilder,
) error {
	ordinal := 0
	for _, sourceFile := range packageBuilder.sourcePackage.Files() {
		for _, declaration := range sourceFile.Syntax().Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || !isPackageInitDeclaration(function) {
				continue
			}
			outputPath, err := targetoutput.SourcePath(
				packageBuilder.sourcePackage,
				sourceFile,
			)
			if err != nil {
				return err
			}
			sourceBuilder, err := s.builderForFile(
				packageBuilder.sourcePackage,
				sourceFile,
				outputPath,
				"init",
			)
			if err != nil {
				return err
			}
			targetName := "$init_" + strconv.Itoa(ordinal)
			emission, err := packageinit.Emit(
				sourceBuilder.context.WithRole(api.RoleFileDeclaration),
				sourceBuilder.emitter,
				function,
				targetName,
			)
			if err != nil {
				return err
			}
			if err := s.applyRequests(
				sourceBuilder,
				emission.Requests(),
			); err != nil {
				return err
			}
			sourceBuilder.packageInitializers = append(
				sourceBuilder.packageInitializers,
				targetPackageInitDeclaration{
					position:   function.Pos(),
					name:       targetName,
					statements: emission.Declarations(),
				},
			)
			modulePath, err := targetoutput.ModuleSpecifier(
				packageBuilder.assemblyPath,
				outputPath,
			)
			if err != nil {
				return err
			}
			request, err := api.NewImportRequest(
				s.factory,
				api.ImportPhaseValue,
				modulePath,
				targetName,
				targetName,
			)
			if err != nil {
				return err
			}
			if err := packageBuilder.assemblyPlacement.Apply(
				[]api.PlacementRequest{request},
			); err != nil {
				return err
			}
			packageBuilder.initialization = append(
				packageBuilder.initialization,
				s.factory.ExpressionStatement(
					s.factory.CallExpression(
						s.factory.Identifier(targetName),
						nil,
						nil,
						nil,
						tsgo.NodeFlagsNone,
					),
				),
			)
			ordinal++
		}
	}
	return nil
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
	if err := builder.statePlacement.RequireTypeOnly(); err != nil {
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
		builder.statePlacement.Statements(s.factory),
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
	statements := builder.assemblyPlacement.Statements(s.factory)
	var initialization []tsgo.Statement
	for _, storage := range builder.storage {
		initialization = append(initialization, storage.zeroStatements...)
	}
	initialization = append(initialization, builder.initialization...)
	if len(initialization) != 0 {
		statements = append(statements, s.factory.FunctionDeclaration(
			[]tsgo.ModifierLike{s.factory.ExportKeyword()},
			nil,
			s.factory.Identifier(packageInitializeName),
			nil,
			nil,
			s.factory.KeywordTypeNode(
				tsgo.KeywordTypeSyntaxKindVoidKeyword,
			),
			s.factory.Block(initialization, true),
		))
	}
	exports, err := s.packageExports(builder)
	if err != nil {
		return TargetFile{}, err
	}
	statements = append(statements, exports...)
	return s.sourceFile(
		builder.assemblyPath,
		builder.sourcePackage.Name(),
		TargetFilePackageAssembly,
		statements,
	)
}

func (b *packageTargetBuilder) hasInitializationWork() bool {
	return len(b.storage) != 0 || len(b.initialization) != 0
}

func (s *programSession) packageExports(
	builder *packageTargetBuilder,
) ([]tsgo.Statement, error) {
	byPath := make(map[string][]string)
	for _, sourceBuilder := range s.builders {
		if sourceBuilder.sourcePackage != builder.sourcePackage {
			continue
		}
		for _, declaration := range sourceBuilder.declarations {
			if !declaration.object.Exported() {
				continue
			}
			binding, ok := s.registry.byObject[declaration.object]
			if !ok {
				return nil, &ScheduleError{
					Object: declaration.object.Name(),
					Reason: "assembly export has no target binding",
				}
			}
			byPath[binding.sourcePath] = append(
				byPath[binding.sourcePath],
				binding.name,
			)
		}
	}
	paths := make([]string, 0, len(byPath))
	for sourcePath := range byPath {
		paths = append(paths, sourcePath)
	}
	sort.Strings(paths)
	exports := make([]tsgo.Statement, 0, len(paths)+1)
	for _, sourcePath := range paths {
		names := byPath[sourcePath]
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
	if hasExportedPackageVariable(builder.storage) {
		exports = append(exports, s.factory.ExportDeclaration(
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
	return exports, nil
}

func hasExportedPackageVariable(storage []packageStorage) bool {
	for _, item := range storage {
		if item.variable.Exported() {
			return true
		}
	}
	return false
}
