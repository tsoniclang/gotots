package emit

import (
	"go/token"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func (s *programSession) targetFiles() ([]TargetFile, error) {
	paths := make([]string, 0, len(s.builders))
	for outputPath := range s.builders {
		paths = append(paths, outputPath)
	}
	sort.Strings(paths)
	files := make([]TargetFile, 0, len(paths)+1)
	primitiveAliases := make(map[api.PrimitiveAlias]struct{})
	for _, outputPath := range paths {
		builder := s.builders[outputPath]
		for _, alias := range builder.placement.PrimitiveAliases() {
			primitiveAliases[alias] = struct{}{}
		}
		type declarationChunk struct {
			position   token.Pos
			name       string
			statements []tsgo.Statement
		}
		chunks := make(
			[]declarationChunk,
			0,
			len(builder.declarations)+len(builder.packageInitializers),
		)
		for _, declaration := range builder.declarations {
			statements := slices.Clone(declaration.statements)
			for _, operation := range []api.CompanionOperation{
				api.CompanionZero,
				api.CompanionCopy,
				api.CompanionEqual,
			} {
				statements = append(
					statements,
					declaration.companions[operation]...,
				)
			}
			chunks = append(chunks, declarationChunk{
				position:   declaration.position,
				name:       declaration.object.Name(),
				statements: statements,
			})
		}
		for _, declaration := range builder.packageInitializers {
			chunks = append(chunks, declarationChunk{
				position:   declaration.position,
				name:       declaration.name,
				statements: slices.Clone(declaration.statements),
			})
		}
		sort.Slice(chunks, func(left, right int) bool {
			if chunks[left].position != chunks[right].position {
				return chunks[left].position < chunks[right].position
			}
			return chunks[left].name < chunks[right].name
		})
		var declarations []tsgo.Statement
		for _, chunk := range chunks {
			declarations = append(declarations, chunk.statements...)
		}
		statements := append(
			builder.placement.Statements(s.factory),
			declarations...,
		)
		target, err := s.sourceFile(
			outputPath,
			builder.sourcePackage.Name(),
			TargetFileSource,
			statements,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, target)
	}
	packageFiles, err := s.packageTargetFiles(primitiveAliases)
	if err != nil {
		return nil, err
	}
	files = append(files, packageFiles...)
	programFile, err := s.programInitializationFile()
	if err != nil {
		return nil, err
	}
	files = append(files, programFile)
	if len(primitiveAliases) != 0 {
		aliases := make([]api.PrimitiveAlias, 0, len(primitiveAliases))
		for alias := range primitiveAliases {
			aliases = append(aliases, alias)
		}
		slices.Sort(aliases)
		support, err := s.scalarSupportFile(aliases)
		if err != nil {
			return nil, err
		}
		files = append(files, support)
	}
	sort.Slice(files, func(left, right int) bool {
		return files[left].outputPath < files[right].outputPath
	})
	return files, nil
}

func (s *programSession) programInitializationFile() (TargetFile, error) {
	order, err := s.packageInitializationOrder()
	if err != nil {
		return TargetFile{}, err
	}
	placement := newPlacementOwner()
	var calls []tsgo.Statement
	for _, builder := range order {
		if !builder.hasInitializationWork() {
			continue
		}
		qualifier := s.registry.importQualifierByPackage[builder.sourcePackage.Types()]
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
		if err := placement.Apply([]api.PlacementRequest{request}); err != nil {
			return TargetFile{}, err
		}
		calls = append(calls, s.factory.ExpressionStatement(
			s.factory.CallExpression(
				s.factory.Identifier(localName),
				nil,
				nil,
				nil,
				tsgo.NodeFlagsNone,
			),
		))
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

func (s *programSession) scalarSupportFile(
	aliases []api.PrimitiveAlias,
) (TargetFile, error) {
	statements := make([]tsgo.Statement, 0, len(aliases))
	for _, alias := range aliases {
		name, keyword, err := api.PrimitiveAliasRepresentation(
			alias,
			s.integer,
		)
		if err != nil {
			return TargetFile{}, err
		}
		statements = append(statements, s.factory.TypeAliasDeclaration(
			[]tsgo.ModifierLike{s.factory.ExportKeyword()},
			s.factory.Identifier(name),
			nil,
			s.factory.KeywordTypeNode(keyword),
		))
	}
	return s.sourceFile(
		targetoutput.ScalarSupportPath,
		"",
		TargetFileSupport,
		statements,
	)
}
