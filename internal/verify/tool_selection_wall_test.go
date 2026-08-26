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

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func verifyToolSelectionRoute(
	relative string,
	file *ast.File,
	importAliases map[string]string,
) error {
	runtimeAlias := importAliases["runtime"]
	execAlias := importAliases["os/exec"]
	osAlias := importAliases["os"]
	packagesAlias := importAliases["golang.org/x/tools/go/packages"]
	tsgoAlias := importAliases[modulePath+"/internal/target/tsgo"]
	var violation string
	ast.Inspect(file, func(node ast.Node) bool {
		if violation != "" {
			return false
		}
		if identifier, ok := node.(*ast.Ident); ok && identifier.Name == "GoBinary" {
			violation = "superseded provider-local Go binary selection is forbidden"
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if identifier, ok := call.Fun.(*ast.Ident); ok && identifier.Name == "inspectRootManifest" &&
			relative != "internal/toolchain/root_manifest.go" &&
			relative != "internal/toolchain/root_snapshot.go" {
			violation = "complete Go root inspection bypasses the snapshot owner"
			return false
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		qualifier, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		switch {
		case selector.Sel.Name == "VerifyComplete" &&
			!strings.HasPrefix(relative, "internal/toolchain/") &&
			relative != "internal/command/build.go":
			violation = "complete Go root verification is outside the pre-publication owner"
		case qualifier.Name == runtimeAlias && selector.Sel.Name == "GOROOT":
			violation = "compiler runtime GOROOT bypasses selected Go tool"
		case qualifier.Name == osAlias && selector.Sel.Name == "TempDir":
			violation = "ambient temporary storage bypasses the selected .temp/cache root"
		case qualifier.Name == packagesAlias && selector.Sel.Name == "Load" &&
			relative != "internal/load/exact_packages.go" &&
			relative != "internal/load/packages_driver.go":
			violation = "go/packages loading bypasses the exact selected Go adapter"
		case qualifier.Name == tsgoAlias &&
			(selector.Sel.Name == "StartClient" || selector.Sel.Name == "Compile"):
			violation = "TS-Go consumer bypasses the resolved tool selection"
		}
		if violation != "" {
			return false
		}
		argument := executableArgument(qualifier.Name, execAlias, selector.Sel.Name)
		if argument >= 0 && argument < len(call.Args) {
			literal, ok := call.Args[argument].(*ast.BasicLit)
			if ok && literal.Kind == token.STRING && literal.Value == `"go"` {
				violation = "hard-coded Go executable bypasses the resolved tool selection"
				return false
			}
		}
		return true
	})
	if violation != "" {
		return &wallError{source: relative, reason: violation}
	}
	return nil
}

func TestCompleteRootVerificationHasOneCompilationConsumer(t *testing.T) {
	root := repositoryRoot(t)
	var consumers []string
	err := filepath.Walk(filepath.Join(root, "internal"), func(
		sourcePath string,
		info os.FileInfo,
		walkErr error,
	) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || filepath.Ext(sourcePath) != ".go" ||
			strings.HasSuffix(sourcePath, "_test.go") {
			return nil
		}
		relative, err := filepath.Rel(root, sourcePath)
		if err != nil {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), sourcePath, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if ok && selector.Sel.Name == "VerifyComplete" &&
				!strings.HasPrefix(filepath.ToSlash(relative), "internal/toolchain/") {
				consumers = append(consumers, filepath.ToSlash(relative))
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(consumers) != 1 || consumers[0] != "internal/command/build.go" {
		t.Fatalf("complete Go root verification consumers = %v", consumers)
	}
}

func toolSelectionMutationFile(t *testing.T, source string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "mutation.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func toolSelectionMutationImports(t *testing.T, file *ast.File) map[string]string {
	t.Helper()
	result := make(map[string]string)
	for _, selected := range file.Imports {
		importPath, err := strconv.Unquote(selected.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		name := path.Base(importPath)
		if selected.Name != nil {
			name = selected.Name.Name
		}
		result[importPath] = name
	}
	return result
}

func TestToolSelectionWallMutationControls(t *testing.T) {
	for name, fixture := range map[string]struct {
		relative string
		source   string
	}{
		"runtime root": {
			relative: "internal/load/leak.go",
			source:   `package load; import "runtime"; func leak() { _ = runtime.GOROOT() }`,
		},
		"ambient temporary root": {
			relative: "internal/load/leak.go",
			source:   `package load; import "os"; func leak() { _ = os.TempDir() }`,
		},
		"hardcoded Go": {
			relative: "internal/load/leak.go",
			source:   `package load; import "os/exec"; func leak() { _ = exec.Command("go") }`,
		},
		"package loader bypass": {
			relative: "internal/command/leak.go",
			source:   `package command; import "golang.org/x/tools/go/packages"; func leak() { _, _ = packages.Load(nil) }`,
		},
		"TS-Go resolver bypass": {
			relative: "internal/command/leak.go",
			source: `package command
import "github.com/tsoniclang/gotots/internal/target/tsgo"
func leak() { _, _ = tsgo.StartClient(".", ".") }`,
		},
		"complete verification bypass": {
			relative: "internal/load/leak.go",
			source:   `package load; func leak(selected Tool) { _ = selected.VerifyComplete() }`,
		},
		"complete inspection bypass": {
			relative: "internal/toolchain/go.go",
			source:   `package toolchain; func leak() { _, _ = inspectRootManifest(".", ".", nil) }`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			file := toolSelectionMutationFile(t, fixture.source)
			if err := verifyToolSelectionRoute(
				fixture.relative,
				file,
				toolSelectionMutationImports(t, file),
			); err == nil {
				t.Fatal("tool-selection mutation passed its owning wall")
			}
		})
	}
}

func executableArgument(qualifier string, execAlias string, operation string) int {
	if qualifier != execAlias {
		return -1
	}
	switch operation {
	case "Command", "LookPath":
		return 0
	case "CommandContext":
		return 1
	default:
		return -1
	}
}
