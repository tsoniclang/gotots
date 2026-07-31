package emit

import (
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	declarationorder "github.com/tsoniclang/gotots/internal/emit/declaration/order"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	runtimeemission "github.com/tsoniclang/gotots/internal/emit/runtime"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (s *programSession) targetFiles() ([]TargetFile, error) {
	if s.sealed {
		return nil, &ScheduleError{Reason: "target files were sealed more than once"}
	}
	if s.scheduler.hasPending() ||
		s.requirements.hasPending() ||
		s.artifacts.HasPending() ||
		s.packageInitializations.hasPending() {
		return nil, &ScheduleError{Reason: "target files sealed with pending work"}
	}
	if err := s.artifacts.VerifyClosure(); err != nil {
		return nil, err
	}
	s.sealed = true
	paths := make([]string, 0, len(s.builders))
	for outputPath := range s.builders {
		paths = append(paths, outputPath)
	}
	sort.Strings(paths)
	files := make([]TargetFile, 0, len(paths)+1)
	primitiveAliases := make(map[api.PrimitiveAlias]struct{})
	runtimeSymbols := make(map[api.RuntimeSymbol]struct{})
	for _, outputPath := range paths {
		builder := s.builders[outputPath]
		placement, err := committedTargetFilePlacement(builder)
		if err != nil {
			return nil, err
		}
		for _, alias := range placement.PrimitiveAliases() {
			primitiveAliases[alias] = struct{}{}
		}
		for _, symbol := range placement.RuntimeSymbols() {
			runtimeSymbols[symbol] = struct{}{}
		}
		orderInput := make(
			[]declarationorder.Declaration,
			len(builder.declarations),
		)
		for index, declaration := range builder.declarations {
			orderInput[index] = declarationorder.Declaration{
				Owner:             declaration.owner,
				Name:              declaration.name,
				Position:          declaration.position,
				EagerDependencies: declaration.eagerDependencies,
			}
		}
		ordered, err := declarationorder.Indices(orderInput)
		if err != nil {
			return nil, err
		}
		var declarations []tsgo.Statement
		for _, index := range ordered {
			declaration := builder.declarations[index]
			declarations = append(
				declarations,
				slices.Clone(declaration.statements)...,
			)
		}
		statements := append(
			placement.Statements(s.factory),
			declarations...,
		)
		kind := TargetFileSource
		packageName := builder.sourcePackage.Name()
		if builder.sourceFile.Syntax() == nil {
			kind = TargetFileSupport
			packageName = ""
		}
		target, err := s.sourceFile(
			outputPath,
			packageName,
			kind,
			statements,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, target)
	}
	environmentFiles, err := s.environmentTargetFiles(
		primitiveAliases,
		runtimeSymbols,
	)
	if err != nil {
		return nil, err
	}
	files = append(files, environmentFiles...)
	packageFiles, err := s.packageTargetFiles(primitiveAliases)
	if err != nil {
		return nil, err
	}
	files = append(files, packageFiles...)
	for _, builder := range s.packageBuilders {
		for _, symbol := range builder.statePlacement.RuntimeSymbols() {
			runtimeSymbols[symbol] = struct{}{}
		}
		for _, symbol := range builder.assemblyPlacement.RuntimeSymbols() {
			runtimeSymbols[symbol] = struct{}{}
		}
		for _, storage := range builder.storage {
			for _, symbol := range storage.statePlacement.RuntimeSymbols() {
				runtimeSymbols[symbol] = struct{}{}
			}
			for _, symbol := range storage.assemblyPlacement.RuntimeSymbols() {
				runtimeSymbols[symbol] = struct{}{}
			}
		}
	}
	programFile, err := s.programInitializationFile()
	if err != nil {
		return nil, err
	}
	files = append(files, programFile)
	aliases := make([]api.PrimitiveAlias, 0, len(primitiveAliases))
	for alias := range primitiveAliases {
		aliases = append(aliases, alias)
	}
	slices.Sort(aliases)
	runtimePackage, err := runtimeemission.AssemblePackage(
		s.factory,
		s.integer,
		runtimeSymbols,
		aliases,
	)
	if err != nil {
		return nil, err
	}
	s.runtimePackage = RuntimePackage{assembled: runtimePackage}
	for _, runtimeFile := range runtimePackage.Files() {
		files = append(files, TargetFile{
			outputPath: runtimeFile.OutputPath(),
			sourceFile: runtimeFile.SourceFile(),
			kind:       TargetFileSupport,
		})
	}
	sort.Slice(files, func(left, right int) bool {
		return files[left].outputPath < files[right].outputPath
	})
	return files, nil
}

func committedTargetFilePlacement(
	builder *targetFileBuilder,
) (*targetplacement.Owner, error) {
	if builder == nil || builder.placement == nil {
		return nil, &ScheduleError{
			Reason: "target file has no root placement owner",
		}
	}
	placement := targetplacement.New()
	if err := placement.Apply(builder.placement.Requests()); err != nil {
		return nil, err
	}
	for _, declaration := range builder.declarations {
		if declaration.placement == nil {
			return nil, &ScheduleError{
				Object: declaration.name,
				Reason: "target artifact has no committed placement",
			}
		}
		if err := placement.Apply(declaration.placement.Requests()); err != nil {
			return nil, err
		}
	}
	return placement, nil
}

func (s *programSession) programInitializationFile() (TargetFile, error) {
	order, err := s.packageInitializationOrder()
	if err != nil {
		return TargetFile{}, err
	}
	placement := targetplacement.New()
	var calls []tsgo.Statement
	for _, builder := range order {
		if !builder.hasInitializationWork() {
			continue
		}
		qualifier := s.registry.ImportQualifier(builder.sourcePackage.Types())
		if qualifier == "" {
			return TargetFile{}, &ScheduleError{
				Object: builder.sourcePackage.Path(),
				Reason: "package initializer has no target qualifier",
			}
		}
		localName := packageInitializeName + "__" + qualifier
		modulePath, err := targetoutput.ModuleSpecifier(
			targetoutput.ProgramInitializationPath,
			builder.assemblyPath,
		)
		if err != nil {
			return TargetFile{}, err
		}
		request, err := api.NewImportRequest(
			s.factory,
			api.ImportPhaseValue,
			modulePath,
			packageInitializeName,
			localName,
		)
		if err != nil {
			return TargetFile{}, err
		}
		if err := placement.Apply([]api.RootRequest{request}); err != nil {
			return TargetFile{}, err
		}
		var call tsgo.Expression = s.factory.CallExpression(
			s.factory.Identifier(localName),
			nil,
			nil,
			nil,
			tsgo.NodeFlagsNone,
		)
		cooperative, err := s.packageInitializationIsCooperative(builder)
		if err != nil {
			return TargetFile{}, err
		}
		if cooperative {
			call = s.factory.AwaitExpression(call)
		}
		calls = append(calls, s.factory.ExpressionStatement(call))
	}
	statements := placement.Statements(s.factory)
	statements = append(statements, calls...)
	return s.sourceFile(
		targetoutput.ProgramInitializationPath,
		"",
		TargetFileProgramInitialization,
		statements,
	)
}

func (s *programSession) packageInitializationOrder() (
	[]*packageTargetBuilder,
	error,
) {
	candidates := make([]*packageTargetBuilder, 0, len(s.packageBuilders))
	for _, builder := range s.packageBuilders {
		candidates = append(candidates, builder)
	}
	sort.Slice(candidates, func(left, right int) bool {
		return candidates[left].sourcePackage.Path() <
			candidates[right].sourcePackage.Path()
	})
	initialized := make(map[*packageTargetBuilder]struct{}, len(candidates))
	order := make([]*packageTargetBuilder, 0, len(candidates))
	for len(order) != len(candidates) {
		var selected *packageTargetBuilder
		for _, candidate := range candidates {
			if _, done := initialized[candidate]; done {
				continue
			}
			ready := true
			for _, imported := range candidate.sourcePackage.Types().Imports() {
				dependency := s.source.PackageForTypes(imported)
				if dependency == nil && s.goRuntime.Owns(imported) {
					continue
				}
				if dependency == nil &&
					s.source.EnvironmentForTypes(imported) != nil {
					continue
				}
				dependencyBuilder := s.packageBuilders[dependency]
				if dependencyBuilder == nil {
					return nil, &ScheduleError{
						Object: imported.Path(),
						Reason: "package initialization dependency is absent",
					}
				}
				if _, done := initialized[dependencyBuilder]; !done {
					ready = false
					break
				}
			}
			if ready {
				selected = candidate
				break
			}
		}
		if selected == nil {
			return nil, &ScheduleError{
				Reason: "package initialization graph has no ready package",
			}
		}
		initialized[selected] = struct{}{}
		order = append(order, selected)
	}
	return order, nil
}

func (s *programSession) sourceFile(
	outputPath string,
	packageName string,
	kind TargetFileKind,
	statements []tsgo.Statement,
) (TargetFile, error) {
	targetPath, err := tsgo.NewPath(outputPath)
	if err != nil {
		return TargetFile{}, err
	}
	return TargetFile{
		outputPath:  outputPath,
		packageName: packageName,
		kind:        kind,
		sourceFile: s.factory.SourceFile(
			statements,
			s.factory.EndOfFile(),
			tsgo.SourceFileData{
				FileName:   targetPath,
				Path:       targetPath,
				ScriptKind: tsgo.ScriptKindTS,
			},
		),
	}, nil
}
