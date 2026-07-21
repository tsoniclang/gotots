package analyze

import (
	"go/ast"
	"sort"

	"github.com/tsoniclang/gotots/internal/language/catalog"
	"github.com/tsoniclang/gotots/internal/source"
)

// Occurrence is one catalog Kind with the number of times it was observed.
type Occurrence struct {
	Kind  catalog.Kind
	Count int
}

// Inventory is the pure, AST-free result of a construct inspection: the file
// inspected and every catalog Kind it contains, sorted by kind name. It carries
// no go/ast type so downstream layers can consume it without importing the
// toolchain AST.
type Inventory struct {
	Path        string
	Occurrences []Occurrence
}

// InspectConstructs parses one Go file and inventories the catalog constructs
// it contains. It fails closed: any node the catalog does not recognize aborts
// the inspection with the classification error rather than being skipped.
func InspectConstructs(path string) (Inventory, error) {
	_, file, err := source.ParseGoFile(path)
	if err != nil {
		return Inventory{}, err
	}
	return inventoryFile(path, file)
}

// inventoryFile walks a parsed file and counts every construct by catalog Kind.
func inventoryFile(path string, file *ast.File) (Inventory, error) {
	counts := map[catalog.Kind]int{}
	var classifyErr error
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil || classifyErr != nil {
			return false
		}
		kind, err := Classify(n)
		if err != nil {
			classifyErr = err
			return false
		}
		counts[kind]++
		return true
	})
	if classifyErr != nil {
		return Inventory{}, classifyErr
	}
	occurrences := make([]Occurrence, 0, len(counts))
	for kind, count := range counts {
		occurrences = append(occurrences, Occurrence{Kind: kind, Count: count})
	}
	sort.Slice(occurrences, func(i, j int) bool {
		return occurrences[i].Kind.Name() < occurrences[j].Kind.Name()
	})
	return Inventory{Path: path, Occurrences: occurrences}, nil
}
