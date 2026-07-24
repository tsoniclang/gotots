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

func TestNormalizedAdmissionDoesNotRebuildPublicRecords(t *testing.T) {
	root := repoRoot(t)
	directory := filepath.Join(
		root,
		"internal",
		"language",
		"semantic",
	)
	fset := token.NewFileSet()
	packages, err := parser.ParseDir(
		fset,
		directory,
		func(info fs.FileInfo) bool {
			name := info.Name()
			return !strings.HasSuffix(name, "_test.go")
		},
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range packages {
		for path, file := range pkg.Files {
			base := filepath.Base(path)
			if !normalizedAdmissionFile(base) {
				continue
			}
			for _, violation := range normalizedAdmissionViolations(
				fset,
				file,
				base == "member_census.go",
			) {
				t.Errorf("%s: %s", path, violation)
			}
		}
	}
}

func TestNormalizedAdmissionWallRejectsProjectedMutation(t *testing.T) {
	const mutation = `package semantic
func validatePackage(pkg Package) error {
	return pkg.VisitTypes(func(record Type) error {
		_ = record.Spec()
		return nil
	})
}
func census(id Identity) string { return id.String() }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(
		fset,
		"member_census.go",
		mutation,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	violations := normalizedAdmissionViolations(fset, file, true)
	if len(violations) != 4 {
		t.Fatalf(
			"normalized-admission wall found %d mutation sites, want 4: %v",
			len(violations),
			violations,
		)
	}
}

func normalizedAdmissionFile(name string) bool {
	return strings.HasPrefix(name, "package_semantic_") ||
		name == "package_reachability.go" ||
		name == "member_census.go"
}

func normalizedAdmissionViolations(
	fset *token.FileSet,
	file *ast.File,
	rejectString bool,
) []string {
	bannedFunctions := map[string]bool{
		"validatePackage":                   true,
		"validateTypeRecords":               true,
		"validateTypeClosure":               true,
		"validatePackageDeclarationTargets": true,
		"memberTargetEntries":               true,
	}
	bannedSelectors := map[string]bool{
		"VisitDefinitions":   true,
		"VisitResolutions":   true,
		"VisitDeclarations":  true,
		"VisitBindings":      true,
		"VisitTypes":         true,
		"VisitTypeWitnesses": true,
		"VisitOperations":    true,
		"VisitUnsupported":   true,
		"Spec":               true,
		"Canonical":          true,
	}
	var out []string
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncDecl:
			if bannedFunctions[node.Name.Name] {
				out = append(
					out,
					fset.Position(node.Name.Pos()).String()+
						" restores superseded normalized validation",
				)
			}
		case *ast.SelectorExpr:
			if bannedSelectors[node.Sel.Name] ||
				(rejectString && node.Sel.Name == "String") {
				out = append(
					out,
					fset.Position(node.Sel.Pos()).String()+
						" projects public semantic data during normalized admission",
				)
			}
		}
		return true
	})
	return out
}
