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
// report bundle.
type Generator struct {
	GoVersion   string `json:"goVersion"`
	Module      string `json:"module,omitempty"`
	Version     string `json:"version,omitempty"`
	VCSRevision string `json:"vcsRevision,omitempty"`
	VCSModified bool   `json:"vcsModified,omitempty"`
}

// Manifest links every report in a published bundle with its hash and the
// generator provenance. A bundle is valid only as a whole.
type Manifest struct {
	SchemaVersion int               `json:"schemaVersion"`
	Generator     Generator         `json:"generator"`
	Files         map[string]string `json:"files"` // name -> sha256
}

func generatorProvenance() Generator {
	generator := Generator{GoVersion: runtime.Version()}
	if info, ok := debug.ReadBuildInfo(); ok {
		generator.Module = info.Main.Path
		generator.Version = info.Main.Version
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				generator.VCSRevision = setting.Value
			case "vcs.modified":
				generator.VCSModified = setting.Value == "true"
			}
		}
	}
	return generator
}

// WriteReports publishes the bundle transactionally: reports are staged
// into a sibling directory with a manifest, fsynced, and atomically renamed
// into place. A failure never leaves a mixture of old and new reports, and
// the output path may not lie inside the pinned source tree.
func WriteReports(result *Result, outDir string) error {
	absOut, err := filepath.Abs(outDir)
	if err != nil {
		return err
	}
	if err := refuseInsideSource(absOut, result.sourceDir); err != nil {
		return err
	}

	staging := absOut + ".staging"
	if err := os.RemoveAll(staging); err != nil {
		return err
	}
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(staging) // no-op after successful rename

	manifest := Manifest{
		SchemaVersion: 1,
		Generator:     generatorProvenance(),
		Files:         map[string]string{},
	}
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

	// Publish atomically: remove any previous bundle, then rename. The
	// bundle is either the complete old one, absent, or the complete new
	// one — never a mixture.
	if err := os.RemoveAll(absOut); err != nil {
		return err
	}
	if err := os.Rename(staging, absOut); err != nil {
		return err
	}
	return syncDir(filepath.Dir(absOut))
}

func refuseInsideSource(outDir, sourceDir string) error {
	if sourceDir == "" {
		return fmt.Errorf("internal error: result has no source directory")
	}
	resolvedSource, err := filepath.EvalSymlinks(sourceDir)
	if err != nil {
		return err
	}
	// Resolve the deepest existing ancestor of the output path so a
	// symlink cannot smuggle the bundle into the source tree.
	probe := outDir
	for {
		if _, err := os.Stat(probe); err == nil {
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
	suffix := strings.TrimPrefix(outDir, probe)
	resolvedOut := filepath.Join(resolvedProbe, suffix)
	if resolvedOut == resolvedSource || strings.HasPrefix(resolvedOut, resolvedSource+string(filepath.Separator)) {
		return fmt.Errorf("output directory %s is inside the pinned source tree %s; refusing to write into verified input", outDir, sourceDir)
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
