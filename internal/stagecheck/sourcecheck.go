// Package stagecheck holds the blocking per-stage independent verifiers. Each
// pipeline stage's artifact is verified here before any downstream stage may
// consume it. Verifiers never reuse the producer's classifier, canonicalizer,
// resolver, or builder: they extract their observed side independently (the
// selected go binary directly, or the toolchain's own AST walker) and join by
// exact canonical identity, reporting both one-sided differences.
package stagecheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
// Standard and Goroot are corroborating facts; std/cmd set membership is the
// authoritative classification input.
type goListPackage struct {
	ImportPath string
	Dir        string
	Standard   bool
	Goroot     bool
	DepOnly    bool
	GoFiles    []string
	Imports    []string
	Module     *struct {
		Path      string
		Version   string
		Main      bool
		Dir       string
		GoVersion string
		Replace   *struct {
			Path    string
			Version string
			Dir     string
		}
	}
}

// universeExpectation is one independently derived package expectation.
type universeExpectation struct {
	provenance  source.Provenance
	acquisition source.Acquisition
	files       map[string]bool
}

// VerifySourceUniverse independently reconciles the loaded universe against
// the selected toolchain's own resolution: it executes `go list -deps -json`,
// `go list std`, and `go list cmd` with the identical binary, environment,
// overlays, flags, and roots, derives owner/provenance/acquisition
// independently from that output, and exact-joins the complete closure by
// canonical identity — reporting both one-sided difference lists.
func VerifySourceUniverse(ws *source.Workspace, req source.Request) error {
	fail := func(reason string) error {
		return &VerificationError{Stage: "source-universe", Reason: reason}
	}
	binary := ws.Toolchain().Binary()
	env := append(os.Environ(), req.Env...)
	env = append(env, "PATH="+filepath.Dir(binary)+string(os.PathListSeparator)+os.Getenv("PATH"))
	stdSet, err := listSet(binary, env, req.Dir, "std")
	if err != nil {
		return fail(err.Error())
	}
	cmdSet, err := listSet(binary, env, req.Dir, "cmd")
	if err != nil {
		return fail(err.Error())
	}
	patterns := req.Patterns
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	args := []string{"list", "-deps", "-json"}
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
	command := exec.Command(binary, args...)
	command.Dir = req.Dir
	command.Env = env
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		return fail("go list -deps failed: " + err.Error() + ": " + stderr.String())
	}
	// Decode all records, then independently compute the import-reachable
	// closure from the root packages (DepOnly=false) over source imports.
	// `go list -deps` additionally names the implicit runtime link closure of
	// main packages; the semantic universe is the import closure.
	records := map[string]*goListPackage{}
	var order []string
	decoder := json.NewDecoder(&stdout)
	for decoder.More() {
		var pkg goListPackage
		if err := decoder.Decode(&pkg); err != nil {
			return fail("go list output undecodable: " + err.Error())
		}
		record := pkg
		if _, dup := records[record.ImportPath]; dup {
			return fail("duplicate toolchain package " + record.ImportPath)
		}
		records[record.ImportPath] = &record
		order = append(order, record.ImportPath)
	}
	reachable := map[string]bool{}
	var visit func(path string)
	visit = func(path string) {
		if reachable[path] {
			return
		}
		reachable[path] = true
		if record, ok := records[path]; ok {
			for _, imported := range record.Imports {
				visit(imported)
			}
		}
	}
	for _, path := range order {
		if !records[path].DepOnly {
			visit(path)
		}
	}
	expected := map[string]*universeExpectation{}
	for _, path := range order {
		pkg := records[path]
		if !reachable[path] {
			continue
		}
		if len(pkg.GoFiles) == 0 && pkg.ImportPath != "unsafe" {
			continue
		}
		expectation, id, err := deriveExpectation(pkg, stdSet, cmdSet, ws.Toolchain().GOROOT(), req.Dir)
		if err != nil {
			return fail(err.Error())
		}
		if _, dup := expected[id]; dup {
			return fail("duplicate toolchain package identity " + id)
		}
		expected[id] = expectation
	}
	// Exact join, both directions, with one-sided identity lists.
	var orphans, mismatches []string
	matched := map[string]bool{}
	for _, pkg := range ws.Packages() {
		id := pkg.ID().String()
		expectation, wanted := expected[id]
		if !wanted {
			orphans = append(orphans, id)
			continue
		}
		matched[id] = true
		if pkg.Provenance() != expectation.provenance {
			mismatches = append(mismatches, fmt.Sprintf("%s provenance %s vs independent %s", id, pkg.Provenance(), expectation.provenance))
		}
		if pkg.Acquisition() != expectation.acquisition {
			mismatches = append(mismatches, fmt.Sprintf("%s acquisition %s vs independent %s", id, pkg.Acquisition(), expectation.acquisition))
		}
		for _, file := range pkg.Files() {
			if !expectation.files[file.ID().String()] {
				mismatches = append(mismatches, fmt.Sprintf("%s holds file %s the toolchain does not name", id, file.ID()))
			}
			delete(expectation.files, file.ID().String())
		}
		for missing := range expectation.files {
			mismatches = append(mismatches, fmt.Sprintf("%s misses toolchain file %s", id, missing))
		}
	}
	var dropped []string
	for id := range expected {
		if !matched[id] {
			dropped = append(dropped, id)
		}
	}
	if len(orphans)+len(dropped)+len(mismatches) > 0 {
		sort.Strings(orphans)
		sort.Strings(dropped)
		sort.Strings(mismatches)
		return fail(fmt.Sprintf("universe join failed; workspace-only=%v toolchain-only=%v mismatches=%v",
			orphans, dropped, mismatches))
	}
	return nil
}

// deriveExpectation independently classifies one go list record. std/cmd set
// membership is authoritative; Standard/Goroot corroborate; spelling never
// classifies.
func deriveExpectation(pkg *goListPackage, stdSet, cmdSet map[string]bool, goroot, workspaceDir string) (*universeExpectation, string, error) {
	out := &universeExpectation{files: map[string]bool{}}
	var owner identity.Owner
	var relBase string
	switch {
	case pkg.ImportPath == "builtin":
		owner = identity.LanguagePseudoOwner()
		out.provenance, out.acquisition = source.ProvenanceLanguagePseudo, source.AcquisitionGOROOT
		relBase = filepath.Join(goroot, "src")
	case stdSet[pkg.ImportPath]:
		if !pkg.Standard || !pkg.Goroot {
			return nil, "", fmt.Errorf("%s: std-set member without corroborating Standard/Goroot facts", pkg.ImportPath)
		}
		owner = identity.StandardLibraryOwner()
		out.provenance, out.acquisition = source.ProvenanceStandardLibrary, source.AcquisitionGOROOT
		relBase = filepath.Join(goroot, "src")
	case cmdSet[pkg.ImportPath] || (pkg.Goroot && pkg.Module == nil):
		owner = identity.ToolchainOwner()
		out.provenance, out.acquisition = source.ProvenanceToolchainPackage, source.AcquisitionGOROOT
		relBase = filepath.Join(goroot, "src")
	case pkg.Module != nil:
		moduleDir := pkg.Module.Dir
		if pkg.Module.Replace != nil && pkg.Module.Replace.Dir != "" {
			moduleDir = pkg.Module.Replace.Dir
		}
		version := pkg.Module.Version
		if pkg.Module.Main {
			version = ""
		}
		moduleID, err := identity.NewModuleID(pkg.Module.Path, version)
		if err != nil {
			return nil, "", err
		}
		owner, err = identity.NewModuleOwner(moduleID)
		if err != nil {
			return nil, "", err
		}
		switch {
		case pkg.Module.Main:
			out.provenance, out.acquisition = source.ProvenanceWorkspaceModule, source.AcquisitionWorkspace
		case pkg.Module.Replace != nil && pkg.Module.Replace.Version == "":
			out.provenance, out.acquisition = source.ProvenanceModuleDependency, source.AcquisitionLocalReplacement
		case moduleDir == "":
			out.provenance, out.acquisition = source.ProvenanceModuleDependency, source.AcquisitionVendor
		default:
			out.provenance = source.ProvenanceModuleDependency
			if under(pkg.Dir, workspaceDir) && strings.Contains(filepath.ToSlash(pkg.Dir), "/vendor/") {
				out.acquisition = source.AcquisitionVendor
			} else {
				out.acquisition = source.AcquisitionModuleCache
			}
		}
		relBase = moduleDir
		if relBase == "" {
			marker := filepath.FromSlash("vendor/" + pkg.Module.Path)
			if idx := strings.Index(pkg.Dir, marker); idx >= 0 {
				relBase = pkg.Dir[:idx+len(marker)]
			} else {
				relBase = workspaceDir
			}
		}
	default:
		return nil, "", fmt.Errorf("%s: toolchain names a package that is neither module-owned nor std/cmd", pkg.ImportPath)
	}
	packageID, err := identity.NewPackageID(owner, pkg.ImportPath)
	if err != nil {
		return nil, "", err
	}
	for _, goFile := range pkg.GoFiles {
		rel, err := filepath.Rel(relBase, filepath.Join(pkg.Dir, goFile))
		if err != nil || strings.HasPrefix(rel, "..") {
			return nil, "", fmt.Errorf("%s: file %s outside owner root", pkg.ImportPath, goFile)
		}
		fileID, err := identity.NewFileID(owner, filepath.ToSlash(rel))
		if err != nil {
			return nil, "", err
		}
		out.files[fileID.String()] = true
	}
	return out, packageID.String(), nil
}

func under(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && !strings.HasPrefix(rel, "..")
}

// listSet resolves one toolchain pattern set with the selected binary.
func listSet(binary string, env []string, dir, pattern string) (map[string]bool, error) {
	command := exec.Command(binary, "list", pattern)
	command.Dir = dir
	command.Env = env
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("go list %s: %v: %s", pattern, err, stderr.String())
	}
	set := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		if line != "" {
			set[line] = true
		}
	}
	return set, nil
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
