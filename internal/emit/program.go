package emit

import (
	"go/ast"
	"go/token"
	"go/types"
	"slices"
	"sort"

	"github.com/tsoniclang/gotots/internal/emit/api"
	namedstructdeclaration "github.com/tsoniclang/gotots/internal/emit/declaration/namedstruct"
	"github.com/tsoniclang/gotots/internal/load"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type TargetFile struct {
	outputPath  string
	packageName string
	sourceFile  tsgo.SourceFile
	kind        TargetFileKind
}

type TargetFileKind uint8

const (
	TargetFileInvalid TargetFileKind = iota
	TargetFileSource
	TargetFileSupport
)

type ProgramEmission struct {
	files []TargetFile
}

type programSession struct {
	source     *load.Program
	factory    tsgo.Factory
	integer    api.IntegerRepresentation
	registry   *declarationRegistry
	scheduler  *scheduler
	companions *companionScheduler
	sites      map[types.Object]declarationSite
	emitters   map[*load.Package]*emitter
	builders   map[string]*targetFileBuilder
}

type targetDeclaration struct {
	object     types.Object
	position   token.Pos
	statements []tsgo.Statement
	companions map[api.CompanionOperation][]tsgo.Statement
}

type targetFileBuilder struct {
	sourcePackage *load.Package
	sourceFile    load.File
	outputPath    string
	emitter       *emitter
	context       api.Context
	placement     *placementOwner
	declarations  []targetDeclaration
	byObject      map[types.Object]struct{}
	indexByObject map[types.Object]int
}

func Compile(source *load.Program, roots []Root) (ProgramEmission, error) {
	return CompileWithOptions(source, roots, DefaultOptions())
}

func CompileWithOptions(
	source *load.Program,
	roots []Root,
	options Options,
) (ProgramEmission, error) {
	if err := options.validate(); err != nil {
		return ProgramEmission{}, err
	}
	if source == nil {
		return ProgramEmission{},
			&ScheduleError{Reason: "source program is nil"}
	}
	if len(roots) == 0 {
		return ProgramEmission{},
			&ScheduleError{Reason: "emission roots are empty"}
	}
	session, err := newProgramSession(source, options)
	if err != nil {
		return ProgramEmission{}, err
	}
	for _, root := range roots {
		if root.object == nil {
			return ProgramEmission{},
				&ScheduleError{Reason: "emission root is invalid"}
		}
	}
	orderedRoots := slices.Clone(roots)
	sort.Slice(orderedRoots, func(left, right int) bool {
		return compareObjects(
			orderedRoots[left].object,
			orderedRoots[right].object,
		) < 0
	})
	for _, root := range orderedRoots {
		if err := session.require(root.object); err != nil {
			return ProgramEmission{}, err
		}
	}
	for {
		if object, ok := session.scheduler.next(); ok {
			if err := session.emit(object); err != nil {
				return ProgramEmission{}, err
			}
			continue
		}
		if companion, ok := session.companions.next(); ok {
			if err := session.emitCompanion(companion); err != nil {
				return ProgramEmission{}, err
			}
			continue
		}
		break
	}
	files, err := session.targetFiles()
	if err != nil {
		return ProgramEmission{}, err
	}
	return ProgramEmission{files: files}, nil
}

func CompileFile(
	sourcePackage *load.Package,
	sourceFile *ast.File,
) (ProgramEmission, error) {
	if sourcePackage == nil || sourcePackage.Program() == nil {
		return ProgramEmission{}, &ScheduleError{Reason: "source package is nil"}
	}
	_, ok := sourcePackage.FileForSyntax(sourceFile)
	if !ok {
		return ProgramEmission{}, &ScheduleError{Reason: "source file is not package-owned"}
	}
	roots, err := fileRoots(sourcePackage, sourceFile)
	if err != nil {
		return ProgramEmission{}, err
	}
	return Compile(sourcePackage.Program(), roots)
}

func fileRoots(
	sourcePackage *load.Package,
	sourceFile *ast.File,
) ([]Root, error) {
	var roots []Root
	for _, declaration := range sourceFile.Decls {
		switch declaration := declaration.(type) {
		case *ast.FuncDecl:
			object := sourcePackage.TypesInfo().Defs[declaration.Name]
			if object == nil {
				return nil, &ScheduleError{
					Object: declaration.Name.Name,
					Reason: "function declaration has no object identity",
				}
			}
			root, err := NewRoot(object)
			if err != nil {
				return nil, err
			}
			roots = append(roots, root)
		case *ast.GenDecl:
			for _, spec := range declaration.Specs {
				switch spec := spec.(type) {
				case *ast.ValueSpec:
					for _, name := range spec.Names {
						object := sourcePackage.TypesInfo().Defs[name]
						if object == nil {
							continue
						}
						root, err := NewRoot(object)
						if err != nil {
							return nil, err
						}
						roots = append(roots, root)
					}
				case *ast.TypeSpec:
					object := sourcePackage.TypesInfo().Defs[spec.Name]
					if object == nil {
						continue
					}
					root, err := NewRoot(object)
					if err != nil {
						return nil, err
					}
					roots = append(roots, root)
				}
			}
		}
	}
	return roots, nil
}

func compareObjects(left types.Object, right types.Object) int {
	leftPackage := ""
	if left != nil && left.Pkg() != nil {
		leftPackage = left.Pkg().Path()
	}
	rightPackage := ""
	if right != nil && right.Pkg() != nil {
		rightPackage = right.Pkg().Path()
	}
	switch {
	case leftPackage < rightPackage:
		return -1
	case leftPackage > rightPackage:
		return 1
	case left.Pos() < right.Pos():
		return -1
	case left.Pos() > right.Pos():
		return 1
	case left.Name() < right.Name():
		return -1
	case left.Name() > right.Name():
		return 1
	default:
		return 0
	}
}

func (e ProgramEmission) Files() []TargetFile {
	return slices.Clone(e.files)
}

func (f TargetFile) OutputPath() string {
	return f.outputPath
}

func (f TargetFile) PackageName() string {
	return f.packageName
}

func (f TargetFile) SourceFile() tsgo.SourceFile {
	return f.sourceFile
}

func (f TargetFile) Kind() TargetFileKind {
	return f.kind
}

func newProgramSession(
	source *load.Program,
	options Options,
) (*programSession, error) {
	sites, err := indexDeclarations(source)
	if err != nil {
		return nil, err
	}
	session := &programSession{
		source:     source,
		factory:    tsgo.NewFactory(),
		integer:    options.IntegerRepresentation,
		registry:   newDeclarationRegistry(),
		scheduler:  newScheduler(),
		companions: newCompanionScheduler(),
		sites:      sites,
		emitters:   make(map[*load.Package]*emitter),
		builders:   make(map[string]*targetFileBuilder),
	}
	for _, sourcePackage := range source.Packages() {
		session.emitters[sourcePackage] = newEmitter(
			sourcePackage,
			session.factory,
			session.registry,
			options.IntegerRepresentation,
			session.require,
		)
	}
	orderedSites := make([]declarationSite, 0, len(sites))
	for _, site := range sites {
		orderedSites = append(orderedSites, site)
	}
	sort.Slice(orderedSites, func(left, right int) bool {
		return compareDeclarationSites(orderedSites[left], orderedSites[right]) < 0
	})
	for _, site := range orderedSites {
		emitter := session.emitters[site.source]
		if emitter == nil {
			return nil, &ScheduleError{
				Object: site.object.Name(),
				Reason: "declaration package has no emitter",
			}
		}
		if _, err := emitter.names.Reserve(
			site.object,
			site.sourceFile.Syntax(),
			site.outputPath,
		); err != nil {
			return nil, err
		}
	}
	return session, nil
}

func (s *programSession) require(object types.Object) error {
	if object == nil {
		return &ScheduleError{Reason: "referenced object is nil"}
	}
	if _, ok := s.sites[object]; !ok {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "object has no supported source declaration",
		}
	}
	s.scheduler.enqueue(object)
	return nil
}

func (s *programSession) emit(object types.Object) error {
	site, ok := s.sites[object]
	if !ok {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "scheduled object lost its declaration",
		}
	}
	builder, err := s.builder(site)
	if err != nil {
		return err
	}
	if _, duplicate := builder.byObject[object]; duplicate {
		return &ScheduleError{
			Object: object.Name(),
			Reason: "object was emitted more than once",
		}
	}
	result, err := builder.emitter.declarationObject(
		builder.context,
		site.declaration,
		object,
	)
	if err != nil {
		return err
	}
	if err := s.applyRequests(builder, result.Requests()); err != nil {
		return err
	}
	builder.byObject[object] = struct{}{}
	builder.indexByObject[object] = len(builder.declarations)
	builder.declarations = append(builder.declarations, targetDeclaration{
		object:     object,
		position:   object.Pos(),
		statements: result.Declarations(),
		companions: make(map[api.CompanionOperation][]tsgo.Statement),
	})
	return nil
}

func (s *programSession) applyRequests(
	builder *targetFileBuilder,
	requests []api.PlacementRequest,
) error {
	imports := make([]api.PlacementRequest, 0, len(requests))
	for _, request := range requests {
		switch request.Kind() {
		case api.PlacementImport:
			imports = append(imports, request)
		case api.PlacementCompanion:
			owner, ok := request.Companion()
			if !ok {
				return &ScheduleError{Reason: "companion request is invalid"}
			}
			s.companions.enqueue(owner)
		default:
			return &ScheduleError{Reason: "placement request kind is invalid"}
		}
	}
	return builder.placement.Apply(imports)
}

func (s *programSession) emitCompanion(owner api.CompanionOwner) error {
	typeName := owner.TypeName()
	site, ok := s.sites[typeName]
	if !ok {
		return &ScheduleError{
			Object: typeName.Name(),
			Reason: "companion owner has no source declaration",
		}
	}
	builder, err := s.builder(site)
	if err != nil {
		return err
	}
	index, ok := builder.indexByObject[typeName]
	if !ok {
		return &ScheduleError{
			Object: typeName.Name(),
			Reason: "companion owner was not emitted first",
		}
	}
	declaration := &builder.declarations[index]
	if _, duplicate := declaration.companions[owner.Operation()]; duplicate {
		return &ScheduleError{
			Object: typeName.Name(),
			Reason: "companion was emitted more than once",
		}
	}
	result, err := namedstructdeclaration.EmitCompanion(
		builder.context,
		builder.emitter,
		site.declaration,
		typeName,
		owner.Operation(),
	)
	if err != nil {
		return err
	}
	if err := s.applyRequests(builder, result.Requests()); err != nil {
		return err
	}
	declaration.companions[owner.Operation()] = result.Declarations()
	return nil
}

func (s *programSession) builder(site declarationSite) (*targetFileBuilder, error) {
	if existing := s.builders[site.outputPath]; existing != nil {
		return existing, nil
	}
	emitter := s.emitters[site.source]
	if emitter == nil {
		return nil, &ScheduleError{
			Object: site.object.Name(),
			Reason: "source package has no emitter",
		}
	}
	context, err := emitter.fileContext(site.sourceFile.Syntax(), site.outputPath)
	if err != nil {
		return nil, err
	}
	builder := &targetFileBuilder{
		sourcePackage: site.source,
		sourceFile:    site.sourceFile,
		outputPath:    site.outputPath,
		emitter:       emitter,
		context:       context,
		placement:     newPlacementOwner(),
		byObject:      make(map[types.Object]struct{}),
		indexByObject: make(map[types.Object]int),
	}
	s.builders[site.outputPath] = builder
	return builder, nil
}

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
		sort.Slice(builder.declarations, func(left, right int) bool {
			if builder.declarations[left].position != builder.declarations[right].position {
				return builder.declarations[left].position < builder.declarations[right].position
			}
			return builder.declarations[left].object.Name() <
				builder.declarations[right].object.Name()
		})
		var declarations []tsgo.Statement
		for _, declaration := range builder.declarations {
			declarations = append(declarations, declaration.statements...)
			for _, operation := range []api.CompanionOperation{
				api.CompanionZero,
				api.CompanionCopy,
				api.CompanionEqual,
			} {
				declarations = append(
					declarations,
					declaration.companions[operation]...,
				)
			}
		}
		statements := append(
			builder.placement.Statements(s.factory),
			declarations...,
		)
		targetPath, err := tsgo.NewPath(outputPath)
		if err != nil {
			return nil, err
		}
		files = append(files, TargetFile{
			outputPath:  outputPath,
			packageName: builder.sourcePackage.Name(),
			kind:        TargetFileSource,
			sourceFile: s.factory.SourceFile(
				statements,
				s.factory.EndOfFile(),
				tsgo.SourceFileData{
					FileName:   targetPath,
					Path:       targetPath,
					ScriptKind: tsgo.ScriptKindTS,
				},
			),
		})
	}
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
		sort.Slice(files, func(left, right int) bool {
			return files[left].outputPath < files[right].outputPath
		})
	}
	return files, nil
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
	targetPath, err := tsgo.NewPath(targetoutput.ScalarSupportPath)
	if err != nil {
		return TargetFile{}, err
	}
	return TargetFile{
		outputPath: targetoutput.ScalarSupportPath,
		kind:       TargetFileSupport,
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
