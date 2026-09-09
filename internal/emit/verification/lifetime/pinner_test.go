package lifetime_test

import (
	"context"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
)

func TestPinnerPreservesProviderObligations(test *testing.T) {
	repository, err := filepath.Abs("../../../..")
	if err != nil {
		test.Fatal(err)
	}
	project := test.TempDir()
	for name, text := range map[string]string{
		"go.mod": "module example.com/pinner\n\ngo 1.26.4\n",
		"source.go": `package lifetime
import "runtime"

func Check(value *uint32) {
	var pinner runtime.Pinner
	pinner.Pin(value)
	pin := pinner.Pin
	pin(value)
	invoke := (*runtime.Pinner).Pin
	invoke(&pinner, value)
	defer pinner.Unpin()
	pinner.Unpin()
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
	observed := make(map[string]int)
	runtimePackage := program.Roots()[0].Types().Imports()[0]
	pinner, ok := runtimePackage.Scope().Lookup("Pinner").(*types.TypeName)
	if !ok {
		test.Fatal("selected runtime has no Pinner type")
	}
	methods := types.NewMethodSet(types.NewPointer(pinner.Type()))
	for _, obligation := range emission.EnvironmentObligations() {
		if obligation.PackagePath() == "runtime" {
			observed[obligation.Name()]++
			if obligation.Name() == "Pin" || obligation.Name() == "Unpin" {
				selection := methods.Lookup(runtimePackage, obligation.Name())
				if selection == nil {
					test.Fatal("selected Pinner method is missing")
				}
				contract, err := environmentcontract.Describe(selection.Obj())
				if err != nil {
					test.Fatal(err)
				}
				if obligation.Identity() != contract.Identity() || obligation.SourceSignature() != contract.Signature() {
					test.Fatalf("%s lost its selected source contract", obligation.Name())
				}
				if obligation.Route() == environmentcontract.RouteGeneratedFacet {
					test.Fatal("Pinner was silently replaced with a generated lifetime helper")
				}
			}
		}
	}
	for _, name := range []string{"Pinner", "Pin", "Unpin"} {
		if observed[name] != 1 {
			test.Fatalf("runtime.%s obligations: got %d, want 1", name, observed[name])
		}
	}
	if observed["KeepAlive"] != 0 {
		test.Fatal("Pinner lifetime was replaced with lexical KeepAlive")
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
	var artifacts strings.Builder
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			test.Fatal(err)
		}
		artifacts.WriteString(printed)
	}
	for _, required := range []string{
		"export declare function Pinner_Pin($receiver: Pointer<Pinner> | undefined, $argument0: GoInterface | undefined): void;",
		"export declare function Pinner_Unpin($receiver: Pointer<Pinner> | undefined): void;",
		"Pinner_Pin__from_runtime(addressOf<Pinner__from_runtime>(pinner), new GoInterfaceAdapter(value))",
		"Pinner_Unpin__from_runtime(addressOf<Pinner__from_runtime>(pinner))",
	} {
		if !strings.Contains(artifacts.String(), required) {
			test.Fatalf("canonical Pinner output lacks %q", required)
		}
	}
	if strings.Contains(artifacts.String(), "keepAlive(") {
		test.Fatal("canonical pin set became a lexical reachability barrier")
	}
}
