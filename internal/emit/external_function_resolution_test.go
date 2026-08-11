package emit

import (
	"context"
	"go/types"
	"path/filepath"
	"runtime"
	"testing"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	externalcertify "github.com/tsoniclang/gotots/internal/contracts/externals/certify"
	gostdlibcertify "github.com/tsoniclang/gotots/internal/contracts/gostdlib/certify"
	declarationindex "github.com/tsoniclang/gotots/internal/emit/declaration/index"
	externalfunction "github.com/tsoniclang/gotots/internal/emit/externalfunction"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"github.com/tsoniclang/gotots/internal/toolchain"
)

func TestExternalSourceBindingSelectsExactPortableFunction(t *testing.T) {
	repository := emitRepositoryRoot(t)
	selectedGo, err := toolchain.ResolveGo(
		"",
		filepath.Join(t.TempDir(), ".temp", "cache", "toolchain"),
	)
	if err != nil {
		t.Fatal(err)
	}
	selectedTSGo, err := tsgo.ResolveTool(selectedGo, repository, "")
	if err != nil {
		t.Fatal(err)
	}
	profile, err := environmentcontract.NewBuildProfileForToolchain(
		selectedGo.Version(),
		"linux",
		"amd64",
		false,
		[]string{"noasm"},
	)
	if err != nil {
		t.Fatal(err)
	}
	program, err := load.Load(context.Background(), load.Request{
		Directory:    repository,
		Pattern:      "github.com/zeebo/xxh3",
		BuildProfile: profile,
		GoTool:       selectedGo,
	})
	if err != nil {
		t.Fatal(err)
	}
	standardLibrary, err := gostdlibcertify.Verify(gostdlibcertify.Config{
		RepositoryRoot:      repository,
		ProviderRoot:        filepath.Join(repository, "gostdlib"),
		ManifestPath:        filepath.Join(repository, "gostdlib", "contract", "manifest.json"),
		ModuleMapPath:       filepath.Join(repository, "gostdlib", "contract", "modules.json"),
		FacetMapPath:        filepath.Join(repository, "gostdlib", "contract", "facets.json"),
		RuntimeContractPath: filepath.Join(repository, "gostdlib", "contract", "runtime.json"),
		TSConfigPath:        filepath.Join(repository, "gostdlib", "tsconfig.json"),
		ScratchDirectory:    t.TempDir(),
		GoTool:              selectedGo,
		TSGoTool:            selectedTSGo,
		BuildProfile:        profile,
		Backend:             "node",
		MinimumGoVersion:    selectedGo.Version(),
		MaximumGoVersion:    selectedGo.Version(),
	})
	if err != nil {
		t.Fatal(err)
	}
	externalProvider, err := externalcertify.Verify(externalcertify.Config{
		RepositoryRoot:              repository,
		ProviderRoot:                filepath.Join(repository, "externals"),
		ManifestPath:                filepath.Join(repository, "externals", "contract", "manifest.json"),
		BindingMapPath:              filepath.Join(repository, "externals", "contract", "bindings.json"),
		TSConfigPath:                filepath.Join(repository, "externals", "tsconfig.json"),
		StandardLibraryManifestPath: filepath.Join(repository, "gostdlib", "contract", "manifest.json"),
		StandardLibraryRuntimePath:  filepath.Join(repository, "gostdlib", "contract", "runtime.json"),
		BuildProfile:                profile,
		Backend:                     "node",
		GoTool:                      selectedGo,
		TSGoTool:                    selectedTSGo,
	})
	if err != nil {
		t.Fatal(err)
	}
	sites, err := declarationindex.Program(program)
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := programScalarABI(program, IntegerRepresentationNumber)
	if err != nil {
		t.Fatal(err)
	}
	providerScalar, err := certifiedProviderScalarABI(
		standardLibrary,
		scalar.NativeIntegerWidth(),
	)
	if err != nil {
		t.Fatal(err)
	}
	resolved, _, err := externalfunction.Resolve(
		program,
		sites,
		externalProvider,
		standardLibrary,
		providerScalar,
	)
	if err != nil {
		t.Fatal(err)
	}
	scope := program.Roots()[0].Types().Scope()
	accelerated, acceleratedOK := scope.Lookup("accumSSE").(*types.Func)
	portable, portableOK := scope.Lookup("accumScalar").(*types.Func)
	if !acceleratedOK || !portableOK {
		t.Fatal("external source functions are absent")
	}
	target, ok := resolved[accelerated]
	if !ok {
		t.Fatal("certified source binding was not selected")
	}
	implementation, ok := target.Source()
	if !ok || implementation != portable {
		t.Fatalf("portable source target = %#v, want accumScalar", implementation)
	}
}

func emitRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", ".."))
}
