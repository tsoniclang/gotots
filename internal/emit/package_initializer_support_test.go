package emit

import (
	"context"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

func TestPackageInitializerCarriesSourceEvidenceIntoLocalTypeFacts(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(directory, "go.mod"),
		[]byte("module example.com/initializerfacts\n\ngo 1.26.4\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "source.go"),
		[]byte(`package initializerfacts

	var Value = func() int32 {
		type local struct {
			Item int32
		}
		return local{Item: 1}.Item
	}()
	`),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	program, err := load.Load(context.Background(), load.Request{
		Directory: directory,
		Pattern:   ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	root, err := NewRoot(program.Roots()[0].Types().Scope().Lookup("Value"))
	if err != nil {
		t.Fatal(err)
	}
	emission, err := Compile(program, []Root{root})
	if err != nil {
		t.Fatal(err)
	}
	var encoded strings.Builder
	for _, file := range emission.Files() {
		payload, err := tsgo.EncodeSourceFile(file.SourceFile())
		if err != nil {
			t.Fatal(err)
		}
		encoded.Write(payload)
	}
	for _, required := range []string{
		"gotots-go-source-declaration-fact-v1",
		"checked-syntax:source.go",
		"local",
	} {
		if !strings.Contains(encoded.String(), required) {
			t.Fatalf("package initializer facts omit %q", required)
		}
	}
}
