// The deterministic staged-generation stage: census evidence and
// generated output both byte-identical across independent runs, staged
// as a complete root and swapped atomically.
package main

import (
	"bytes"
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
	for _, name := range []string{"inventory.json", "census.json", "declarations.json", "externals.json", "environment.json", "manifest.json"} {
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
	// Complete evidence determinism: proofs, support, ownership,
	// withholding, and module imports must be identical across runs —
	// byte-identical files alone do not prove the evidence ledgers
	// regenerated identically.
	for name, pair := range map[string][2]any{
		"proofs":        {corpusGenerated.Proofs, regenerated.Proofs},
		"support":       {corpusGenerated.Support, regenerated.Support},
		"ownership":     {corpusGenerated.Ownership, regenerated.Ownership},
		"withheld":      {corpusGenerated.Withheld, regenerated.Withheld},
		"moduleImports": {corpusGenerated.ModuleImports, regenerated.ModuleImports},
	} {
		firstJSON, err := json.Marshal(pair[0])
		if err != nil {
			return "fail", nil, err
		}
		secondJSON, err := json.Marshal(pair[1])
		if err != nil {
			return "fail", nil, err
		}
		if !bytes.Equal(firstJSON, secondJSON) {
			return "fail", []string{name}, fmt.Errorf("nondeterministic %s evidence across two runs", name)
		}
	}
	publicationRoot := filepath.Join(staging, "publication")
	stage := func() error {
		stagingRoot := publicationRoot + ".staging"
		for path, content := range corpusGenerated.Files {
			target := filepath.Join(stagingRoot, filepath.FromSlash(path))
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
				return err
			}
		}
		// Replace-existing semantics: an accepted root may already exist;
		// publication retires it only after the complete staging root is
		// ready, never leaving a partial tree.
		replaced := publicationRoot + ".retired"
		if _, err := os.Stat(publicationRoot); err == nil {
			if err := os.Rename(publicationRoot, replaced); err != nil {
				return fmt.Errorf("retire existing accepted root: %w", err)
			}
		}
		if err := os.Rename(stagingRoot, publicationRoot); err != nil {
			return fmt.Errorf("atomic staging swap: %w", err)
		}
		return os.RemoveAll(replaced)
	}
	// First publication into an empty destination, then a second
	// publication REPLACING the accepted root — both paths exercised.
	if err := stage(); err != nil {
		return "fail", nil, err
	}
	if err := stage(); err != nil {
		return "fail", nil, fmt.Errorf("replacing an existing accepted root: %w", err)
	}
	return "pass", []string{
		"census evidence deterministic across two hermetic runs",
		fmt.Sprintf("generated output deterministic across two runs (%d files byte-identical)", len(corpusGenerated.Files)),
		"complete evidence (proofs, support, ownership, withholding, module imports) deterministic across two runs",
		"emission staged as a complete root, swapped atomically, and re-published over an existing accepted root",
		fmt.Sprintf("partial bundle: %d packages withheld as honest unimplemented", len(corpusGenerated.Withheld)),
	}, nil
}
