// Package stagecheck holds the blocking per-stage independent verifiers. Each
// pipeline stage's artifact is verified here before any downstream stage may
// consume it. Verifiers never reuse the producer's classifier, canonicalizer,
// resolver, or builder: they extract their observed side independently (the
// go toolchain directly, or the toolchain's own AST walker) and join by exact
// canonical identity.
package stagecheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tsoniclang/gotots/internal/identity"
	"github.com/tsoniclang/gotots/internal/source"
)

// VerificationError is the typed failure of one stage verifier. A failed
// verification blocks every downstream stage.
type VerificationError struct {
	Stage  string
	Reason string
}

func (e *VerificationError) Error() string {
	return fmt.Sprintf("GOTOTS_STAGE_VERIFICATION: %s: %s", e.Stage, e.Reason)
}

// goListPackage is the subset of `go list -json` output the verifier joins.
type goListPackage struct {
	ImportPath string
	Dir        string
	GoFiles    []string
	Module     *struct {
		Path    string
		Version string
		Main    bool
		Dir     string
		Replace *struct {
			Path string
			Dir  string
		}
	}
}

// VerifySourceUniverse independently reconciles a loaded workspace against the
// Go toolchain's own module/build selection: it runs `go list -json` directly
// (not through the loader) and exact-joins the resulting file identity
// multiset against the workspace artifact. Dropped inputs, orphan outputs,
// and duplicate identities all fail.
func VerifySourceUniverse(ws *source.Workspace, req source.Request) error {
	fail := func(reason string) error {
		return &VerificationError{Stage: "source-universe", Reason: reason}
	}
	patterns := req.Patterns
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	args := []string{"list", "-json", "-deps=false"}
	if len(req.Overlay) > 0 {
		overlayFile, cleanup, err := materializeOverlay(req.Overlay)
		if err != nil {
			return fail("overlay materialization failed: " + err.Error())
		}
		defer cleanup()
		args = append(args, "-overlay", overlayFile)
	}
	args = append(args, req.BuildFlags...)
	args = append(args, patterns...)
	cmd := exec.Command("go", args...)
	cmd.Dir = req.Dir
	cmd.Env = append(os.Environ(), req.Env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return fail("go list failed: " + err.Error() + ": " + stderr.String())
	}
	expected := map[string]bool{} // file identity -> seen in workspace
	decoder := json.NewDecoder(&stdout)
	for decoder.More() {
		var pkg goListPackage
		if err := decoder.Decode(&pkg); err != nil {
			return fail("go list output undecodable: " + err.Error())
		}
		if len(pkg.GoFiles) == 0 {
			continue
		}
		if pkg.Module == nil {
			return fail(pkg.ImportPath + ": toolchain reports no module")
		}
		modulePath, moduleDir, version := pkg.Module.Path, pkg.Module.Dir, pkg.Module.Version
		if pkg.Module.Replace != nil {
			moduleDir = pkg.Module.Replace.Dir
		}
		if pkg.Module.Main {
			version = ""
		}
		moduleID, err := identity.NewModuleID(modulePath, version)
		if err != nil {
			return fail("toolchain module identity invalid: " + err.Error())
		}
		for _, goFile := range pkg.GoFiles {
			rel, err := filepath.Rel(moduleDir, filepath.Join(pkg.Dir, goFile))
			if err != nil || strings.HasPrefix(rel, "..") {
				return fail(goFile + ": toolchain file outside module root")
			}
			fileID, err := identity.NewFileID(moduleID, filepath.ToSlash(rel))
			if err != nil {
				return fail("toolchain file identity invalid: " + err.Error())
			}
			if _, dup := expected[fileID.String()]; dup {
				return fail("duplicate toolchain file identity " + fileID.String())
			}
			expected[fileID.String()] = false
		}
	}
	for _, pkg := range ws.Packages() {
		for _, file := range pkg.Files() {
			id := file.ID().String()
			seen, wanted := expected[id]
			if !wanted {
				return fail("orphan output: workspace holds " + id + " which the toolchain selection does not name")
			}
			if seen {
				return fail("duplicate workspace file identity " + id)
			}
			expected[id] = true
		}
	}
	for id, seen := range expected {
		if !seen {
			return fail("dropped input: toolchain selection names " + id + " which the workspace does not hold")
		}
	}
	return nil
}

// materializeOverlay writes overlay contents to disk and produces the go tool
// overlay JSON file referencing them.
func materializeOverlay(overlay map[string][]byte) (string, func(), error) {
	dir, err := os.MkdirTemp("", "gotots-overlay-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.RemoveAll(dir) }
	replace := map[string]string{}
	i := 0
	for path, content := range overlay {
		materialized := filepath.Join(dir, fmt.Sprintf("overlay-%d%s", i, filepath.Ext(path)))
		if err := os.WriteFile(materialized, content, 0o644); err != nil {
			cleanup()
			return "", nil, err
		}
		replace[path] = materialized
		i++
	}
	encoded, err := json.Marshal(map[string]any{"Replace": replace})
	if err != nil {
		cleanup()
		return "", nil, err
	}
	overlayFile := filepath.Join(dir, "overlay.json")
	if err := os.WriteFile(overlayFile, encoded, 0o644); err != nil {
		cleanup()
		return "", nil, err
	}
	return overlayFile, cleanup, nil
}
