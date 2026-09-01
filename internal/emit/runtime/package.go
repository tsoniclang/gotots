package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/contracts/tsoniccore"
	"github.com/tsoniclang/gotots/internal/emit/api"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
	"github.com/tsoniclang/gotots/internal/emit/runtime/sourcefact"
	targetoutput "github.com/tsoniclang/gotots/internal/output"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

type PackageFile struct {
	outputPath string
	sourceFile tsgo.SourceFile
}

func (f PackageFile) OutputPath() string {
	return f.outputPath
}

func (f PackageFile) SourceFile() tsgo.SourceFile {
	return f.sourceFile
}

type Package struct {
	files       []PackageFile
	manifest    []byte
	fingerprint string
	scalar      api.ScalarABI
	valid       bool
}

func (p Package) Valid() bool {
	return p.valid
}

func (p Package) Name() string {
	if !p.valid {
		return ""
	}
	return targetoutput.RuntimePackageName
}

func (p Package) Version() string {
	if !p.valid {
		return ""
	}
	return targetoutput.RuntimePackageVersion
}

func (p Package) RootPath() string {
	if !p.valid {
		return ""
	}
	return targetoutput.RuntimePackageRootPath
}

func (p Package) ManifestPath() string {
	if !p.valid {
		return ""
	}
	return targetoutput.RuntimePackageManifestPath
}

func (p Package) Profile() api.IntegerRepresentation {
	return p.scalar.IntegerRepresentation()
}

func (p Package) NativeIntegerWidth() api.NativeIntegerWidth {
	return p.scalar.NativeIntegerWidth()
}

func (p Package) Files() []PackageFile {
	return slices.Clone(p.files)
}

func (p Package) Manifest() []byte {
	return slices.Clone(p.manifest)
}

func (p Package) Fingerprint() string {
	return p.fingerprint
}

func AssemblePackage(
	factory tsgo.Factory,
	scalar api.ScalarABI,
	requested map[api.RuntimeSymbol]struct{},
	aliases []api.PrimitiveAlias,
) (Package, error) {
	return assemblePackage(
		factory,
		scalar,
		requested,
		aliases,
	)
}

func AssembleProviderCertificationPackage(
	factory tsgo.Factory,
	scalar api.ScalarABI,
	requested map[api.RuntimeSymbol]struct{},
	aliases []api.PrimitiveAlias,
) (Package, error) {
	return assemblePackage(
		factory,
		scalar,
		requested,
		aliases,
	)
}

func assemblePackage(
	factory tsgo.Factory,
	scalar api.ScalarABI,
	requested map[api.RuntimeSymbol]struct{},
	aliases []api.PrimitiveAlias,
) (Package, error) {
	if !scalar.Valid() {
		return Package{}, &AssemblyError{
			Reason: "runtime package scalar ABI is invalid",
		}
	}
	aliases = slices.Clone(aliases)
	slices.Sort(aliases)
	for index, alias := range aliases {
		if _, _, err := api.PrimitiveAliasRepresentation(alias, scalar); err != nil {
			return Package{}, err
		}
		if index != 0 && alias == aliases[index-1] {
			return Package{}, &AssemblyError{
				Reason: "runtime package contains a duplicate primitive alias",
			}
		}
	}
	requested = maps.Clone(requested)
	if requested == nil {
		requested = make(map[api.RuntimeSymbol]struct{})
	}
	primitives := make([]sourcefact.Primitive, len(aliases))
	companionCount := 0
	for index, alias := range aliases {
		primitive, err := sourcefact.DescribePrimitive(alias, scalar)
		if err != nil {
			return Package{}, err
		}
		primitives[index] = primitive
		if primitive.RequiresCompanion() {
			companionCount++
		}
	}
	if companionCount != 0 {
		requested[api.RuntimeSourceBasicFact] = struct{}{}
	}
	closed, err := dependencyClosure(requested)
	if err != nil {
		return Package{}, err
	}
	for symbol := range closed {
		for _, fact := range sourcefact.FactSymbols(symbol) {
			requested[fact] = struct{}{}
		}
	}
	closed, err = dependencyClosure(requested)
	if err != nil {
		return Package{}, err
	}
	if len(closed) == 0 && len(aliases) == 0 {
		return Package{}, nil
	}
	byModule := make(map[api.RuntimeModule][]api.RuntimeSymbol)
	paths := make(map[api.RuntimeModule]string)
	for symbol := range closed {
		contract, err := api.RuntimeContract(symbol)
		if err != nil {
			return Package{}, err
		}
		module := contract.Module()
		if existing := paths[module]; existing != "" &&
			existing != contract.OutputPath() {
			return Package{}, &AssemblyError{
				Module: module,
				Symbol: symbol,
				Reason: "one runtime module has multiple output paths",
			}
		}
		paths[module] = contract.OutputPath()
		byModule[module] = append(byModule[module], symbol)
	}
	modules := make([]api.RuntimeModule, 0, len(byModule))
	for module := range byModule {
		modules = append(modules, module)
	}
	sort.Slice(modules, func(left, right int) bool {
		return modules[left] < modules[right]
	})
	files := make([]PackageFile, 0, len(modules)+1)
	statements := make([]tsgo.Statement, 0, len(aliases)*2+2)
	aliasDeclarations := make([]tsgo.Statement, 0, len(aliases))
	for index, alias := range aliases {
		name, keyword, err := api.PrimitiveAliasRepresentation(alias, scalar)
		if err != nil {
			return Package{}, err
		}
		underlying := tsgo.TypeNode(factory.KeywordTypeNode(keyword))
		if shared, selected, err := primitives[index].SharedDeclaration(); err != nil {
			return Package{}, err
		} else if selected {
			underlying = factory.TypeReferenceNode(
				factory.Identifier(sharedPrimitiveLocalName(shared.Export())),
				nil,
			)
		}
		declaration := factory.TypeAliasDeclaration(
			[]tsgo.ModifierLike{factory.ExportKeyword()},
			factory.Identifier(name),
			nil,
			underlying,
		)
		aliasDeclarations = append(aliasDeclarations, declaration)
		statements = append(statements, declaration)
	}
	if len(statements) != 0 {
		imports, err := scalarSourceFactImports(
			factory,
			primitives,
			companionCount != 0,
		)
		if err != nil {
			return Package{}, err
		}
		annotations, err := scalarSourceFactAnnotations(
			factory,
			aliases,
			primitives,
			aliasDeclarations,
		)
		if err != nil {
			return Package{}, err
		}
		statements = append(imports, append(statements, annotations...)...)
		file, err := packageSourceFile(
			factory,
			targetoutput.ScalarSupportPath,
			statements,
		)
		if err != nil {
			return Package{}, err
		}
		files = append(files, file)
	}
	for _, module := range modules {
		symbols := byModule[module]
		symbols, err = orderModuleSymbols(module, symbols)
		if err != nil {
			return Package{}, err
		}
		definitions, err := Build(
			factory,
			module,
			symbols,
		)
		if err != nil {
			return Package{}, err
		}
		statements, err := exactDefinitions(module, symbols, definitions)
		if err != nil {
			return Package{}, err
		}
		annotations, err := sourceFactAnnotations(
			factory,
			module,
			symbols,
			statements,
		)
		if err != nil {
			return Package{}, err
		}
		statements = append(statements, annotations...)
		imports, err := moduleImports(
			factory,
			paths[module],
			module,
			symbols,
		)
		if err != nil {
			return Package{}, err
		}
		file, err := packageSourceFile(
			factory,
			paths[module],
			append(imports, statements...),
		)
		if err != nil {
			return Package{}, err
		}
		files = append(files, file)
	}
	sort.Slice(files, func(left, right int) bool {
		return files[left].outputPath < files[right].outputPath
	})
	manifest, fingerprint, err := packageManifest(scalar, files)
	if err != nil {
		return Package{}, err
	}
	return Package{
		files:       files,
		manifest:    manifest,
		fingerprint: fingerprint,
		scalar:      scalar,
		valid:       true,
	}, nil
}

func scalarSourceFactImports(
	factory tsgo.Factory,
	primitives []sourcefact.Primitive,
	companions bool,
) ([]tsgo.Statement, error) {
	placement := targetplacement.New()
	var requests []api.RootRequest
	for _, primitive := range primitives {
		declaration, selected, err := primitive.SharedDeclaration()
		if err != nil {
			return nil, err
		}
		if !selected {
			continue
		}
		request, err := api.NewImportRequest(
			factory,
			api.ImportPhaseType,
			declaration.Module(),
			declaration.Export(),
			sharedPrimitiveLocalName(declaration.Export()),
		)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	if companions {
		attribute, err := tsoniccore.Resolve(tsoniccore.SymbolAttribute)
		if err != nil {
			return nil, err
		}
		attributeRequest, err := api.NewImportRequest(
			factory,
			api.ImportPhaseValue,
			attribute.Module(),
			attribute.Export(),
			attribute.Export(),
		)
		if err != nil {
			return nil, err
		}
		fact, err := api.RuntimeContract(api.RuntimeSourceBasicFact)
		if err != nil {
			return nil, err
		}
		factRequest, err := api.NewRuntimeImportRequest(
			factory,
			api.ImportPhaseValue,
			"./source-fact.js",
			api.RuntimeSourceBasicFact,
			fact.ExportedName(),
		)
		if err != nil {
			return nil, err
		}
		requests = append(requests, attributeRequest, factRequest)
	}
	if err := placement.Apply(requests); err != nil {
		return nil, err
	}
	return placement.Statements(factory), nil
}

func sharedPrimitiveLocalName(exportedName string) string {
	return "$go$core$" + exportedName
}

func scalarSourceFactAnnotations(
	factory tsgo.Factory,
	aliases []api.PrimitiveAlias,
	primitives []sourcefact.Primitive,
	declarations []tsgo.Statement,
) ([]tsgo.Statement, error) {
	if len(aliases) != len(primitives) || len(aliases) != len(declarations) {
		return nil, &AssemblyError{Reason: "primitive fact denominator is not exact"}
	}
	fact, err := api.RuntimeContract(api.RuntimeSourceBasicFact)
	if err != nil {
		return nil, err
	}
	annotations := make([]tsgo.Statement, 0, len(aliases))
	for index, alias := range aliases {
		primitive := primitives[index]
		if !primitive.RequiresCompanion() {
			continue
		}
		name, err := api.PrimitiveAliasName(alias)
		if err != nil {
			return nil, err
		}
		arguments, err := sourcefact.PrimitiveArguments(factory, primitive)
		if err != nil {
			return nil, err
		}
		annotation, err := sourcefact.AnnotationWithArguments(
			factory,
			name,
			fact.ExportedName(),
			declarations[index],
			arguments...,
		)
		if err != nil {
			return nil, err
		}
		annotations = append(annotations, annotation)
	}
	return annotations, nil
}

func sourceFactAnnotations(
	factory tsgo.Factory,
	module api.RuntimeModule,
	symbols []api.RuntimeSymbol,
	statements []tsgo.Statement,
) ([]tsgo.Statement, error) {
	if module == api.RuntimeModuleSourceFact {
		return nil, nil
	}
	if len(symbols) != len(statements) {
		return nil, &AssemblyError{
			Module: module,
			Reason: "source-fact annotation input is not exact",
		}
	}
	annotations := make([]tsgo.Statement, 0, len(symbols))
	operationFact, err := api.RuntimeContract(api.RuntimeSourceOperationFact)
	if err != nil {
		return nil, err
	}
	for index, symbol := range symbols {
		fact, ok := sourcefact.FactSymbol(symbol)
		if !ok {
			continue
		}
		contract, err := api.RuntimeContract(symbol)
		if err != nil {
			return nil, err
		}
		factContract, err := api.RuntimeContract(fact)
		if err != nil {
			return nil, err
		}
		identity, err := sourcefact.Identity(symbol)
		if err != nil {
			return nil, err
		}
		annotation, err := sourcefact.Annotation(
			factory,
			contract.ExportedName(),
			factContract.ExportedName(),
			identity,
			uint16(symbol),
			statements[index],
		)
		if err != nil {
			return nil, err
		}
		annotations = append(annotations, annotation)
		memberAnnotations, err := sourcefact.MemberAnnotations(
			factory,
			contract.ExportedName(),
			operationFact.ExportedName(),
			identity,
			uint16(symbol),
			statements[index],
		)
		if err != nil {
			return nil, err
		}
		annotations = append(annotations, memberAnnotations...)
	}
	return annotations, nil
}

func packageSourceFile(
	factory tsgo.Factory,
	outputPath string,
	statements []tsgo.Statement,
) (PackageFile, error) {
	targetPath, err := tsgo.NewPath(outputPath)
	if err != nil {
		return PackageFile{}, err
	}
	return PackageFile{
		outputPath: outputPath,
		sourceFile: factory.SourceFile(
			statements,
			factory.EndOfFile(),
			tsgo.SourceFileData{
				FileName:   targetPath,
				Path:       targetPath,
				ScriptKind: tsgo.ScriptKindTS,
			},
		),
	}, nil
}

type packageMetadata struct {
	IntegerRepresentation string `json:"integerRepresentation"`
	NativeIntegerBits     uint8  `json:"nativeIntegerBits"`
}

type packageDocument struct {
	Name    string            `json:"name"`
	Version string            `json:"version"`
	Private bool              `json:"private"`
	Type    string            `json:"type"`
	GoToTS  packageMetadata   `json:"gotots"`
	Exports map[string]string `json:"exports"`
}

func packageManifest(
	scalar api.ScalarABI,
	files []PackageFile,
) ([]byte, string, error) {
	exports := make(map[string]string, len(files))
	for _, file := range files {
		relative, ok := strings.CutPrefix(
			file.outputPath,
			targetoutput.RuntimePackageRootPath+"/",
		)
		if !ok ||
			path.Clean(relative) != relative ||
			strings.HasPrefix(relative, "../") ||
			path.Ext(relative) != ".ts" {
			return nil, "", &AssemblyError{
				Reason: fmt.Sprintf(
					"runtime package file %q is outside the package root",
					file.outputPath,
				),
			}
		}
		base := strings.TrimSuffix(relative, ".ts")
		subpath := "./" + base + ".js"
		if _, duplicate := exports[subpath]; duplicate {
			return nil, "", &AssemblyError{
				Reason: "runtime package contains duplicate module " + subpath,
			}
		}
		exports[subpath] = "./" + base + ".js"
	}
	document := packageDocument{
		Name:    targetoutput.RuntimePackageName,
		Version: targetoutput.RuntimePackageVersion,
		Private: true,
		Type:    "module",
		GoToTS: packageMetadata{
			IntegerRepresentation: scalar.IntegerRepresentation().String(),
			NativeIntegerBits:     uint8(scalar.NativeIntegerWidth()),
		},
		Exports: exports,
	}
	encoded, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, "", &AssemblyError{
			Reason: "encode runtime package manifest: " + err.Error(),
		}
	}
	encoded = append(encoded, '\n')
	sum := sha256.Sum256(encoded)
	return encoded, hex.EncodeToString(sum[:]), nil
}
