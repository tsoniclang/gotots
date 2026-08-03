package emit

import (
	"go/types"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	artifactstate "github.com/tsoniclang/gotots/internal/emit/artifact"
	constantbinding "github.com/tsoniclang/gotots/internal/emit/constant"
	emitordering "github.com/tsoniclang/gotots/internal/emit/ordering"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

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
			constantExports = append(
				constantExports,
				deferredConstantPackageExport(s.factory, binding.Name, deferredName),
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
	return nil
}

func deferredConstantPackageExport(
	factory tsgo.Factory,
	publicName string,
	deferredName string,
) tsgo.VariableStatement {
	return factory.VariableStatement(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.VariableDeclarationList(
			[]tsgo.VariableDeclaration{factory.VariableDeclaration(
				factory.Identifier(publicName),
				nil,
				nil,
				factory.CallExpression(
					factory.Identifier(deferredName),
					nil,
					nil,
					nil,
					tsgo.NodeFlagsNone,
				),
			)},
			tsgo.NodeFlagsConst,
		),
	)
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
