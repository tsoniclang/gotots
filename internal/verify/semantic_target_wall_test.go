package verify

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestSemanticSchemaIsTargetIndependent(t *testing.T) {
	directory := filepath.Join(
		repoRoot(t), "internal", "language", "semantic",
	)
	fileSet := token.NewFileSet()
	packages, err := parser.ParseDir(
		fileSet,
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
			for _, violation := range semanticTargetLeaks(
				fileSet, file,
			) {
				t.Errorf("%s: %s", path, violation)
			}
		}
	}
}

func TestSemanticTargetWallRejectsTargetSpecificFields(t *testing.T) {
	const mutation = `package semantic
import "example.com/target/tsgo"
type OperationSpec struct {
	TypeScriptNode tsgo.Node
	OutputPath string
}`
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(
		fileSet, "semantic_target_mutation.go", mutation, 0,
	)
	if err != nil {
		t.Fatal(err)
	}
	violations := semanticTargetLeaks(fileSet, file)
	if len(violations) != 4 {
		t.Fatalf(
			"semantic target wall found %d mutation sites, want 4: %v",
			len(violations),
			violations,
		)
	}
}

func semanticTargetLeaks(
	fileSet *token.FileSet,
	file *ast.File,
) []string {
	var violations []string
	for _, imported := range file.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err != nil {
			violations = append(
				violations,
				fileSet.Position(imported.Pos()).String()+
					" has an invalid import path",
			)
			continue
		}
		if targetImportPath(path) {
			violations = append(
				violations,
				fileSet.Position(imported.Pos()).String()+
					" imports target package "+path,
			)
		}
	}
	for _, declaration := range file.Decls {
		generic, ok := declaration.(*ast.GenDecl)
		if !ok || generic.Tok != token.TYPE {
			continue
		}
		for _, item := range generic.Specs {
			spec, ok := item.(*ast.TypeSpec)
			if !ok {
				continue
			}
			ast.Inspect(spec.Type, func(node ast.Node) bool {
				identifier, ok := node.(*ast.Ident)
				if !ok || !targetSchemaName(identifier.Name) {
					return true
				}
				violations = append(
					violations,
					fileSet.Position(identifier.Pos()).String()+
						" introduces target-specific schema name "+
						identifier.Name,
				)
				return true
			})
		}
	}
	return violations
}

func targetImportPath(path string) bool {
	for _, segment := range strings.Split(path, "/") {
		switch strings.ToLower(segment) {
		case "emit", "emitter", "lower", "lowering", "target",
			"typescript", "tsgo":
			return true
		}
	}
	return false
}

func targetSchemaName(name string) bool {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "typescript") ||
		strings.Contains(lower, "tsgo") {
		return true
	}
	switch name {
	case "EmitContext", "Emitter", "ImplementationOwner", "OutputPath",
		"RepresentationPlan", "RuntimeDictionary", "RuntimeHelper",
		"TargetAST", "TargetDeclaration", "TargetExpression",
		"TargetNode", "TargetStatement":
		return true
	default:
		return false
	}
}
