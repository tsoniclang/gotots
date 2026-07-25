package source

import (
	"fmt"
	"go/token"
	"sort"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/identity"
)

type PackageInitializationKind uint8

const (
	PackageInitializationInvalid PackageInitializationKind = 0
	PackageInitializationNone    PackageInitializationKind = 1
	PackageInitializationGo      PackageInitializationKind = 2
)

func (kind PackageInitializationKind) Valid() bool {
	return kind == PackageInitializationNone ||
		kind == PackageInitializationGo
}

func (kind PackageInitializationKind) String() string {
	switch kind {
	case PackageInitializationNone:
		return "none"
	case PackageInitializationGo:
		return "go"
	default:
		return fmt.Sprintf(
			"source.PackageInitializationKind(%d)", uint8(kind),
		)
	}
}

type PackageInitialization struct {
	kind    PackageInitializationKind
	ordinal int
}

func noPackageInitialization() PackageInitialization {
	return PackageInitialization{kind: PackageInitializationNone}
}

func goPackageInitialization(
	ordinal int,
) (PackageInitialization, error) {
	if ordinal < 0 {
		return PackageInitialization{}, &LoadError{
			Reason: "Go package initialization requires a nonnegative ordinal",
		}
	}
	return PackageInitialization{
		kind: PackageInitializationGo, ordinal: ordinal,
	}, nil
}

func (initialization PackageInitialization) Valid() bool {
	switch initialization.kind {
	case PackageInitializationNone:
		return initialization.ordinal == 0
	case PackageInitializationGo:
		return initialization.ordinal >= 0
	default:
		return false
	}
}

func (initialization PackageInitialization) Kind() PackageInitializationKind {
	return initialization.kind
}

func (initialization PackageInitialization) GoOrdinal() (int, bool) {
	return initialization.ordinal,
		initialization.kind == PackageInitializationGo
}

type PackageImport struct {
	importer identity.PackageID
	imported identity.PackageID
}

func NewPackageImport(
	importer identity.PackageID,
	imported identity.PackageID,
) (PackageImport, error) {
	if importer.IsZero() || imported.IsZero() || importer == imported {
		return PackageImport{}, &LoadError{
			Reason: "package import requires distinct package identities",
		}
	}
	return PackageImport{importer: importer, imported: imported}, nil
}

func (edge PackageImport) Importer() identity.PackageID {
	return edge.importer
}

func (edge PackageImport) Imported() identity.PackageID {
	return edge.imported
}

func (edge PackageImport) IsZero() bool {
	return edge.importer.IsZero() && edge.imported.IsZero()
}

func (edge PackageImport) Compare(other PackageImport) int {
	if order := edge.importer.Compare(other.importer); order != 0 {
		return order
	}
	return edge.imported.Compare(other.imported)
}

type PackageNode struct {
	pkg            identity.PackageID
	name           string
	requestedRoot  bool
	initialization PackageInitialization
}

func (node PackageNode) Package() identity.PackageID {
	return node.pkg
}

func (node PackageNode) DeclaredName() string {
	return node.name
}

func (node PackageNode) RequestedRoot() bool {
	return node.requestedRoot
}

func (node PackageNode) Initialization() PackageInitialization {
	return node.initialization
}

type PackageTopology struct {
	nodes   []PackageNode
	imports []PackageImport
}

func (topology PackageTopology) Nodes() []PackageNode {
	return append([]PackageNode(nil), topology.nodes...)
}

func (topology PackageTopology) Imports() []PackageImport {
	return append([]PackageImport(nil), topology.imports...)
}

func (workspace *Workspace) PackageTopology() PackageTopology {
	if workspace == nil {
		return PackageTopology{}
	}
	topology := PackageTopology{
		nodes: make([]PackageNode, 0, len(workspace.packages)),
	}
	for _, pkg := range workspace.packages {
		topology.nodes = append(topology.nodes, PackageNode{
			pkg: pkg.id, name: pkg.name,
			requestedRoot:  pkg.requestedRoot,
			initialization: pkg.initialization,
		})
		topology.imports = append(topology.imports, pkg.imports...)
	}
	sort.Slice(topology.imports, func(left, right int) bool {
		return topology.imports[left].Compare(
			topology.imports[right],
		) < 0
	})
	return topology
}

func resolvePackageTopology(
	universe *Universe,
	metadata map[string]*packages.Package,
) error {
	byPath := make(map[string]*LoadedPackage, len(universe.packages))
	for _, pkg := range universe.packages {
		if pkg.name == "" || !token.IsIdentifier(pkg.name) ||
			pkg.name == "_" {
			return &LoadError{
				Dir: universe.request.Dir,
				Reason: fmt.Sprintf(
					"package %s has invalid declared name %q",
					pkg.id, pkg.name,
				),
			}
		}
		if prior := byPath[pkg.id.ImportPath()]; prior != nil {
			return &LoadError{
				Dir:    universe.request.Dir,
				Reason: "duplicate import path " + pkg.id.ImportPath(),
			}
		}
		byPath[pkg.id.ImportPath()] = pkg
	}
	for _, pkg := range universe.packages {
		resolved := metadata[pkg.id.ImportPath()]
		if resolved == nil {
			if pkg.disposition != DispositionBuiltinUniverse {
				return &LoadError{
					Dir: universe.request.Dir,
					Reason: "package metadata is absent for " +
						pkg.id.String(),
				}
			}
			continue
		}
		for spelling, importedMetadata := range resolved.Imports {
			if spelling == "C" {
				continue
			}
			if importedMetadata == nil {
				return &LoadError{
					Dir: universe.request.Dir,
					Reason: fmt.Sprintf(
						"package %s has unresolved import %q",
						pkg.id, spelling,
					),
				}
			}
			imported := byPath[importedMetadata.PkgPath]
			if imported == nil {
				return &LoadError{
					Dir: universe.request.Dir,
					Reason: fmt.Sprintf(
						"package %s imports absent package %q",
						pkg.id, importedMetadata.PkgPath,
					),
				}
			}
			edge, err := NewPackageImport(pkg.id, imported.id)
			if err != nil {
				return err
			}
			pkg.imports = append(pkg.imports, edge)
		}
		sort.Slice(pkg.imports, func(left, right int) bool {
			return pkg.imports[left].Compare(pkg.imports[right]) < 0
		})
	}
	return assignPackageInitializationOrdinals(universe.packages)
}

func assignPackageInitializationOrdinals(
	packages []*LoadedPackage,
) error {
	byID := make(map[identity.PackageID]*LoadedPackage, len(packages))
	remaining := map[identity.PackageID]bool{}
	for _, pkg := range packages {
		byID[pkg.id] = pkg
		pkg.initialization = PackageInitialization{}
		switch pkg.disposition {
		case DispositionOrdinarySource:
			remaining[pkg.id] = true
		case DispositionBuiltinUniverse, DispositionUnsafeIntrinsic:
			pkg.initialization = noPackageInitialization()
		default:
			return &LoadError{
				Reason: "package initialization has invalid disposition",
			}
		}
	}
	initialized := map[identity.PackageID]bool{}
	for ordinal := 0; len(remaining) != 0; ordinal++ {
		var candidate *LoadedPackage
		for packageID := range remaining {
			pkg := byID[packageID]
			if !importsInitialized(pkg.imports, byID, initialized) {
				continue
			}
			if candidate == nil ||
				pkg.id.ImportPath() < candidate.id.ImportPath() ||
				(pkg.id.ImportPath() == candidate.id.ImportPath() &&
					pkg.id.Compare(candidate.id) < 0) {
				candidate = pkg
			}
		}
		if candidate == nil {
			return &LoadError{
				Reason: "package import graph contains an initialization cycle",
			}
		}
		initialization, err := goPackageInitialization(ordinal)
		if err != nil {
			return err
		}
		candidate.initialization = initialization
		initialized[candidate.id] = true
		delete(remaining, candidate.id)
	}
	return nil
}

func importsInitialized(
	imports []PackageImport,
	packages map[identity.PackageID]*LoadedPackage,
	initialized map[identity.PackageID]bool,
) bool {
	for _, edge := range imports {
		imported := packages[edge.imported]
		if imported == nil {
			return false
		}
		if imported.disposition == DispositionOrdinarySource &&
			!initialized[imported.id] {
			return false
		}
	}
	return true
}

func validatePackageTopology(packages []*Package) error {
	byID := make(map[identity.PackageID]*Package, len(packages))
	runtimeCount := 0
	ordinals := map[int]identity.PackageID{}
	for _, pkg := range packages {
		if pkg == nil || pkg.id.IsZero() || byID[pkg.id] != nil {
			return &LoadError{Reason: "package topology repeats a package"}
		}
		if pkg.name == "" || !token.IsIdentifier(pkg.name) ||
			pkg.name == "_" {
			return &LoadError{
				Reason: "package topology has an invalid declared name",
			}
		}
		if !pkg.initialization.Valid() {
			return &LoadError{
				Reason: "package topology has invalid initialization evidence",
			}
		}
		byID[pkg.id] = pkg
		ordinal, initializes := pkg.initialization.GoOrdinal()
		if initializes {
			if pkg.disposition != DispositionOrdinarySource {
				return &LoadError{
					Reason: "intrinsic package has Go initialization",
				}
			}
			runtimeCount++
			if prior, duplicate := ordinals[ordinal]; duplicate {
				return &LoadError{Reason: fmt.Sprintf(
					"packages %s and %s share initialization ordinal %d",
					prior, pkg.id, ordinal,
				)}
			}
			ordinals[ordinal] = pkg.id
		} else if pkg.disposition == DispositionOrdinarySource {
			return &LoadError{
				Reason: "ordinary package has no Go initialization",
			}
		}
	}
	for ordinal := 0; ordinal < runtimeCount; ordinal++ {
		if ordinals[ordinal].IsZero() {
			return &LoadError{
				Reason: "package initialization ordinal range has a gap",
			}
		}
	}
	for _, pkg := range packages {
		var previous PackageImport
		for index, edge := range pkg.imports {
			imported := byID[edge.imported]
			if edge.importer != pkg.id || imported == nil ||
				(index > 0 && previous.Compare(edge) >= 0) {
				return &LoadError{
					Reason: "package import topology is invalid",
				}
			}
			pkgOrdinal, pkgInitializes := pkg.initialization.GoOrdinal()
			importedOrdinal, importedInitializes :=
				imported.initialization.GoOrdinal()
			if pkgInitializes && importedInitializes &&
				importedOrdinal >= pkgOrdinal {
				return &LoadError{
					Reason: "package initialization violates an import edge",
				}
			}
			previous = edge
		}
	}
	remaining := map[identity.PackageID]bool{}
	initialized := map[identity.PackageID]bool{}
	for _, pkg := range packages {
		if _, initializes := pkg.initialization.GoOrdinal(); initializes {
			remaining[pkg.id] = true
		}
	}
	for expectedOrdinal := 0; len(remaining) > 0; expectedOrdinal++ {
		var candidate *Package
		for packageID := range remaining {
			pkg := byID[packageID]
			if !finalizedImportsInitialized(
				pkg.imports, byID, initialized,
			) {
				continue
			}
			if candidate == nil ||
				pkg.id.ImportPath() < candidate.id.ImportPath() ||
				(pkg.id.ImportPath() == candidate.id.ImportPath() &&
					pkg.id.Compare(candidate.id) < 0) {
				candidate = pkg
			}
		}
		if candidate == nil {
			return &LoadError{
				Reason: "package topology has an initialization cycle",
			}
		}
		actualOrdinal, _ := candidate.initialization.GoOrdinal()
		if actualOrdinal != expectedOrdinal {
			return &LoadError{
				Reason: "package initialization order is noncanonical",
			}
		}
		initialized[candidate.id] = true
		delete(remaining, candidate.id)
	}
	return nil
}

func finalizedImportsInitialized(
	imports []PackageImport,
	packages map[identity.PackageID]*Package,
	initialized map[identity.PackageID]bool,
) bool {
	for _, edge := range imports {
		imported := packages[edge.imported]
		if imported == nil {
			return false
		}
		if _, initializes := imported.initialization.GoOrdinal(); initializes &&
			!initialized[imported.id] {
			return false
		}
	}
	return true
}
