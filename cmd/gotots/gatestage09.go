// The deterministic staged-generation stage: census evidence and
// generated output both byte-identical across independent runs, staged
// as a complete root and swapped atomically.
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tsoniclang/gotots/internal/census"
	"github.com/tsoniclang/gotots/internal/goenv"
	"github.com/tsoniclang/gotots/internal/pinning"
	"github.com/tsoniclang/gotots/internal/profile"
	"github.com/tsoniclang/gotots/internal/translate"
)

func runStagedGenerationGate(repoDir, profilePath, buildProfile, sourceDir string, report *GateReport, firstRun *census.Result, corpusGenerated *translate.Generated) (string, []string, error) {
	if firstRun == nil {
		return "blocked", []string{"census did not run"}, nil
	}
	prof, err := profile.Load(filepath.Join(repoDir, filepath.FromSlash(profilePath)))
	if err != nil {
		return "fail", nil, err
	}
	secondRun, err := census.Run(prof, sourceDir, buildProfile)
	if err != nil {
		return "fail", nil, err
	}
	staging := filepath.Join(os.TempDir(), fmt.Sprintf("gotots-gate-%d", os.Getpid()))
	defer os.RemoveAll(staging)
	first := filepath.Join(staging, "run1")
	second := filepath.Join(staging, "run2")
	if err := census.WriteReports(firstRun, first); err != nil {
		return "fail", nil, err
	}
	if err := census.WriteReports(secondRun, second); err != nil {
		return "fail", nil, err
	}
	var mismatches []string
	for _, name := range []string{"inventory.json", "census.json", "declarations.json", "externals.json"} {
		firstBytes, err := os.ReadFile(filepath.Join(first, name))
		if err != nil {
			return "fail", nil, err
		}
		secondBytes, err := os.ReadFile(filepath.Join(second, name))
		if err != nil {
			return "fail", nil, err
		}
		if !bytes.Equal(firstBytes, secondBytes) {
			mismatches = append(mismatches, name)
		}
	}
	if len(mismatches) > 0 {
		return "fail", mismatches, fmt.Errorf("nondeterministic reports")
	}
	if _, err := census.VerifyBundle(first); err != nil {
		return "fail", nil, err
	}
	// Generated output: two independent corpus generations must be
	// byte-identical, and the emission stages into a complete root
	// swapped into place atomically — never a partial tree.
	if corpusGenerated == nil {
		return "blocked", []string{"corpus generation did not run"}, nil
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
	regenerated, err := translate.Corpus(prof, env, sourceDir, translate.Options{
		SourceRevision: report.Inputs.SourceRevision,
		ProfileHash:    report.Inputs.ProfileSha256,
	})
	if err != nil {
		return "fail", nil, err
	}
	if len(regenerated.Files) != len(corpusGenerated.Files) {
		return "fail", nil, fmt.Errorf("nondeterministic generation: %d vs %d files", len(corpusGenerated.Files), len(regenerated.Files))
	}
	for path, content := range corpusGenerated.Files {
		if regenerated.Files[path] != content {
			return "fail", []string{path}, fmt.Errorf("nondeterministic generation")
		}
	}
	publicationRoot := filepath.Join(staging, "publication")
	stagingRoot := publicationRoot + ".staging"
	for path, content := range corpusGenerated.Files {
		target := filepath.Join(stagingRoot, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "fail", nil, err
		}
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			return "fail", nil, err
		}
	}
	if err := os.Rename(stagingRoot, publicationRoot); err != nil {
		return "fail", nil, fmt.Errorf("atomic staging swap: %w", err)
	}
	return "pass", []string{
		"census evidence deterministic across two hermetic runs",
		fmt.Sprintf("generated output deterministic across two runs (%d files byte-identical)", len(corpusGenerated.Files)),
		"emission staged as a complete root and swapped atomically",
		fmt.Sprintf("partial bundle: %d packages withheld as honest unimplemented", len(corpusGenerated.Withheld)),
	}, nil
}
