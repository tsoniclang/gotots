// The strict TypeScript staticness stage: generate the owned corpus,
// stage it with the pinned strict configuration, and let the pinned
// compiler judge it.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tsoniclang/gotots/internal/census"
	"github.com/tsoniclang/gotots/internal/goenv"
	"github.com/tsoniclang/gotots/internal/pinning"
	"github.com/tsoniclang/gotots/internal/productinputs"
	"github.com/tsoniclang/gotots/internal/profile"
	"github.com/tsoniclang/gotots/internal/translate"
)

func runTscGate(repoDir, profilePath, buildProfile, sourceDir string, report *GateReport, productPin *productinputs.Pin, firstRun *census.Result) (string, []string, error) {
	if firstRun == nil || productPin == nil {
		return "blocked", []string{"input attestation did not complete"}, nil
	}
	// The compiler materialization must match the declared identity
	// before it is trusted to judge staticness.
	tscJs, err := filepath.Abs(filepath.Join(repoDir, "product", "node_modules", "typescript", "lib", "tsc.js"))
	if err != nil {
		return "fail", nil, err
	}
	tscDigest, err := digestFile(tscJs)
	if err != nil {
		return "blocked", []string{"typescript compiler not materialized under product/node_modules (install the pinned typescript version)"}, nil
	}
	if tscDigest != productPin.TypescriptCompiler.TscJsSha256 {
		return "fail", nil, fmt.Errorf("materialized tsc.js digest %s does not match pinned %s", tscDigest, productPin.TypescriptCompiler.TscJsSha256)
	}
	prof, err := profile.Load(filepath.Join(repoDir, filepath.FromSlash(profilePath)))
	if err != nil {
		return "fail", nil, err
	}
	build, err := prof.BuildProfileByName(buildProfile)
	if err != nil {
		return "fail", nil, err
	}
	resolved, err := pinning.VerifyToolchain(prof.Pin)
	if err != nil {
		return "fail", nil, err
	}
	env := resolved.Environ(goenv.EnvOptions{
		GOOS: build.GOOS, GOARCH: build.GOARCH,
		GOAMD64: build.GOAMD64, GOARM64: build.GOARM64,
	})
	generated, err := translate.Corpus(prof, env, sourceDir, translate.Options{
		SourceRevision: report.Inputs.SourceRevision,
		ProfileHash:    report.Inputs.ProfileSha256,
	})
	if err != nil {
		return "fail", nil, err
	}
	staging, err := os.MkdirTemp("", "gotots-tsc-")
	if err != nil {
		return "fail", nil, err
	}
	defer os.RemoveAll(staging)
	for path, content := range generated.Files {
		target := filepath.Join(staging, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "fail", nil, err
		}
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			return "fail", nil, err
		}
	}
	strictConfig, err := os.ReadFile(filepath.Join(repoDir, "product", "tsconfig.strict.json"))
	if err != nil {
		return "fail", nil, err
	}
	if err := os.WriteFile(filepath.Join(staging, "tsconfig.json"), strictConfig, 0o644); err != nil {
		return "fail", nil, err
	}
	// The generated output is strict ESM.
	if err := os.WriteFile(filepath.Join(staging, "package.json"), []byte("{\n  \"type\": \"module\"\n}\n"), 0o644); err != nil {
		return "fail", nil, err
	}
	out, err := runInRepo(staging, "node", tscJs, "-p", ".")
	if err != nil {
		lines := splitLines(out)
		if len(lines) > 40 {
			lines = append(lines[:40], fmt.Sprintf("... and %d more lines", len(lines)-40))
		}
		return "fail", lines, fmt.Errorf("strict typecheck rejected the generated output")
	}
	return "pass", []string{
		fmt.Sprintf("strict tsc (%s@%s) accepted %d generated files", productPin.TypescriptCompiler.Package, productPin.TypescriptCompiler.Version, len(generated.Files)),
		fmt.Sprintf("withheld packages (honest unimplemented, not typechecked): %d", len(generated.Withheld)),
	}, nil
}
