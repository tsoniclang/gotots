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

func TestSemanticArtifactHasOneNormalizedBinaryDetailPath(t *testing.T) {
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
			for _, violation := range forbiddenSemanticArtifactImports(
				path,
				file,
			) {
				t.Errorf("%s: %s", path, violation)
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
	const codecMutation = `package semantic
import "encoding/json"
func decodeBinaryDetail(value []byte) error {
	return json.Unmarshal(value, &struct{}{})
}`
	file, err = parser.ParseFile(
		fset,
		"artifact_binary_mutation.go",
		codecMutation,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	if violations := forbiddenSemanticArtifactImports(
		"artifact_binary_mutation.go",
		file,
	); len(violations) != 1 {
		t.Fatalf(
			"binary-codec gate found %d mutation imports, want 1",
			len(violations),
		)
	}
}

func forbiddenSemanticArtifactImports(
	path string,
	file *ast.File,
) []string {
	if !strings.HasPrefix(
		filepath.Base(path),
		"artifact_",
	) {
		return nil
	}
	manifestJSONOwners := map[string]bool{
		"artifact_parse.go":  true,
		"artifact_store.go":  true,
		"artifact_writer.go": true,
	}
	var out []string
	for _, imported := range file.Imports {
		name := strings.Trim(imported.Path.Value, `"`)
		switch {
		case name == "encoding/json" &&
			!manifestJSONOwners[filepath.Base(path)]:
			out = append(
				out,
				"semantic detail imports encoding/json outside manifest owner",
			)
		case name == "encoding/gob" ||
			name == "reflect" ||
			name == "unsafe":
			out = append(
				out,
				"semantic artifact imports forbidden codec dependency "+
					name,
			)
		}
	}
	return out
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
