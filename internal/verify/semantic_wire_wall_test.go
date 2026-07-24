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

func TestSemanticPackageHasNoResidentWireRecordTree(t *testing.T) {
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
			return !strings.HasSuffix(info.Name(), "_test.go")
		},
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range packages {
		for path, file := range pkg.Files {
			for _, violation := range wireRecordTreeFields(
				fset,
				file,
			) {
				t.Errorf(
					"%s: package-wide wire record tree %s",
					path,
					violation,
				)
			}
			for _, violation := range wireExpandedIdentityFields(
				fset,
				file,
			) {
				t.Errorf(
					"%s: expanded wire identity state %s",
					path,
					violation,
				)
			}
		}
	}
	const mutation = `package semantic
type retainedWireTree struct {
	Definitions []wireDefinitionRecord
}`
	file, err := parser.ParseFile(
		fset,
		"wire_tree_mutation.go",
		mutation,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if violations := wireRecordTreeFields(
		fset,
		file,
	); len(violations) != 1 {
		t.Fatalf(
			"wire-tree gate found %d mutation fields, want 1",
			len(violations),
		)
	}
	const identityMutation = `package semantic
import "github.com/tsoniclang/gotots/internal/identity"
type wireIdentityDecoder struct {
	Definitions []identity.DefinitionID
}`
	file, err = parser.ParseFile(
		fset,
		"wire_identity_mutation.go",
		identityMutation,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if violations := wireExpandedIdentityFields(
		fset,
		file,
	); len(violations) != 1 {
		t.Fatalf(
			"wire-identity gate found %d mutation fields, want 1",
			len(violations),
		)
	}
}

func wireRecordTreeFields(
	fset *token.FileSet,
	file *ast.File,
) []string {
	var out []string
	ast.Inspect(file, func(node ast.Node) bool {
		field, ok := node.(*ast.Field)
		if !ok {
			return true
		}
		array, ok := field.Type.(*ast.ArrayType)
		if !ok {
			return true
		}
		element, ok := array.Elt.(*ast.Ident)
		if !ok || !strings.HasPrefix(
			element.Name,
			"wire",
		) || !strings.HasSuffix(
			element.Name,
			"Record",
		) {
			return true
		}
		position := fset.Position(field.Pos())
		out = append(
			out,
			position.String()+" "+element.Name,
		)
		return true
	})
	return out
}

func wireExpandedIdentityFields(
	fset *token.FileSet,
	file *ast.File,
) []string {
	var out []string
	for _, declaration := range file.Decls {
		generic, ok := declaration.(*ast.GenDecl)
		if !ok || generic.Tok != token.TYPE {
			continue
		}
		for _, item := range generic.Specs {
			spec, ok := item.(*ast.TypeSpec)
			if !ok || !strings.Contains(
				spec.Name.Name,
				"wireIdentityDecoder",
			) {
				continue
			}
			structure, ok := spec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			for _, field := range structure.Fields.List {
				array, ok := field.Type.(*ast.ArrayType)
				if !ok {
					continue
				}
				selector, ok := array.Elt.(*ast.SelectorExpr)
				if !ok {
					continue
				}
				pkg, ok := selector.X.(*ast.Ident)
				if !ok || pkg.Name != "identity" {
					continue
				}
				out = append(
					out,
					fset.Position(field.Pos()).String()+
						" []identity."+selector.Sel.Name,
				)
			}
		}
	}
	return out
}
