package provider_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	environmentcontract "github.com/tsoniclang/gotots/internal/contracts/environment"
	"github.com/tsoniclang/gotots/internal/contracts/gostdlib/certify"
	"github.com/tsoniclang/gotots/internal/emit"
	"github.com/tsoniclang/gotots/internal/target/tsgo"
	corefixture "github.com/tsoniclang/gotots/internal/testfixture/tsoniccore"
	"github.com/tsoniclang/gotots/internal/toolchain"
)

type renderedArtifacts struct {
	paths   []string
	printed string
}

var (
	linkedCertificateOnce  sync.Once
	linkedCertificate      *certify.Certificate
	linkedCertificateError error
)

func providerNumberOptions() emit.Options {
	return emit.Options{
		IntegerRepresentation: emit.IntegerRepresentationNumber,
		EvaluationOrder:       emit.EvaluationOrderDirect,
	}
}

func materializeArtifacts(
	t *testing.T,
	emission emit.ProgramEmission,
	workingDirectory string,
) renderedArtifacts {
	t.Helper()
	client, err := tsgo.StartClient(repositoryRoot(), workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close TS-Go client: %v", err)
		}
	})
	var result renderedArtifacts
	for _, file := range emission.Files() {
		printed, err := client.PrintNode(file.SourceFile(), tsgo.PrintOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tsgo.EncodeSourceFile(file.SourceFile()); err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{
			": any",
			": unknown",
			" as any",
			" as unknown",
			".call(",
			".apply(",
			".bind(",
			"import(",
		} {
			if strings.Contains(printed, forbidden) {
				t.Fatalf(
					"%s contains forbidden %q:\n%s",
					file.OutputPath(),
					forbidden,
					printed,
				)
			}
		}
		targetPath := filepath.Join(
			workingDirectory,
			filepath.FromSlash(file.OutputPath()),
		)
		writeProgramFile(t, targetPath, printed)
		result.paths = append(result.paths, targetPath)
		result.printed += "\n// " + file.OutputPath() + "\n" + printed
	}
	if len(result.paths) == 0 {
		t.Fatal("provider fixture emitted no target files")
	}
	return result
}

func waveThreeTypecheck(
	t *testing.T,
	workingDirectory string,
	paths []string,
) {
	t.Helper()
	if err := corefixture.InstallResolutionOnly(workingDirectory); err != nil {
		t.Fatal(err)
	}
	writeProgramFile(
		t,
		filepath.Join(workingDirectory, "package.json"),
		"{\"type\":\"module\"}\n",
	)
	providerRoot, err := filepath.Abs(
		filepath.Join(repositoryRoot(), "gostdlib"),
	)
	if err != nil {
		t.Fatal(err)
	}
	packageRoot := filepath.Join(
		workingDirectory,
		"node_modules",
		"@gotots",
	)
	if err := os.MkdirAll(packageRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	providerPackage := filepath.Join(packageRoot, "gostdlib")
	if err := os.MkdirAll(providerPackage, 0o755); err != nil {
		t.Fatal(err)
	}
	packageDocument, err := os.ReadFile(
		filepath.Join(providerRoot, "package.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(providerPackage, "package.json"),
		packageDocument,
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(
		filepath.Join(providerPackage, "dist"),
		os.DirFS(filepath.Join(providerRoot, "dist")),
	); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(
		"../../runtime",
		filepath.Join(packageRoot, "runtime"),
	); err != nil {
		t.Fatal(err)
	}
	arguments := []string{
		"--target", "es2022",
		"--module", "nodenext",
		"--moduleResolution", "nodenext",
		"--strict",
		"--noUncheckedIndexedAccess",
		"--outDir", filepath.Join(workingDirectory, "out"),
	}
	arguments = append(arguments, paths...)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		arguments,
	); err != nil {
		t.Fatal(err)
	}
}

func typecheckProviderRunner(
	t *testing.T,
	workingDirectory string,
	paths []string,
	assemblyPath string,
	exports []string,
	runnerBody string,
) {
	t.Helper()
	waveThreeTypecheck(t, workingDirectory, paths)
	runnerPath := filepath.Join(workingDirectory, "runner.ts")
	modulePath := "./" + strings.TrimSuffix(assemblyPath, ".ts") + ".js"
	writeProgramFile(t, runnerPath, `import "./program.js";
import { `+strings.Join(exports, ", ")+` } from "`+modulePath+`";

`+runnerBody)
	outputDirectory := filepath.Join(workingDirectory, "out")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := tsgo.Compile(
		ctx,
		repositoryRoot(),
		workingDirectory,
		[]string{
			"--target", "es2022",
			"--module", "nodenext",
			"--moduleResolution", "nodenext",
			"--strict",
			"--noUncheckedIndexedAccess",
			"--outDir", outputDirectory,
			runnerPath,
		},
	); err != nil {
		t.Fatal(err)
	}
}

func linkedProviderCertificate(t *testing.T) *certify.Certificate {
	t.Helper()
	linkedCertificateOnce.Do(func() {
		repository := repositoryRoot()
		selectedGo, err := toolchain.ResolveGo(
			"",
			filepath.Join(t.TempDir(), ".temp", "cache", "toolchain"),
		)
		if err != nil {
			linkedCertificateError = err
			return
		}
		selectedTSGo, err := tsgo.ResolveTool(selectedGo, repository, "")
		if err != nil {
			linkedCertificateError = err
			return
		}
		linkedCertificate, linkedCertificateError = certify.Verify(certify.Config{
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
			BuildProfile:        linkedProviderBuildProfile(t),
			Backend:             "node",
			MinimumGoVersion:    "go1.26.4",
			MaximumGoVersion:    "go1.26.4",
		})
	})
	if linkedCertificateError != nil {
		t.Fatal(linkedCertificateError)
	}
	if linkedCertificate == nil {
		t.Fatal("linked provider certificate is absent")
	}
	return linkedCertificate
}

func linkedProviderBuildProfile(t *testing.T) environmentcontract.BuildProfile {
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

func writeProgramFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func repositoryRoot() string {
	return filepath.Join("..", "..", "..", "..")
}
