package verify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	externalcertify "github.com/tsoniclang/gotots/internal/contracts/externals/certify"
	gostdlibcertify "github.com/tsoniclang/gotots/internal/contracts/gostdlib/certify"
	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/load"
)

func TestCertifiedExternalModuleLinksBodylessFunctionExactly(t *testing.T) {
	profile := externalProviderBuildProfile(t)
	program, sourcePackage := loadExternalLinkageProgram(t, profile)
	run := closureRoot(t, sourcePackage, "Run")

	unlinkedOptions := emit.DefaultOptions()
	unlinkedOptions.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	unlinked, err := emit.CompileWithOptions(
		program,
		[]emit.Root{run},
		unlinkedOptions,
	)
	if err != nil {
		t.Fatal(err)
	}
	obligations := unlinked.ExternalFunctionObligations()
	if len(obligations) != 2 {
		t.Fatalf("unlinked external obligations = %#v", obligations)
	}
	byName := make(map[string]emit.ExternalFunctionObligation, len(obligations))
	for _, obligation := range obligations {
		byName[obligation.Function().Name()] = obligation
	}
	for _, name := range []string{"Syscall", "Syscall6"} {
		obligation, ok := byName[name]
		if !ok || obligation.ModulePath() != "golang.org/x/sys" ||
			obligation.ModuleVersion() != "v0.46.0" {
			t.Fatalf("unlinked %s obligation = %#v", name, obligation)
		}
	}

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
				"goNumberToBigInt(trap)",
				"globalThis.Number(BigInt.asUintN(64, __gotots_results_0[0]))",
			},
		},
		{
			name:          "bigint product",
			integer:       emit.IntegerRepresentationBigInt,
			resultLiteral: "0n",
			required:      []string{"unix.Syscall(trap, a1, a2, a3)"},
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
					t.Fatalf("linked artifact lacks %q", required)
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

func TestCertifiedExternalSourceLinksBodylessFunctionExactly(t *testing.T) {
	profile := externalProviderBuildProfile(t)
	program, err := load.Load(context.Background(), load.Request{
		Directory:    repositoryRoot(t),
		Pattern:      "github.com/zeebo/xxh3",
		BuildProfile: profile,
	})
	if err != nil {
		t.Fatal(err)
	}
	sourcePackage := program.Roots()[0]
	standardLibrary, externalProvider := externalProviderCertificates(t, profile)
	options := emit.DefaultOptions()
	options.ConcurrencySemantics = emit.ConcurrencySemanticsCooperative
	options.StandardLibrary = standardLibrary
	options.ExternalProvider = externalProvider
	linked, err := emit.CompileWithOptions(
		program,
		[]emit.Root{closureRoot(t, sourcePackage, "accumSSE")},
		options,
	)
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
	if !strings.Contains(artifacts.printed, "return accumScalar(") ||
		strings.Contains(artifacts.printed, "unresolved external Go function") {
		t.Fatalf("source-linked artifact is not exact:\n%s", artifacts.printed)
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
		Directory:    closureDirectory("external-linked"),
		Pattern:      ".",
		BuildProfile: profile,
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
	standardLibrary, err := gostdlibcertify.Verify(gostdlibcertify.Config{
		RepositoryRoot:      repository,
		ProviderRoot:        filepath.Join(repository, "gostdlib"),
		ManifestPath:        filepath.Join(repository, "gostdlib", "contract", "manifest.json"),
		ModuleMapPath:       filepath.Join(repository, "gostdlib", "contract", "modules.json"),
		FacetMapPath:        filepath.Join(repository, "gostdlib", "contract", "facets.json"),
		RuntimeContractPath: filepath.Join(repository, "gostdlib", "contract", "runtime.json"),
		TSConfigPath:        filepath.Join(repository, "gostdlib", "tsconfig.json"),
		ScratchDirectory:    t.TempDir(),
		GoBinary:            "go",
		BuildProfile:        profile,
		Backend:             "node",
		MinimumGoVersion:    runtime.Version(),
		MaximumGoVersion:    runtime.Version(),
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
