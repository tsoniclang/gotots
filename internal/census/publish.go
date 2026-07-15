package census

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
)

// Generator records the provenance of the gotots binary that produced a
// report bundle, including the module graph it was built from.
type Generator struct {
	GoVersion   string            `json:"goVersion"`
	Module      string            `json:"module,omitempty"`
	Version     string            `json:"version,omitempty"`
	VCSRevision string            `json:"vcsRevision,omitempty"`
	VCSModified bool              `json:"vcsModified,omitempty"`
	BuildDeps   map[string]string `json:"buildDeps,omitempty"` // module -> version
	// ProvenanceIssues lists why this generator cannot produce an
	// authoritative bundle (dirty build, missing VCS stamp, ...).
	ProvenanceIssues []string `json:"provenanceIssues,omitempty"`
}

// Manifest links every report in a published bundle with its hash and the
// generator provenance. A bundle is valid only as a whole.
type Manifest struct {
	SchemaVersion int       `json:"schemaVersion"`
	Generator     Generator `json:"generator"`
	// Authoritative is true only when generator provenance is clean and the
	// census recorded no blockers (unknown directives, contradictions).
	Authoritative bool              `json:"authoritative"`
	Blockers      []string          `json:"blockers,omitempty"`
	Files         map[string]string `json:"files"` // name -> sha256
}

func generatorProvenance() Generator {
	generator := Generator{GoVersion: runtime.Version()}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		generator.ProvenanceIssues = append(generator.ProvenanceIssues, "build info unavailable")
		return generator
	}
	generator.Module = info.Main.Path
	generator.Version = info.Main.Version
	generator.BuildDeps = map[string]string{}
	for _, dep := range info.Deps {
		effective := dep
		if dep.Replace != nil {
			effective = dep.Replace
		}
		generator.BuildDeps[dep.Path] = effective.Version
	}
	vcsStamped := false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			generator.VCSRevision = setting.Value
			vcsStamped = true
		case "vcs.modified":
			generator.VCSModified = setting.Value == "true"
		}
	}
	if !vcsStamped {
		generator.ProvenanceIssues = append(generator.ProvenanceIssues, "no VCS revision stamped into the binary")
	}
	if generator.VCSModified {
		generator.ProvenanceIssues = append(generator.ProvenanceIssues, "binary built from a dirty worktree")
	}
	return generator
}

// WriteReports publishes the bundle transactionally and nondestructively:
//
//   - the destination must not already exist; prior evidence is never
//     removed or overwritten — bundles are immutable;
//   - reports are staged in a unique same-filesystem directory, hashed into
//     a manifest, fsynced, and atomically renamed into place;
//   - the destination and the pinned source tree must not overlap in either
//     direction after symlink resolution.
//
// A failure never leaves a partial destination.
func WriteReports(result *Result, outDir string) error {
	absOut, err := filepath.Abs(outDir)
	if err != nil {
		return err
	}
	if err := refuseOverlap(absOut, result.sourceDir); err != nil {
		return err
	}
	if _, err := os.Lstat(absOut); err == nil {
		return fmt.Errorf("output bundle %s already exists; bundles are immutable, choose a fresh path", absOut)
	} else if !os.IsNotExist(err) {
		return err
	}
	parent := filepath.Dir(absOut)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}

	staging, err := os.MkdirTemp(parent, ".gotots-staging-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging) // no-op after successful rename

	manifest := Manifest{
		SchemaVersion: 1,
		Generator:     generatorProvenance(),
		Blockers:      result.Report.Blockers,
	}
	manifest.Authoritative = len(manifest.Generator.ProvenanceIssues) == 0 && len(manifest.Blockers) == 0
	manifest.Files = map[string]string{}

	write := func(name string, value any) error {
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return err
		}
		data = append(data, '\n')
		digest := sha256.Sum256(data)
		manifest.Files[name] = hex.EncodeToString(digest[:])
		return syncWrite(filepath.Join(staging, name), data)
	}
	if err := write("inventory.json", result.Inventory); err != nil {
		return err
	}
	if err := write("census.json", result.Report); err != nil {
		return err
	}
	if err := write("declarations.json", result.Shapes); err != nil {
		return err
	}
	if err := write("externals.json", result.Externals); err != nil {
		return err
	}
	if err := write("environment.json", result.Environment); err != nil {
		return err
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := syncWrite(filepath.Join(staging, "manifest.json"), append(manifestData, '\n')); err != nil {
		return err
	}
	if err := syncDir(staging); err != nil {
		return err
	}

	// Atomic publication to a previously absent path. If a concurrent
	// writer won the race, rename fails and this bundle's staging is
	// discarded; the destination is never a mixture.
	if err := os.Rename(staging, absOut); err != nil {
		return fmt.Errorf("publish bundle: %w", err)
	}
	return syncDir(parent)
}

// VerifyBundle proves a published bundle is complete and untampered: every
// manifest entry exists with a matching hash and no unexpected report files
// are present.
func VerifyBundle(dir string) (*Manifest, error) {
	manifestData, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("bundle manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("bundle manifest: %w", err)
	}
	for name, expected := range manifest.Files {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("bundle file %s: %w", name, err)
		}
		digest := sha256.Sum256(data)
		if actual := hex.EncodeToString(digest[:]); actual != expected {
			return nil, fmt.Errorf("bundle file %s hash %s does not match manifest %s", name, actual, expected)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		name := entry.Name()
		if name == "manifest.json" {
			continue
		}
		if _, ok := manifest.Files[name]; !ok {
			return nil, fmt.Errorf("bundle contains %s which is not in the manifest", name)
		}
	}
	return &manifest, nil
}

// refuseOverlap rejects any overlap between the output path and the pinned
// source tree in either direction, after resolving symlinks: writing inside
// the source would pollute verified input, and a source nested inside the
// output could be destroyed by output management.
func refuseOverlap(outDir, sourceDir string) error {
	if sourceDir == "" {
		return fmt.Errorf("internal error: result has no source directory")
	}
	resolvedSource, err := filepath.EvalSymlinks(sourceDir)
	if err != nil {
		return err
	}
	// Resolve the deepest existing ancestor of the output path so a
	// symlink cannot smuggle the bundle across the boundary.
	probe := outDir
	for {
		if _, err := os.Lstat(probe); err == nil {
			break
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			break
		}
		probe = parent
	}
	resolvedProbe, err := filepath.EvalSymlinks(probe)
	if err != nil {
		return err
	}
	resolvedOut := filepath.Join(resolvedProbe, strings.TrimPrefix(outDir, probe))

	separator := string(filepath.Separator)
	switch {
	case resolvedOut == resolvedSource:
		return fmt.Errorf("output %s is the pinned source tree; refusing", outDir)
	case strings.HasPrefix(resolvedOut, resolvedSource+separator):
		return fmt.Errorf("output %s is inside the pinned source tree %s; refusing to write into verified input", outDir, sourceDir)
	case strings.HasPrefix(resolvedSource, resolvedOut+separator):
		return fmt.Errorf("output %s is an ancestor of the pinned source tree %s; refusing to manage a tree that contains verified input", outDir, sourceDir)
	}
	return nil
}

func syncWrite(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
