package load

import (
	"context"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/tsoniclang/gotots/internal/toolchain"
)

func TestExactPackagesDriverPreservesOverlayUniverseAndMetadata(t *testing.T) {
	project := t.TempDir()
	for name, content := range map[string]string{
		"go.mod": "module example.com/overlay\n\ngo 1.26.4\n",
		"main.go": `package overlay

const Selected = "disk"
`,
		"main_test.go": `package overlay

func TestSelected() { _ = Selected }
`,
		"dep/dep.go": `package dep

const Value = "overlay"
`,
	} {
		path := filepath.Join(project, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	selectedGo, err := toolchain.ResolveGo(
		"",
		exactPackageToolCache(t),
	)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := NewBuildProfileForToolchain(
		selectedGo.Version(),
		selectedGo.DefaultGOOS(),
		selectedGo.DefaultGOARCH(),
		false,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	overlayPath := filepath.Join(project, "main.go")
	loaded, err := GoPackages(selectedGo, profile, PackageRequest{
		Context:   context.Background(),
		Directory: project,
		FileSet:   token.NewFileSet(),
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedTypesSizes |
			packages.NeedModule |
			packages.NeedForTest,
		Overlay: map[string][]byte{
			overlayPath: []byte(`package overlay

import "example.com/overlay/dep"

const Selected = dep.Value
`),
		},
		Tests: true,
	}, ".")
	if err != nil {
		t.Fatal(err)
	}

	var dependencySeen bool
	var overlaySeen bool
	var testVariantSeen bool
	var moduleMetadataSeen bool
	packages.Visit(loaded, nil, func(current *packages.Package) {
		if current.PkgPath == "example.com/overlay/dep" {
			dependencySeen = true
			moduleMetadataSeen = current.Module != nil &&
				current.Module.Path == "example.com/overlay" &&
				current.Dir == filepath.Join(project, "dep")
		}
		if current.PkgPath == "example.com/overlay" && current.ForTest == "" {
			for _, file := range current.Syntax {
				ast.Inspect(file, func(node ast.Node) bool {
					selector, ok := node.(*ast.SelectorExpr)
					if ok && selector.Sel.Name == "Value" {
						overlaySeen = true
					}
					return true
				})
			}
		}
		if current.ForTest != "" {
			testVariantSeen = true
		}
	})
	if !dependencySeen || !overlaySeen || !testVariantSeen || !moduleMetadataSeen {
		t.Fatalf(
			"exact package evidence: dependency=%t overlay=%t testVariant=%t metadata=%t",
			dependencySeen,
			overlaySeen,
			testVariantSeen,
			moduleMetadataSeen,
		)
	}
}

func TestExactPackagesDriverDoesNotCompileBodylessSourceRoot(t *testing.T) {
	project := t.TempDir()
	for name, content := range map[string]string{
		"go.mod":    "module example.com/external\n\ngo 1.26.4\n",
		"source.go": "package external\nfunc Read([]byte) (int, error)\n",
	} {
		if err := os.WriteFile(filepath.Join(project, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	selectedGo, err := toolchain.ResolveGo("", exactPackageToolCache(t))
	if err != nil {
		t.Fatal(err)
	}
	profile, err := NewBuildProfileForToolchain(
		selectedGo.Version(),
		selectedGo.DefaultGOOS(),
		selectedGo.DefaultGOARCH(),
		false,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := GoPackages(
		selectedGo,
		profile,
		PackageRequest{
			Context: context.Background(), Directory: project,
			FileSet: token.NewFileSet(),
			Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
				packages.NeedImports | packages.NeedDeps | packages.NeedTypes |
				packages.NeedSyntax | packages.NeedTypesInfo,
		},
		".",
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].Types == nil || len(loaded[0].Syntax) != 1 {
		t.Fatalf("bodyless package evidence = %#v", loaded)
	}
}

func exactPackageToolCache(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(root, ".temp", "cache", "toolchain-tests")
}
