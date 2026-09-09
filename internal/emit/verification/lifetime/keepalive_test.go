package lifetime_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestKeepAliveUsesOneNeutralBodyAcrossCallableUses(test *testing.T) {
	repository, err := filepath.Abs("../../../..")
	if err != nil {
		test.Fatal(err)
	}
	project := test.TempDir()
	for name, text := range map[string]string{
		"go.mod": "module example.com/lifetime\n\ngo 1.26.4\n",
		"source.go": `package lifetime
import "runtime"

func KeepAlive(value any) { _ = value }
func Retainer() func(any) { return runtime.KeepAlive }

func Check(value *uint32) {
	runtime.KeepAlive(value)
	alias := runtime.KeepAlive
	alias(value)
	returned := Retainer()
	returned(value)
	defer runtime.KeepAlive(value)
	KeepAlive(value)
}
`,
	} {
		if err := os.WriteFile(filepath.Join(project, name), []byte(text), 0o600); err != nil {
			test.Fatal(err)
		}
	}
	program, err := load.Load(context.Background(), load.Request{
		Directory: project, Pattern: ".", ToolCacheRoot: filepath.Join(repository, ".temp", "cache", "toolchain-tests"),
	})
	if err != nil {
		test.Fatal(err)
	}
	root, err := emit.NewRoot(program.Roots()[0].Types().Scope().Lookup("Check"))
	if err != nil {
		test.Fatal(err)
	}
	emission, err := emit.Compile(program, []emit.Root{root})
	if err != nil {
		test.Fatal(err)
	}
	observed := 0
	for _, obligation := range emission.EnvironmentObligations() {
		if obligation.PackagePath() == "runtime" && obligation.Name() == "KeepAlive" {
			observed++
			if obligation.Route() != environmentcontract.RouteGeneratedFacet {
				test.Fatal("KeepAlive retained a provider or placeholder route")
			}
		}
	}
	if observed != 1 {
		test.Fatalf("KeepAlive environment identities: got %d, want 1", observed)
	}
	client, err := tsgo.StartClient(repository, project)
	if err != nil {
		test.Fatal(err)
	}
	test.Cleanup(func() {
		if err := client.Close(); err != nil {
			test.Error(err)
		}
	})
	var printed strings.Builder
	for _, file := range emission.Files() {
		text, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			test.Fatal(err)
		}
		printed.WriteString(text)
	}
	if count := strings.Count(printed.String(), "keepAlive(value)"); count != 1 {
		test.Fatalf("neutral barrier bodies: got %d, want 1", count)
	}
	for _, required := range []string{"function goKeepAlive(", "function KeepAlive(", "= goKeepAlive;", "return goKeepAlive;"} {
		if !strings.Contains(printed.String(), required) {
			test.Fatalf("generated lifetime output lacks %q", required)
		}
	}
}
