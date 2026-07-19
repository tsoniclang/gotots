// The deterministic staged-generation stage: census evidence and
// generated output both byte-identical across independent runs, staged
// as a complete root and swapped atomically.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
	// RELOCATED-CHECKOUT reproducibility: the same census from a copy of
	// the checkout at a different filesystem root must produce byte-
	// identical SEMANTIC reports; environment.json (the machine-evidence
	// record) compares under source-root normalization — any other
	// absolute path leaking into semantic evidence fails here.
	relocated := filepath.Join(staging, "relocated-src")
	if out, err := runInRepo(".", "cp", "-a", sourceDir, relocated); err != nil {
		return "fail", splitLines(out), fmt.Errorf("relocate checkout: %w", err)
	}
	relocatedRun, err := census.Run(prof, relocated, buildProfile)
	if err != nil {
		return "fail", nil, fmt.Errorf("relocated census: %w", err)
	}
	third := filepath.Join(staging, "run3")
	if err := census.WriteReports(relocatedRun, third); err != nil {
		return "fail", nil, err
	}
	absSource, err := filepath.Abs(sourceDir)
	if err != nil {
		return "fail", nil, err
	}
	var relocatedMismatches []string
	for _, name := range []string{"inventory.json", "census.json", "declarations.json", "externals.json", "environment.json", "manifest.json"} {
		firstBytes, err := os.ReadFile(filepath.Join(first, name))
		if err != nil {
			return "fail", nil, err
		}
		thirdBytes, err := os.ReadFile(filepath.Join(third, name))
		if err != nil {
			return "fail", nil, err
		}
		normalizedFirst := bytes.ReplaceAll(firstBytes, []byte(absSource), []byte("<sourceDir>"))
		normalizedThird := bytes.ReplaceAll(thirdBytes, []byte(relocated), []byte("<sourceDir>"))
		if !bytes.Equal(normalizedFirst, normalizedThird) {
			relocatedMismatches = append(relocatedMismatches, name)
		}
	}
	if len(relocatedMismatches) > 0 {
		return "fail", relocatedMismatches, fmt.Errorf("relocated-checkout reports differ beyond the source root (machine-local paths inside semantic evidence)")
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
		"proofs":            {corpusGenerated.Proofs, regenerated.Proofs},
		"support":           {corpusGenerated.Support, regenerated.Support},
		"ownership":         {corpusGenerated.Ownership, regenerated.Ownership},
		"withheld":          {corpusGenerated.Withheld, regenerated.Withheld},
		"moduleImports":     {corpusGenerated.ModuleImports, regenerated.ModuleImports},
		"moduleTypeImports": {corpusGenerated.ModuleTypeImports, regenerated.ModuleTypeImports},
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
	// Publication is immutable-bundle plus one atomic CURRENT-pointer
	// swap (the census bundle model): the new bundle materializes fully
	// under its own versioned root, then a symlink rename replaces the
	// pointer — a crash at ANY point leaves the previous accepted root
	// reachable; no sequence of steps removes it first.
	publicationDir := filepath.Join(staging, "publication")
	publish := func(version string) error {
		bundle := filepath.Join(publicationDir, "bundle-"+version)
		if _, err := os.Stat(bundle); err == nil {
			// Bundles are IMMUTABLE: republishing an existing version is
			// a protocol violation, never an overwrite.
			return fmt.Errorf("bundle version %s already published; bundles are immutable", version)
		}
		for path, content := range corpusGenerated.Files {
			target := filepath.Join(bundle, filepath.FromSlash(path))
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
				return err
			}
		}
		pointer := filepath.Join(publicationDir, "current")
		next := pointer + ".next"
		os.Remove(next)
		if err := os.Symlink("bundle-"+version, next); err != nil {
			return err
		}
		// One atomic rename: readers always resolve either the previous
		// or the new bundle, never nothing.
		return os.Rename(next, pointer)
	}
	if err := publish("one"); err != nil {
		return "fail", nil, err
	}
	firstTarget, err := os.Readlink(filepath.Join(publicationDir, "current"))
	if err != nil || firstTarget != "bundle-one" {
		return "fail", nil, fmt.Errorf("current pointer after first publication: %q %v", firstTarget, err)
	}
	if err := publish("two"); err != nil {
		return "fail", nil, fmt.Errorf("replacing the accepted root: %w", err)
	}
	secondTarget, err := os.Readlink(filepath.Join(publicationDir, "current"))
	if err != nil || secondTarget != "bundle-two" {
		return "fail", nil, fmt.Errorf("current pointer after replacement: %q %v", secondTarget, err)
	}
	// The previous accepted bundle is still intact after replacement (a
	// failed swap can never have removed it).
	if _, err := os.Stat(filepath.Join(publicationDir, "bundle-one")); err != nil {
		return "fail", nil, fmt.Errorf("previous accepted bundle removed by replacement: %w", err)
	}
	// Immutability: republishing an existing version must fail.
	if err := publish("two"); err == nil {
		return "fail", nil, fmt.Errorf("same-version republication was not rejected")
	}
	// Injected swap failure: with the pointer's staging name occupied by
	// a directory, the atomic rename fails — and the accepted pointer
	// must remain exactly where it was.
	if err := os.MkdirAll(filepath.Join(publicationDir, "current.next", "block"), 0o755); err != nil {
		return "fail", nil, err
	}
	if err := publish("three"); err == nil {
		return "fail", nil, fmt.Errorf("injected swap failure did not surface")
	}
	target, err := os.Readlink(filepath.Join(publicationDir, "current"))
	if err != nil || target != "bundle-two" {
		return "fail", nil, fmt.Errorf("accepted pointer damaged by failed swap: %q %v", target, err)
	}
	details := []string{
		"census evidence deterministic across two hermetic runs",
		fmt.Sprintf("generated output deterministic across two runs (%d files byte-identical)", len(corpusGenerated.Files)),
		"complete evidence (proofs, support, ownership, withholding, module imports) deterministic across two runs",
		"publication = immutable bundles + one atomic current-pointer rename; replacement leaves the previous bundle intact",
		fmt.Sprintf("partial bundle: %d packages withheld as honest unimplemented", len(corpusGenerated.Withheld)),
	}
	// Per-body analysis artifacts (spec 01): every lowered function and
	// method body — withheld packages included — retains its exact
	// fragment under analysis/bodies/ with a matching recorded hash.
	artifactCount, missing, hashMismatches := 0, 0, 0
	for i := range corpusGenerated.Proofs {
		proof := &corpusGenerated.Proofs[i]
		if proof.LoweredHash == "" {
			continue
		}
		artifactCount++
		path := "analysis/bodies/" + proof.Package + "/" + sanitizedTail(proof.ID) + ".ts.txt"
		found, has := corpusGenerated.Files[path]
		if !has {
			missing++
			continue
		}
		digest := sha256.Sum256([]byte(found))
		if hex.EncodeToString(digest[:]) != proof.LoweredHash {
			hashMismatches++
		}
	}
	details = append(details, fmt.Sprintf("per-body analysis artifacts: %d retained with recorded hashes", artifactCount))
	if missing > 0 || hashMismatches > 0 {
		return "fail", details, fmt.Errorf("body artifacts: %d missing, %d hash mismatches", missing, hashMismatches)
	}
	if artifactCount == 0 {
		return "fail", details, fmt.Errorf("no per-body analysis artifacts retained")
	}
	details = append(details, "relocated-checkout reproducibility proven; per-body artifacts retained")
	return "pass", details, nil
}

// sanitizedTail mirrors the translator's artifact file naming for one
// body identity.
func sanitizedTail(id string) string {
	out := make([]rune, 0, len(id))
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-':
			out = append(out, r)
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}
