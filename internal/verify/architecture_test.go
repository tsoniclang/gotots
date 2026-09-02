package verify

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
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
	if err := verifyCompilerDependencyBoundary(root); err != nil {
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
			"shared", "sourcefact":
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
	if err := verifyNoGoTargetFactSchema(relative, file); err != nil {
		return err
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
				!strings.HasPrefix(relative, "internal/contracts/callableimplementation/") &&
				!strings.HasPrefix(relative, "internal/contracts/environment/") &&
				!strings.HasPrefix(relative, "internal/contracts/externals/certify/") &&
				!strings.HasPrefix(relative, "internal/contracts/gostdlib/certify/") &&
				!strings.HasPrefix(relative, "internal/contracts/gostdlib/sourcecontract/") &&
				!strings.HasPrefix(relative, "internal/contracts/representation/") &&
				relative != "internal/contracts/sourceimplementation/verify_callable.go" &&
				relative != "internal/command/roots.go" &&
				!strings.HasPrefix(relative, "internal/emit/") {
				return &wallError{
					source: relative,
					reason: "Go semantic graph import is outside an approved owner",
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
	if err := verifyToolSelectionRoute(relative, file, importAliases); err != nil {
		return err
	}
	return nil
}

func verifyNoGoTargetFactSchema(relative string, file *ast.File) error {
	var forbidden string
	ast.Inspect(file, func(node ast.Node) bool {
		if forbidden != "" {
			return false
		}
		identifier, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		if strings.HasPrefix(identifier.Name, "Go") &&
			strings.HasSuffix(identifier.Name, "Fact") {
			forbidden = identifier.Name
			return false
		}
		return true
	})
	if forbidden == "" {
		return nil
	}
	return &wallError{
		source: relative,
		reason: "Go-specific target fact schema is forbidden: " + forbidden,
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
			relative: "internal/contracts/externals/leak.go",
			source: `package externals
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
		"source formatter inside emission": {
			relative: "internal/emit/leak.go",
			source: `package emit
import _ "go/format"
`,
		},
		"raw dynamic target fragment": {
			relative: "internal/emit/expression/leak.go",
			source: `package expression
const leak = "value.call(receiver)"
`,
		},
		"raw generated temporary": {relative: "internal/emit/expression/leak.go", source: "package expression\nconst leak = \"__gotots_field_0\"\n"},
		"Go-specific target fact schema": {
			relative: "internal/emit/api/leak.go",
			source: `package api
type GoCallableFact struct{}
`,
		},
		"runtime throw outside panic owner": {
			relative: "internal/emit/runtime/array/leak.go",
			source: `package array
func leak(factory Factory, value Expression) { factory.ThrowStatement(value) }
`,
		},
		"raw class declaration": {
			relative: "internal/emit/declaration/leak.go",
			source: `package declaration
func leak(factory Factory) { factory.ClassDeclaration(nil, nil, nil, nil, nil) }
`,
		},
		"raw Promise assimilation spelling": {
			relative: "internal/emit/declaration/leak.go",
			source: `package declaration
const leak = "then"
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
	tsonic := "tso" + "nic"
	allowed := "import type { Pointer } from \"@" + tsonic + "/core/lang.js\";"
	if err := verifyCompilerDependencyText("canonical.ts", []byte(allowed)); err != nil {
		t.Fatalf("canonical marker dependency was rejected: %v", err)
	}
	for _, mutation := range []string{
		"go run github.com/tsoniclang/" + tsonic,
		"import { createTsonicPlugin } from \"@" + tsonic + "/target-typescript\";",
		"import { location } from \"@" + tsonic + "/typescript-runtime\";",
		"import type { TargetSourceProgram } from \"@" + tsonic + "/tsts\";",
	} {
		if err := verifyCompilerDependencyText("mutation.ts", []byte(mutation)); err == nil {
			t.Fatalf("compiler dependency mutation passed: %q", mutation)
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
			[]byte("import {} from \"@"+tsonic+"/target-typescript\";"),
			0o644,
		); err != nil {
			t.Fatal(err)
		}
	}
	if err := verifyCompilerDependencyBoundary(externalRoot); err != nil {
		t.Fatalf("external/generated tree entered ownership wall: %v", err)
	}
	ownedPath := filepath.Join(externalRoot, "src", "owned.ts")
	if err := os.MkdirAll(filepath.Dir(ownedPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		ownedPath,
		[]byte("import {} from \"@"+tsonic+"/target-typescript\";"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := verifyCompilerDependencyBoundary(externalRoot); err == nil {
		t.Fatal("repository-owned source escaped compiler dependency wall")
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

func verifyCompilerDependencyBoundary(root string) error {
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
			return verifyCompilerDependencyText(filepath.ToSlash(relative), source)
		},
	)
}

func verifyCompilerDependencyText(relative string, source []byte) error {
	sourceText := string(source)
	for _, forbidden := range []string{
		"github.com/tsoniclang/" + "tso" + "nic",
		"@" + "tso" + "nic/target-",
		"@" + "tso" + "nic/typescript-runtime",
		"@" + "tso" + "nic/tsts",
	} {
		if strings.Contains(sourceText, forbidden) {
			return &wallError{
				source: relative,
				reason: "GoToTS compiler source depends on a downstream checker or target",
			}
		}
	}
	return nil
}

func layerFor(relative string) int {
	relative = filepath.ToSlash(relative)
	switch {
	case strings.HasPrefix(relative, "internal/load/"),
		strings.HasPrefix(relative, "internal/contracts/"),
		strings.HasPrefix(relative, "internal/testfixture/"),
		strings.HasPrefix(relative, "internal/toolchain/"),
		strings.HasPrefix(relative, "internal/target/tsgo/"),
		strings.HasPrefix(relative, "internal/target/tsgoprinter/"):
		return 10
	case strings.HasPrefix(relative, "internal/output/"),
		strings.HasPrefix(relative, "internal/emit/api/"):
		return 20
	case filepath.Dir(relative) == "internal/emit":
		return 40
	case strings.HasPrefix(relative, "internal/emit/"):
		return 30
	case strings.HasPrefix(relative, "internal/config/"):
		return 50
	case strings.HasPrefix(relative, "internal/command/"):
		return 60
	case strings.HasPrefix(relative, "internal/verify/"):
		return 70
	default:
		return 0
	}
}
