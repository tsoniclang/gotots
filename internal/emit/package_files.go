package emit

import (
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	constantbinding "github.com/tsoniclang/gotots/internal/emit/constant"
	packagevariable "github.com/tsoniclang/gotots/internal/emit/declaration/packagevariable"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

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
	placement := newPlacementOwner()
	if err := placement.Apply(builder.assemblyPlacement.Requests()); err != nil {
		return TargetFile{}, err
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
	initialization = append(initialization, builder.initFunctions...)
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
	return len(b.storage) != 0 ||
		len(b.initialization) != 0 ||
		len(b.initFunctions) != 0
}

func (s *programSession) exportedBindingNames(
	object types.Object,
	baseName string,
) ([]string, error) {
	constant, ok := object.(*types.Const)
	if !ok || !constantbinding.IsUntyped(constant.Type()) {
		return []string{baseName}, nil
	}
	requirements := s.requirements.appliedFor(
		api.MustSourceArtifactOwner(constant),
	)
	names := make([]string, 0, len(requirements))
	for _, requirement := range requirements {
		_, projection, ok := requirement.ConstantProjection()
		if !ok {
			continue
		}
		name, err := api.ConstantProjectionName(baseName, projection)
		if err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, nil
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
			object, sourceOwned := declaration.owner.Source()
			if !sourceOwned || !object.Exported() {
				continue
			}
			binding, ok := s.registry.byObject[object]
			if !ok {
				return nil, &ScheduleError{
					Object: object.Name(),
					Reason: "assembly export has no target binding",
				}
			}
			names, err := s.exportedBindingNames(object, binding.name)
			if err != nil {
				return nil, err
			}
			byPath[binding.sourcePath] = append(
				byPath[binding.sourcePath],
				names...,
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
