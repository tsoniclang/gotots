package emit

import (
	"go/types"
	"sort"
	"testing"

	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func packageExportBindings(statements []tsgo.Statement) []string {
	var names []string
	for _, statement := range statements {
		declaration, ok := statement.(tsgo.ExportDeclaration)
		if !ok {
			continue
		}
		exports, ok := declaration.ExportClause().(tsgo.NamedExports)
		if !ok {
			continue
		}
		for _, specifier := range exports.Elements() {
			name, ok := specifier.Name().(tsgo.Identifier)
			if ok {
				names = append(names, name.Text())
			}
		}
	}
	sort.Strings(names)
	return names
}

func packageInitializerForVariable(
	t *testing.T,
	sourcePackage *load.Package,
	variable *types.Var,
) *types.Initializer {
	t.Helper()
	var selected *types.Initializer
	for _, initializer := range sourcePackage.TypesInfo().InitOrder {
		for _, target := range initializer.Lhs {
			if target != variable {
				continue
			}
			if selected != nil {
				t.Fatalf(
					"variable %s belongs to multiple package initializers",
					variable.Name(),
				)
			}
			selected = initializer
		}
	}
	if selected == nil {
		t.Fatalf(
			"variable %s has no package initializer",
			variable.Name(),
		)
	}
	return selected
}
