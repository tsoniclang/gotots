package stagecheck

import (
	"fmt"
	"go/token"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/source"
)

func resolveExpectedPackageTopology(
	expected map[identity.PackageID]*universeExpectation,
) error {
	byPath := make(map[string]identity.PackageID, len(expected))
	remaining := map[identity.PackageID]bool{}
	for packageID, record := range expected {
		if record.name == "" || !token.IsIdentifier(record.name) ||
			record.name == "_" {
			return fmt.Errorf(
				"%s has invalid independently derived package name %q",
				packageID, record.name,
			)
		}
		if prior, duplicate := byPath[packageID.ImportPath()]; duplicate {
			return fmt.Errorf(
				"%s and %s share import path %q",
				prior, packageID, packageID.ImportPath(),
			)
		}
		byPath[packageID.ImportPath()] = packageID
		record.imports = map[identity.PackageID]bool{}
		record.initOrdinal = -1
		if record.initialization == source.PackageInitializationGo {
			remaining[packageID] = true
		}
	}
	for packageID, record := range expected {
		for importPath := range record.importPaths {
			imported, present := byPath[importPath]
			if !present {
				return fmt.Errorf(
					"%s independently imports absent package %q",
					packageID, importPath,
				)
			}
			if imported == packageID || record.imports[imported] {
				return fmt.Errorf(
					"%s has invalid independent import target %s",
					packageID, imported,
				)
			}
			record.imports[imported] = true
		}
	}
	initialized := map[identity.PackageID]bool{}
	for ordinal := 0; len(remaining) > 0; ordinal++ {
		var candidate identity.PackageID
		for packageID := range remaining {
			if !expectedImportsInitialized(
				expected[packageID], expected, initialized,
			) {
				continue
			}
			if candidate.IsZero() ||
				packageID.ImportPath() < candidate.ImportPath() ||
				(packageID.ImportPath() == candidate.ImportPath() &&
					packageID.Compare(candidate) < 0) {
				candidate = packageID
			}
		}
		if candidate.IsZero() {
			return fmt.Errorf(
				"independent package topology contains an initialization cycle",
			)
		}
		expected[candidate].initOrdinal = ordinal
		initialized[candidate] = true
		delete(remaining, candidate)
	}
	return nil
}

func expectedImportsInitialized(
	record *universeExpectation,
	expected map[identity.PackageID]*universeExpectation,
	initialized map[identity.PackageID]bool,
) bool {
	for imported := range record.imports {
		target := expected[imported]
		if target == nil {
			return false
		}
		if target.initialization == source.PackageInitializationGo &&
			!initialized[imported] {
			return false
		}
	}
	return true
}

func packageImportSet(
	pkg *source.Package,
	problems *problemSet,
) map[identity.PackageID]bool {
	out := map[identity.PackageID]bool{}
	for _, edge := range pkg.Imports() {
		if edge.Importer() != pkg.ID() {
			problems.addf(
				"%s carries import edge owned by %s",
				pkg.ID(), edge.Importer(),
			)
		}
		if out[edge.Imported()] {
			problems.addf(
				"%s repeats import target %s",
				pkg.ID(), edge.Imported(),
			)
		}
		out[edge.Imported()] = true
	}
	return out
}

func joinPackageIDSet(
	pkg identity.PackageID,
	class string,
	actual, expected map[identity.PackageID]bool,
	problems *problemSet,
) {
	for member := range actual {
		if !expected[member] {
			problems.addf(
				"%s holds %s %s the toolchain does not name",
				pkg, class, member,
			)
		}
	}
	for member := range expected {
		if !actual[member] {
			problems.addf(
				"%s misses toolchain %s %s", pkg, class, member,
			)
		}
	}
}

func verifyPackageInitialization(
	pkg *source.Package,
	expected *universeExpectation,
	problems *problemSet,
) {
	actual := pkg.Initialization()
	if actual.Kind() != expected.initialization {
		problems.addf(
			"%s initialization %d vs independent %d",
			pkg.ID(), actual.Kind(), expected.initialization,
		)
		return
	}
	ordinal, initializes := actual.GoOrdinal()
	expectedInitializes :=
		expected.initialization == source.PackageInitializationGo
	if initializes != expectedInitializes ||
		(initializes && ordinal != expected.initOrdinal) {
		problems.addf(
			"%s initialization ordinal %d/%t vs independent %d/%t",
			pkg.ID(), ordinal, initializes,
			expected.initOrdinal, expectedInitializes,
		)
	}
}
