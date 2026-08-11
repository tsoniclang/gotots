package verify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	externalcertify "github.com/tsoniclang/gotots/internal/contracts/externals/certify"
	gostdlibcertify "github.com/tsoniclang/gotots/internal/contracts/gostdlib/certify"
	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	"github.com/tsoniclang/gotots/internal/toolchain"
)

func TestCertifiedExternalModuleLinksBodylessFunctionExactly(t *testing.T) {
	profile := externalProviderBuildProfile(t)
	program, sourcePackage := loadExternalLinkageProgram(t, profile)
	run := closureRoot(t, sourcePackage, "Run")

	standardLibrary, externalProvider := externalProviderCertificates(t, profile)
	tests := []struct {
		name          string
		integer       emit.IntegerRepresentation
		resultLiteral string
		required      []string
		forbidden     []string
	}{
		{
			name:          "number product",
			integer:       emit.IntegerRepresentationNumber,
			resultLiteral: "0",
			required: []string{
				"goNumberToBigInt($argument0)",
				"globalThis.Number(BigInt.asUintN(64, __gotots_results_0[0]))",
			},
		},
		{
			name:          "bigint product",
			integer:       emit.IntegerRepresentationBigInt,
			resultLiteral: "0n",
			required: []string{
				"unix.Syscall($argument0, $argument1, $argument2, $argument3)",
			},
			forbidden: []string{
				"unix.Syscall(BigInt.asUintN(64, goNumberToBigInt(trap))",
				"globalThis.Number(BigInt.asUintN(64, __gotots_results_0[0]))",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options := emit.DefaultOptions()
			options.IntegerRepresentation = test.integer
			options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
			options.StandardLibrary = standardLibrary
			options.ExternalProvider = externalProvider
			linked, err := emit.CompileWithOptions(program, []emit.Root{run}, options)
			if err != nil {
				t.Fatal(err)
			}
			if remaining := linked.ExternalFunctionObligations(); len(remaining) != 0 {
				t.Fatalf("linked external obligations = %d, want 0", len(remaining))
			}
			artifacts := materializeClosureWithSetup(
				t,
				linked,
				func(directory string, emission emit.ProgramEmission) {
					installCertifiedProviderPackages(t, directory, emission)
				},
			)
			for _, required := range append(
				[]string{`from "@gotots/externals/golang.org/x/sys/unix.js"`},
				test.required...,
			) {
				if !strings.Contains(artifacts.printed, required) {
					t.Fatalf("linked artifact lacks %q:\n%s", required, artifacts.printed)
				}
			}
			for _, forbidden := range append(
				[]string{"unresolved external Go function"},
				test.forbidden...,
			) {
				if strings.Contains(artifacts.printed, forbidden) {
					t.Fatalf("linked artifact contains %q", forbidden)
				}
			}
			executeLinkedRun(t, linked, artifacts, test.resultLiteral)
		})
	}
}

func TestCertifiedExternalProviderProfileFailsClosed(t *testing.T) {
	profile := externalProviderBuildProfile(t)
	standardLibrary, externalProvider := externalProviderCertificates(t, profile)
	mutatedProfile, err := environmentcontract.NewBuildProfile(
		"linux",
		"amd64",
		false,
		[]string{"externalprofilemutation", "noasm"},
	)
	if err != nil {
		t.Fatal(err)
	}
	program, sourcePackage := loadExternalLinkageProgram(t, mutatedProfile)
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	options.StandardLibrary = standardLibrary
	options.ExternalProvider = externalProvider
	_, err = emit.CompileWithOptions(
		program,
		[]emit.Root{closureRoot(t, sourcePackage, "Run")},
		options,
	)
	var linkageError *emit.ExternalFunctionBindingError
	if !errors.As(err, &linkageError) ||
		linkageError.Reason != "provider target profile does not match compilation" {
		t.Fatalf("profile mutation error = %#v", err)
	}

	options.StandardLibrary = nil
	program, sourcePackage = loadExternalLinkageProgram(t, profile)
	_, err = emit.CompileWithOptions(
		program,
		[]emit.Root{closureRoot(t, sourcePackage, "Run")},
		options,
	)
	if !errors.As(err, &linkageError) ||
		linkageError.Reason != "provider and standard-library certificates do not exact-join" {
		t.Fatalf("certificate join error = %#v", err)
	}
}

func loadExternalLinkageProgram(
	t *testing.T,
	profile load.BuildProfile,
) (*load.Program, *load.Package) {
	t.Helper()
	program, err := load.Load(context.Background(), load.Request{
		Directory:            closureDirectory("external-linked"),
		Pattern:              ".",
		ContractDependencies: true,
		BuildProfile:         profile,
	})
	if err != nil {
		t.Fatal(err)
	}
	return program, program.Roots()[0]
}

func externalProviderBuildProfile(t *testing.T) environmentcontract.BuildProfile {
	t.Helper()
	profile, err := environmentcontract.NewBuildProfile(
		"linux",
		"amd64",
		false,
		[]string{"noasm"},
	)
	if err != nil {
		t.Fatal(err)
	}
	return profile
}

func externalProviderCertificates(
	t *testing.T,
	profile environmentcontract.BuildProfile,
) (*gostdlibcertify.Certificate, *externalcertify.Certificate) {
	t.Helper()
	repository := repositoryRoot(t)
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
	return standardLibrary, externalProvider
}

func installCertifiedProviderPackages(
	t *testing.T,
	directory string,
	_ emit.ProgramEmission,
) {
	t.Helper()
	packageRoot := filepath.Join(directory, "node_modules", "@gotots")
	if err := os.MkdirAll(packageRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	copyCertifiedProviderPackage(t, packageRoot, "gostdlib")
	copyCertifiedProviderPackage(t, packageRoot, "externals")
	if err := os.Symlink(
		"../../runtime",
		filepath.Join(packageRoot, "runtime"),
	); err != nil {
		t.Fatal(err)
	}
}

func copyCertifiedProviderPackage(
	t *testing.T,
	packageRoot string,
	name string,
) {
	t.Helper()
	source := filepath.Join(repositoryRoot(t), name)
	target := filepath.Join(packageRoot, name)
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	packageDocument, err := os.ReadFile(filepath.Join(source, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(target, "package.json"),
		packageDocument,
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(
		filepath.Join(target, "dist"),
		os.DirFS(filepath.Join(source, "dist")),
	); err != nil {
		t.Fatal(err)
	}
}
