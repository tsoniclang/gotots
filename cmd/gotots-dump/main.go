// Corpus dump: full generation written to GOTOTS_DUMP_DIR for offline
// measurement (byte attribution, shape analysis, calibration joins,
// fingerprinting). Alongside the generated files it persists the typed
// producer ledgers under ledgers/ so every downstream consumer joins
// through recorded facts, never through generated text. Not part of
// the product.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tsoniclang/gotots/internal/census"
	"github.com/tsoniclang/gotots/internal/goenv"
	"github.com/tsoniclang/gotots/internal/pinning"
	"github.com/tsoniclang/gotots/internal/profile"
	"github.com/tsoniclang/gotots/internal/translate"
)

func writeJSON(dumpDir, name string, value any) error {
	data, err := json.MarshalIndent(value, "", " ")
	if err != nil {
		return err
	}
	full := filepath.Join(dumpDir, "ledgers", name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, append(data, '\n'), 0o644)
}

func main() {
	sourceDir := os.Getenv("GOTOTS_CORPUS_DIR")
	dumpDir := os.Getenv("GOTOTS_DUMP_DIR")
	if sourceDir == "" || dumpDir == "" {
		fmt.Fprintln(os.Stderr, "set GOTOTS_CORPUS_DIR and GOTOTS_DUMP_DIR")
		os.Exit(1)
	}
	prof, err := profile.Load("profiles/tsts/project.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	build, err := prof.BuildProfileByName("linux-amd64")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	resolved, err := pinning.VerifyToolchain(prof.Pin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	env := resolved.Environ(goenv.EnvOptions{GOOS: build.GOOS, GOARCH: build.GOARCH, GOAMD64: build.GOAMD64, GOARM64: build.GOARM64})
	run, err := census.Run(prof, sourceDir, "linux-amd64")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	g, err := translate.Corpus(prof, env, sourceDir, translate.Options{SourceRevision: prof.Pin.Revision, ProfileHash: prof.Hash})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for path, content := range g.Files {
		full := filepath.Join(dumpDir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	// The typed producer ledgers: the offline join surface.
	ledgers := map[string]any{
		"declarations.json":             run.Report.Declarations,
		"implementation-artifacts.json": g.ImplementationArtifacts,
		"emissions.json":                g.Emissions,
		"extern-symbols.json":           g.ExternSymbols,
		"module-dispositions.json":      g.ModuleDispositions,
		"support.json":                  g.Support,
		"func-lits.json":                g.FuncLits,
		"proofs.json":                   g.Proofs,
		"withheld.json":                 g.Withheld,
		"not-materialized.json":         g.NotMaterialized,
	}
	for name, value := range ledgers {
		if err := writeJSON(dumpDir, name, value); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	fmt.Printf("dumped %d files + %d ledgers; defects=%d\n", len(g.Files), len(ledgers), len(g.EmitterDefects))
}
