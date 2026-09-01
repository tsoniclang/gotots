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
	scalarcontract "github.com/tsoniclang/gotots/internal/emit/runtime/scalar"
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
	var scalarImports []api.RootRequest
	for _, alias := range aliases {
		name, keyword, err := api.PrimitiveAliasRepresentation(alias, scalar)
		if err != nil {
			return Package{}, err
		}
		underlying := tsgo.TypeNode(factory.KeywordTypeNode(keyword))
		shared, selected, err := scalarcontract.SharedDeclaration(alias, scalar)
		if err != nil {
			return Package{}, err
		}
		if selected {
			localName := sharedPrimitiveLocalName(shared.Export())
			underlying = factory.TypeReferenceNode(
				factory.Identifier(localName),
				nil,
			)
			request, requestErr := api.NewImportRequest(
				factory,
				api.ImportPhaseType,
				shared.Module(),
				shared.Export(),
				localName,
			)
			if requestErr != nil {
				return Package{}, requestErr
			}
			scalarImports = append(scalarImports, request)
		}
		statements = append(statements, factory.TypeAliasDeclaration(
			[]tsgo.ModifierLike{factory.ExportKeyword()},
			factory.Identifier(name),
			nil,
			underlying,
		))
	}
	if len(statements) != 0 {
		placement := targetplacement.New()
		if err := placement.Apply(scalarImports); err != nil {
			return Package{}, err
		}
		statements = append(placement.Statements(factory), statements...)
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

func sharedPrimitiveLocalName(exportedName string) string {
	return "Tsonic" + strings.ToUpper(exportedName[:1]) + exportedName[1:]
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
