package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/emit/api"
	targetplacement "github.com/tsoniclang/gotots/internal/emit/placement"
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
	integer     api.IntegerRepresentation
	concurrency api.ConcurrencySemantics
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
	return p.integer
}

func (p Package) Concurrency() api.ConcurrencySemantics {
	return p.concurrency
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
	integer api.IntegerRepresentation,
	concurrency api.ConcurrencySemantics,
	requested map[api.RuntimeSymbol]struct{},
	aliases []api.PrimitiveAlias,
) (Package, error) {
	if !integer.Valid() {
		return Package{}, &AssemblyError{
			Reason: "runtime package integer profile is invalid",
		}
	}
	if !concurrency.Valid() {
		return Package{}, &AssemblyError{
			Reason: "runtime package concurrency profile is invalid",
		}
	}
	aliases = slices.Clone(aliases)
	slices.Sort(aliases)
	for index, alias := range aliases {
		if _, _, err := api.PrimitiveAliasRepresentation(alias, integer); err != nil {
			return Package{}, err
		}
		if index != 0 && alias == aliases[index-1] {
			return Package{}, &AssemblyError{
				Reason: "runtime package contains a duplicate primitive alias",
			}
		}
	}
	closed, err := dependencyClosure(requested)
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
	statements := make([]tsgo.Statement, 0, len(aliases)+1)
	if symbols := byModule[api.RuntimeModuleScalar]; len(symbols) != 0 {
		slices.Sort(symbols)
		definitions, err := Build(
			factory,
			api.RuntimeModuleScalar,
			symbols,
			concurrency,
		)
		if err != nil {
			return Package{}, err
		}
		statements, err = exactDefinitions(
			api.RuntimeModuleScalar,
			symbols,
			definitions,
		)
		if err != nil {
			return Package{}, err
		}
		delete(byModule, api.RuntimeModuleScalar)
	}
	for _, alias := range aliases {
		name, keyword, err := api.PrimitiveAliasRepresentation(alias, integer)
		if err != nil {
			return Package{}, err
		}
		statements = append(statements, factory.TypeAliasDeclaration(
			[]tsgo.ModifierLike{factory.ExportKeyword()},
			factory.Identifier(name),
			nil,
			factory.KeywordTypeNode(keyword),
		))
	}
	if len(statements) != 0 {
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
		if module == api.RuntimeModuleScalar {
			continue
		}
		symbols := byModule[module]
		slices.Sort(symbols)
		definitions, err := Build(factory, module, symbols, concurrency)
		if err != nil {
			return Package{}, err
		}
		statements, err := exactDefinitions(module, symbols, definitions)
		if err != nil {
			return Package{}, err
		}
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
	manifest, fingerprint, err := packageManifest(integer, concurrency, files)
	if err != nil {
		return Package{}, err
	}
	return Package{
		files:       files,
		manifest:    manifest,
		fingerprint: fingerprint,
		integer:     integer,
		concurrency: concurrency,
		valid:       true,
	}, nil
}

func awaitableType(
	factory tsgo.Factory,
) tsgo.Statement {
	parameter := factory.Identifier("T")
	value := factory.TypeReferenceNode(parameter, nil)
	result := factory.UnionTypeNode([]tsgo.TypeNode{
		value,
		factory.TypeReferenceNode(
			api.TargetIntrinsicPromise.TypeName(factory),
			[]tsgo.TypeNode{value},
		),
	})
	return factory.TypeAliasDeclaration(
		[]tsgo.ModifierLike{factory.ExportKeyword()},
		factory.Identifier("Awaitable"),
		[]tsgo.TypeParameterDeclaration{
			factory.TypeParameterDeclaration(nil, parameter, nil, nil, nil),
		},
		result,
	)
}

func dependencyClosure(
	requested map[api.RuntimeSymbol]struct{},
) (map[api.RuntimeSymbol]struct{}, error) {
	result := make(map[api.RuntimeSymbol]struct{}, len(requested))
	state := make(map[api.RuntimeSymbol]uint8, len(requested))
	var visit func(api.RuntimeSymbol) error
	visit = func(symbol api.RuntimeSymbol) error {
		switch state[symbol] {
		case 1:
			return &AssemblyError{
				Symbol: symbol,
				Reason: "runtime dependency graph contains a cycle",
			}
		case 2:
			return nil
		}
		contract, err := api.RuntimeContract(symbol)
		if err != nil {
			return err
		}
		state[symbol] = 1
		dependencies := contract.Dependencies()
		slices.Sort(dependencies)
		for _, dependency := range dependencies {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		state[symbol] = 2
		result[symbol] = struct{}{}
		return nil
	}
	symbols := make([]api.RuntimeSymbol, 0, len(requested))
	for symbol := range requested {
		symbols = append(symbols, symbol)
	}
	slices.Sort(symbols)
	for _, symbol := range symbols {
		if err := visit(symbol); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func exactDefinitions(
	module api.RuntimeModule,
	symbols []api.RuntimeSymbol,
	definitions []Definition,
) ([]tsgo.Statement, error) {
	requested := make(map[api.RuntimeSymbol]struct{}, len(symbols))
	for _, symbol := range symbols {
		requested[symbol] = struct{}{}
	}
	bySymbol := make(map[api.RuntimeSymbol]tsgo.Statement, len(definitions))
	for _, definition := range definitions {
		symbol := definition.Symbol()
		contract, err := api.RuntimeContract(symbol)
		if err != nil {
			return nil, err
		}
		if contract.Module() != module {
			return nil, &AssemblyError{
				Module: module,
				Symbol: symbol,
				Reason: "definition belongs to another runtime module",
			}
		}
		if _, ok := requested[symbol]; !ok {
			return nil, &AssemblyError{
				Module: module,
				Symbol: symbol,
				Reason: "definition was not requested",
			}
		}
		if _, duplicate := bySymbol[symbol]; duplicate {
			return nil, &AssemblyError{
				Module: module,
				Symbol: symbol,
				Reason: "runtime symbol has duplicate definitions",
			}
		}
		bySymbol[symbol] = definition.Statement()
	}
	statements := make([]tsgo.Statement, 0, len(symbols))
	for _, symbol := range symbols {
		statement := bySymbol[symbol]
		if statement == nil {
			return nil, &AssemblyError{
				Module: module,
				Symbol: symbol,
				Reason: "requested runtime symbol has no definition",
			}
		}
		statements = append(statements, statement)
	}
	return statements, nil
}

func moduleImports(
	factory tsgo.Factory,
	outputPath string,
	module api.RuntimeModule,
	symbols []api.RuntimeSymbol,
) ([]tsgo.Statement, error) {
	placement := targetplacement.New()
	for _, symbol := range symbols {
		contract, err := api.RuntimeContract(symbol)
		if err != nil {
			return nil, err
		}
		for _, dependency := range contract.Dependencies() {
			dependencyContract, err := api.RuntimeContract(dependency)
			if err != nil {
				return nil, err
			}
			if dependencyContract.Module() == module {
				continue
			}
			modulePath, err := targetoutput.ModuleSpecifier(
				outputPath,
				dependencyContract.OutputPath(),
			)
			if err != nil {
				return nil, err
			}
			request, err := api.NewRuntimeImportRequest(
				factory,
				api.ImportPhaseValue,
				modulePath,
				dependency,
				dependencyContract.ExportedName(),
			)
			if err != nil {
				return nil, err
			}
			if err := placement.Apply([]api.RootRequest{request}); err != nil {
				return nil, err
			}
		}
	}
	return placement.Statements(factory), nil
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

type packageExport struct {
	Types   string `json:"types"`
	Default string `json:"default"`
}

type packageMetadata struct {
	IntegerRepresentation string `json:"integerRepresentation"`
	ConcurrencySemantics  string `json:"concurrencySemantics"`
}

type packageDocument struct {
	Name    string                   `json:"name"`
	Version string                   `json:"version"`
	Private bool                     `json:"private"`
	Type    string                   `json:"type"`
	GoToTS  packageMetadata          `json:"gotots"`
	Exports map[string]packageExport `json:"exports"`
}

func packageManifest(
	integer api.IntegerRepresentation,
	concurrency api.ConcurrencySemantics,
	files []PackageFile,
) ([]byte, string, error) {
	exports := make(map[string]packageExport, len(files))
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
		exports[subpath] = packageExport{
			Types:   "./" + base + ".d.ts",
			Default: "./" + base + ".js",
		}
	}
	document := packageDocument{
		Name:    targetoutput.RuntimePackageName,
		Version: "0.0.0",
		Private: true,
		Type:    "module",
		GoToTS: packageMetadata{
			IntegerRepresentation: integer.String(),
			ConcurrencySemantics:  concurrency.String(),
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
