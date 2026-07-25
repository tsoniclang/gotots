package verify

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestSemanticProductionHasOnePackageRepresentation(t *testing.T) {
	root := repoRoot(t)
	directories := []string{
		filepath.Join(root, "internal", "language", "frontend"),
		filepath.Join(root, "internal", "language", "semantic"),
	}
	fset := token.NewFileSet()
	for _, directory := range directories {
		packages, err := parser.ParseDir(
			fset,
			directory,
			func(info fs.FileInfo) bool {
				return !strings.HasSuffix(info.Name(), "_test.go")
			},
			0,
		)
		if err != nil {
			t.Fatal(err)
		}
		for _, pkg := range packages {
			for path, file := range pkg.Files {
				for _, violation := range semanticRepresentationViolations(
					fset,
					file,
				) {
					t.Errorf("%s: %s", path, violation)
				}
			}
		}
	}
}

func TestSemanticRepresentationWallRejectsSupersededPaths(t *testing.T) {
	const mutation = `package semantic
func ProjectLocalPackage(pkg Package) Package { return pkg }
func records() []Declaration { return nil }
func rebuild(input PackageInput) {
	_, _ = NewPackage(input)
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(
		fset,
		"semantic_representation_mutation.go",
		mutation,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	violations := semanticRepresentationViolations(fset, file)
	if len(violations) != 3 {
		t.Fatalf(
			"semantic representation wall found %d mutation sites, want 3: %v",
			len(violations),
			violations,
		)
	}
}

func semanticRepresentationViolations(
	fset *token.FileSet,
	file *ast.File,
) []string {
	publicRecords := map[string]bool{
		"DefinitionSemantics":  true,
		"OccurrenceResolution": true,
		"Declaration":          true,
		"Binding":              true,
		"Type":                 true,
		"TypeWitness":          true,
		"Operation":            true,
		"Unsupported":          true,
		"Package":              true,
	}
	var violations []string
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncDecl:
			if node.Name.Name == "ProjectLocalPackage" ||
				node.Name.Name == "projectLocalPackage" {
				violations = append(
					violations,
					fset.Position(node.Name.Pos()).String()+
						" restores post-seal package projection",
				)
			}
			if functionReturnsSemanticRecordSlice(
				node.Type,
				file.Name.Name == "semantic",
				publicRecords,
			) {
				violations = append(
					violations,
					fset.Position(node.Name.Pos()).String()+
						" returns a package-wide semantic record slice",
				)
			}
		case *ast.CallExpr:
			if function, ok := node.Fun.(*ast.Ident); ok &&
				function.Name == "NewPackage" &&
				file.Name.Name == "semantic" {
				violations = append(
					violations,
					fset.Position(function.Pos()).String()+
						" rebuilds a sealed semantic package",
				)
				break
			}
			selector, ok := node.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "NewPackage" {
				break
			}
			pkg, ok := selector.X.(*ast.Ident)
			if ok && pkg.Name == "semantic" {
				violations = append(
					violations,
					fset.Position(selector.Sel.Pos()).String()+
						" rebuilds a sealed semantic package",
				)
			}
		}
		return true
	})
	return violations
}

func functionReturnsSemanticRecordSlice(
	function *ast.FuncType,
	semanticPackage bool,
	publicRecords map[string]bool,
) bool {
	if function == nil || function.Results == nil {
		return false
	}
	for _, result := range function.Results.List {
		array, ok := result.Type.(*ast.ArrayType)
		if !ok || array.Len != nil {
			continue
		}
		if record, ok := array.Elt.(*ast.Ident); ok &&
			semanticPackage &&
			publicRecords[record.Name] {
			return true
		}
		selector, ok := array.Elt.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		pkg, ok := selector.X.(*ast.Ident)
		if ok &&
			pkg.Name == "semantic" &&
			publicRecords[selector.Sel.Name] {
			return true
		}
	}
	return false
}
