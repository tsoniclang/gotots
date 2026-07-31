package verify

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/tsoniclang/gotots"

func TestRepositoryArchitectureWalls(t *testing.T) {
	root := repositoryRoot(t)
	agents, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	claude, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(agents) != string(claude) {
		t.Fatal("AGENTS.md and CLAUDE.md differ")
	}
	if err := verifyStandaloneOwnership(root); err != nil {
		t.Fatal(err)
	}

	productionFiles := make(map[string]int)
	testFiles := make(map[string]int)
	err = filepath.Walk(
		filepath.Join(root, "internal"),
		func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() || filepath.Ext(path) != ".go" {
				return nil
			}
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			relative = filepath.ToSlash(relative)
			if strings.HasSuffix(relative, "_generated.go") {
				return nil
			}
			directory := filepath.ToSlash(filepath.Dir(relative))
			if strings.HasSuffix(relative, "_test.go") {
				testFiles[directory]++
			} else {
				productionFiles[directory]++
			}
			lines, err := physicalLines(path)
			if err != nil {
				return err
			}
			if lines >= 600 {
				t.Errorf("%s has %d physical lines, want fewer than 600", relative, lines)
			}
			if strings.HasSuffix(relative, "_test.go") {
				return nil
			}
			if err := verifyProductionFile(relative, path); err != nil {
				t.Error(err)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, sourceSet := range []struct {
		kind   string
		counts map[string]int
	}{
		{kind: "production", counts: productionFiles},
		{kind: "test", counts: testFiles},
	} {
		for directory, count := range sourceSet.counts {
			if err := directoryFileBudgetError(
				directory,
				sourceSet.kind,
				count,
			); err != nil {
				t.Error(err)
			}
		}
	}
}

func verifyProductionFile(relative string, sourcePath string) error {
	for _, element := range strings.Split(filepath.ToSlash(filepath.Dir(relative)), "/") {
		switch element {
		case "ir", "plan", "lower", "catalog", "inventory", "legacy", "compat",
			"fallback", "util", "utils", "helper", "helpers", "misc", "common",
			"shared":
			return &wallError{source: relative, reason: "forbidden package directory " + element}
		}
	}
	sourceLayer := layerFor(relative)
	if sourceLayer == 0 {
		return &wallError{source: relative, reason: "production package has no layer owner"}
	}
	file, err := parser.ParseFile(token.NewFileSet(), sourcePath, nil, 0)
	if err != nil {
		return &wallError{source: relative, reason: err.Error()}
	}
	importAliases := make(map[string]string)
	for _, sourceImport := range file.Imports {
		importPath, err := strconv.Unquote(sourceImport.Path.Value)
		if err != nil {
			return &wallError{source: relative, reason: "invalid import literal"}
		}
		if importPath == "reflect" {
			return &wallError{source: relative, reason: "production reflection is forbidden"}
		}
		alias := path.Base(importPath)
		if sourceImport.Name != nil {
			alias = sourceImport.Name.Name
		}
		importAliases[importPath] = alias
		if importPath == "go/ast" || importPath == "go/types" {
			if !strings.HasPrefix(relative, "internal/load/") &&
				!strings.HasPrefix(relative, "internal/contracts/environment/") &&
				!strings.HasPrefix(relative, "internal/contracts/gostdlib/certify/") &&
				!strings.HasPrefix(relative, "internal/emit/") {
				return &wallError{
					source: relative,
					reason: "Go semantic graph import is outside load/emit",
				}
			}
		}
		internalPrefix := modulePath + "/internal/"
		if strings.HasPrefix(
			relative,
			"internal/emit/statement/assignment/",
		) && strings.HasPrefix(importPath, internalPrefix) {
			switch importPath {
			case modulePath + "/internal/emit/api",
				modulePath + "/internal/emit/resulttuple",
				modulePath + "/internal/emit/type/basic",
				modulePath + "/internal/target/tsgo":
			default:
				return &wallError{
					source: relative,
					reason: "assignment owner imports a value-family route",
				}
			}
		}
		if !strings.HasPrefix(importPath, internalPrefix) {
			continue
		}
		importRelative := strings.TrimPrefix(importPath, modulePath+"/") + "/owner.go"
		targetLayer := layerFor(importRelative)
		if targetLayer == 0 {
			return &wallError{
				source: relative,
				reason: "internal import has no layer owner: " + importPath,
			}
		}
		if targetLayer > sourceLayer {
			return &wallError{
				source: relative,
				reason: "reverse layer import: " + importPath,
			}
		}
	}
	if strings.HasPrefix(relative, "internal/emit/") {
		if err := verifyEmissionSource(relative, file, importAliases); err != nil {
			return err
		}
	}
	return nil
}

func verifyEmissionSource(
	relative string,
	file *ast.File,
	importAliases map[string]string,
) error {
	for _, forbiddenImport := range []string{
		"go/format",
		"go/parser",
		"go/printer",
		"html/template",
		"text/template",
	} {
		if importAliases[forbiddenImport] != "" {
			return &wallError{
				source: relative,
				reason: "emission may construct only typed TS-Go AST",
			}
		}
	}
	astAlias := importAliases["go/ast"]
	interfaceValueAlias := importAliases[modulePath+"/internal/emit/value/interfacevalue"]
	var violation string
	ast.Inspect(file, func(node ast.Node) bool {
		if violation != "" {
			return false
		}
		switch node := node.(type) {
		case *ast.CallExpr:
			selector, ok := node.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if selector.Sel.Name == "ThrowStatement" &&
				relative != "internal/emit/runtime/panic/owner.go" {
				violation = "target throw is owned only by the panic runtime"
				return false
			}
			if selector.Sel.Name == "NonNullExpression" {
				violation = "unchecked target non-null assertion is forbidden"
				return false
			}
			qualifier, qualifierOK := selector.X.(*ast.Ident)
			if qualifierOK &&
				qualifier.Name == interfaceValueAlias &&
				(selector.Sel.Name == "AdaptExpected" ||
					(selector.Sel.Name == "Assign" &&
						relative != "internal/emit/value/representation/owner.go")) {
				violation = "interface transfer bypasses the value-transfer owner"
				return false
			}
			for _, forbidden := range forbiddenInterfaceBoundarySelectors(relative) {
				if selector.Sel.Name == forbidden {
					violation = "interface boundary bypasses its method-token owner with " +
						forbidden
					return false
				}
			}
			if qualifierOK && qualifier.Name == astAlias &&
				(selector.Sel.Name == "Walk" || selector.Sel.Name == "Inspect") {
				violation = "production emission uses generic AST recursion"
			}
		case *ast.BasicLit:
			if node.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(node.Value)
			if err != nil {
				violation = "invalid production string literal"
				return false
			}
			for _, forbidden := range []string{
				".apply(",
				".bind(",
				".call(",
				"as any",
				"as unknown",
				"import(",
				"module.exports",
				"require(",
				"/// <reference",
			} {
				if strings.Contains(value, forbidden) {
					violation = "production emission contains forbidden target fragment " + forbidden
					return false
				}
			}
		}
		return true
	})
	if violation != "" {
		return &wallError{source: relative, reason: violation}
	}
	return nil
}

func forbiddenInterfaceBoundarySelectors(relative string) []string {
	switch relative {
	case "internal/emit/declaration/interfaceadapter/handler.go":
		return []string{
			"ValueContract",
			"SourceValueContract",
			"GeneratedValueCall",
			"SelectedMethodCall",
		}
	case "internal/emit/declaration/interfacetype/handler.go":
		return []string{"ValueContract", "SourceValueContract"}
	case "internal/emit/expression/call/method.go":
		return []string{
			"ValueContract",
			"ValueCall",
			"DetachedValueCall",
			"InterfaceMethodToken",
		}
	case "internal/emit/expression/call/deferred_method.go":
		return []string{"ValueContract", "ValueCall", "InterfaceMethodToken"}
	case "internal/emit/expression/methodvalue/handler.go",
		"internal/emit/expression/methodexpression/handler.go":
		return []string{"InterfaceMethodToken"}
	case "internal/emit/generic/capability/constraint_method.go":
		return []string{"ValueContract", "ValueCall", "InterfaceMethodToken"}
	default:
		return nil
	}
}

func TestArchitectureWallMutationControls(t *testing.T) {
	for name, fixture := range map[string]struct {
		relative string
		source   string
	}{
		"reverse layer": {
			relative: "internal/output/leak.go",
			source: `package output
import _ "github.com/tsoniclang/gotots/internal/emit"
`,
		},
		"reflection": {
			relative: "internal/load/leak.go",
			source: `package load
import _ "reflect"
`,
		},
		"semantic graph outside owner": {
			relative: "internal/output/leak.go",
			source: `package output
import _ "go/ast"
`,
		},
		"generic production walk": {
			relative: "internal/emit/expression/leak.go",
			source: `package expression
import "go/ast"
func leak(node ast.Node) { ast.Inspect(node, func(ast.Node) bool { return true }) }
`,
		},
		"raw dynamic target fragment": {
			relative: "internal/emit/expression/leak.go",
			source: `package expression
const leak = "value.call(receiver)"
`,
		},
		"runtime throw outside panic owner": {
			relative: "internal/emit/runtime/array/leak.go",
			source: `package array
func leak(factory Factory, value Expression) { factory.ThrowStatement(value) }
`,
		},
		"target non-null assertion": {
			relative: "internal/emit/runtime/slice/leak.go",
			source: `package slice
func leak(factory Factory, value Expression) { factory.NonNullExpression(value, 0) }
`,
		},
		"family-specific assignment route": {
			relative: "internal/emit/statement/assignment/leak.go",
			source: `package assignment
import _ "github.com/tsoniclang/gotots/internal/emit/store/map"
`,
		},
		"interface call through value ABI": {
			relative: "internal/emit/expression/call/method.go",
			source: `package call
func leak(owner Owner, signature Signature) { owner.ValueContract(signature) }
`,
		},
		"open interface call through runtime token": {
			relative: "internal/emit/expression/call/method.go",
			source: `package call
func leak(names Names, method Method) { names.InterfaceMethodToken(method) }
`,
		},
		"interface adapter raw selected call": {
			relative: "internal/emit/declaration/interfaceadapter/handler.go",
			source: `package interfaceadapter
func leak(owner Owner, method Method) { owner.SelectedMethodCall(method) }
`,
		},
		"interface transfer outside owner": {
			relative: "internal/emit/resulttuple/leak.go",
			source: `package resulttuple
import interfacevalue "github.com/tsoniclang/gotots/internal/emit/value/interfacevalue"
func leak(context Context, source Source, value Value) {
	interfacevalue.Assign(context, source, source.Type(), source.Type(), value)
}
`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			sourcePath := filepath.Join(t.TempDir(), "leak.go")
			if err := os.WriteFile(sourcePath, []byte(fixture.source), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := verifyProductionFile(fixture.relative, sourcePath); err == nil {
				t.Fatal("architecture mutation passed its owning wall")
			}
		})
	}
	if layerFor("internal/rogue/rogue.go") != 0 {
		t.Fatal("unregistered production package received an implicit layer")
	}
	unrelatedProduct := "tso" + "nic"
	for _, mutation := range []string{
		"import type { int64 } from \"@" + unrelatedProduct + "/core\";",
		"go run github.com/tsoniclang/" + unrelatedProduct,
		"target language: " + "c" + "sharp",
	} {
		if err := verifyStandaloneText("mutation.txt", []byte(mutation)); err == nil {
			t.Fatalf("standalone ownership mutation passed: %q", mutation)
		}
	}
	externalRoot := t.TempDir()
	for _, directory := range []string{"node_modules", "dist"} {
		path := filepath.Join(externalRoot, directory, "dependency.ts")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(
			path,
			[]byte("type Origin = \""+unrelatedProduct+"\";"),
			0o644,
		); err != nil {
			t.Fatal(err)
		}
	}
	if err := verifyStandaloneOwnership(externalRoot); err != nil {
		t.Fatalf("external/generated tree entered ownership wall: %v", err)
	}
	ownedPath := filepath.Join(externalRoot, "src", "owned.ts")
	if err := os.MkdirAll(filepath.Dir(ownedPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		ownedPath,
		[]byte("type Origin = \""+unrelatedProduct+"\";"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := verifyStandaloneOwnership(externalRoot); err == nil {
		t.Fatal("repository-owned source escaped standalone ownership wall")
	}
}

func TestBuiltinIdentityResolutionHasOneOwner(t *testing.T) {
	root := repositoryRoot(t)
	owner := "internal/emit/expression/builtin/handler.go"
	resolvers := 0
	err := filepath.Walk(
		filepath.Join(root, "internal", "emit"),
		func(sourcePath string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() ||
				filepath.Ext(sourcePath) != ".go" ||
				strings.HasSuffix(sourcePath, "_test.go") {
				return nil
			}
			relative, err := filepath.Rel(root, sourcePath)
			if err != nil {
				return err
			}
			relative = filepath.ToSlash(relative)
			source, err := os.ReadFile(sourcePath)
			if err != nil {
				return err
			}
			resolverCount := strings.Count(
				string(source),
				".(*types.Builtin)",
			)
			if resolverCount != 0 && relative != owner {
				t.Errorf(
					"%s resolves builtin identity outside %s",
					relative,
					owner,
				)
			}
			resolvers += resolverCount
			if relative != owner &&
				strings.Contains(string(source), "TypesInfo().Uses") &&
				strings.HasPrefix(
					relative,
					"internal/emit/expression/builtin/",
				) {
				t.Errorf(
					"%s rediscovers builtin identity inside a family owner",
					relative,
				)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if resolvers != 1 {
		t.Fatalf("builtin identity resolvers = %d, want one", resolvers)
	}
}

func verifyStandaloneOwnership(root string) error {
	return filepath.Walk(
		root,
		func(sourcePath string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() {
				switch info.Name() {
				case ".analysis", ".claude", ".git", ".temp", ".tests",
					"dist", "node_modules":
					return filepath.SkipDir
				default:
					return nil
				}
			}
			switch filepath.Ext(sourcePath) {
			case ".go", ".json", ".md", ".mod", ".sh", ".sum", ".ts", ".yaml", ".yml":
			default:
				return nil
			}
			relative, err := filepath.Rel(root, sourcePath)
			if err != nil {
				return err
			}
			source, err := os.ReadFile(sourcePath)
			if err != nil {
				return err
			}
			return verifyStandaloneText(filepath.ToSlash(relative), source)
		},
	)
}

func verifyStandaloneText(relative string, source []byte) error {
	unrelatedProduct := "tso" + "nic"
	forbidden := regexp.MustCompile(
		`(?i)(\b` + regexp.QuoteMeta(unrelatedProduct) +
			`\b|@` + regexp.QuoteMeta(unrelatedProduct) +
			`/|\b` + "c" + `sharp\b|` + "c" + `#)`,
	)
	if forbidden.MatchString(relative) || forbidden.Match(source) {
		return &wallError{
			source: relative,
			reason: "standalone GoToTS source references an unrelated product or target",
		}
	}
	return nil
}

func layerFor(relative string) int {
	relative = filepath.ToSlash(relative)
	switch {
	case strings.HasPrefix(relative, "internal/load/"):
		return 10
	case strings.HasPrefix(relative, "internal/contracts/"):
		return 10
	case strings.HasPrefix(relative, "internal/target/tsgo/"):
		return 10
	case strings.HasPrefix(relative, "internal/output/"):
		return 20
	case strings.HasPrefix(relative, "internal/emit/api/"):
		return 20
	case filepath.Dir(relative) == "internal/emit":
		return 40
	case strings.HasPrefix(relative, "internal/emit/"):
		return 30
	case strings.HasPrefix(relative, "internal/verify/"):
		return 50
	default:
		return 0
	}
}

type wallError struct {
	source string
	reason string
}

func (e *wallError) Error() string {
	return e.source + ": " + e.reason
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}
