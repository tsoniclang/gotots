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
	"go/build/constraint"
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
	ImportPath    string
	Dir           string
	Standard      bool
	Goroot        bool
	DepOnly       bool
	GoFiles       []string
	CgoFiles      []string
	CFiles        []string
	CXXFiles      []string
	MFiles        []string
	HFiles        []string
	FFiles        []string
	SFiles        []string
	SwigFiles     []string
	SwigCXXFiles  []string
	SysoFiles     []string
	EmbedFiles    []string
	EmbedPatterns []string
	Imports       []string
	Module        *struct {
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

func (p *goListPackage) otherFiles() []string {
	var out []string
	for _, group := range [][]string{p.CFiles, p.CXXFiles, p.MFiles, p.HFiles, p.FFiles, p.SFiles, p.SwigFiles, p.SwigCXXFiles, p.SysoFiles} {
		out = append(out, group...)
	}
	return out
}

// universeExpectation is one independently derived package expectation.
type universeExpectation struct {
	root          bool
	cgoSources    map[string]bool
	provenance    source.Provenance
	acquisition   source.Acquisition
	disposition   source.LanguageDisposition
	moduleGo      string
	files         map[string]bool
	otherFiles    map[string]bool
	embedFiles    map[string]bool
	embedPatterns map[string]bool
	relBase       string
	moduleGoRaw   string
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
		if len(pkg.GoFiles) == 0 && len(pkg.CgoFiles) == 0 && pkg.ImportPath != "unsafe" {
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
		if pkg.RequestedRoot() != expectation.root {
			mismatches = append(mismatches, fmt.Sprintf("%s root=%v vs independent %v", id, pkg.RequestedRoot(), expectation.root))
		}
		if pkg.Provenance() != expectation.provenance {
			mismatches = append(mismatches, fmt.Sprintf("%s provenance %s vs independent %s", id, pkg.Provenance(), expectation.provenance))
		}
		if pkg.Acquisition() != expectation.acquisition {
			mismatches = append(mismatches, fmt.Sprintf("%s acquisition %s vs independent %s", id, pkg.Acquisition(), expectation.acquisition))
		}
		if pkg.Disposition() != expectation.disposition {
			mismatches = append(mismatches, fmt.Sprintf("%s disposition %s vs independent %s", id, pkg.Disposition(), expectation.disposition))
		}
		if pkg.ModuleGoVersion() != expectation.moduleGo {
			mismatches = append(mismatches, fmt.Sprintf("%s module go %q vs independent %q", id, pkg.ModuleGoVersion(), expectation.moduleGo))
		}
		mismatches = append(mismatches, joinSet(id, "file", fileIDSet(pkg), expectation.files)...)
		mismatches = append(mismatches, joinSet(id, "other-file", stringSet(pkg.OtherFiles()), expectation.otherFiles)...)
		mismatches = append(mismatches, joinSet(id, "embed-file", stringSet(pkg.EmbedFiles()), expectation.embedFiles)...)
		mismatches = append(mismatches, joinSet(id, "embed-pattern", stringSet(pkg.EmbedPatterns()), expectation.embedPatterns)...)
		mismatches = append(mismatches, verifyEvidenceState(pkg, expectation)...)
		mismatches = append(mismatches, verifyFileVersions(pkg, expectation, req.Overlay, ws.Toolchain().GOROOT())...)
	}
	mismatches = append(mismatches, verifyTypeGraphCoherence(ws)...)
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

func stringSet(values []string) map[string]bool {
	set := map[string]bool{}
	for _, v := range values {
		set[v] = true
	}
	return set
}

func fileIDSet(pkg *source.Package) map[string]bool {
	set := map[string]bool{}
	for _, file := range pkg.Files() {
		set[file.ID().String()] = true
	}
	return set
}

// joinSet exact-joins one input-file class, reporting both one-sided lists.
func joinSet(id, class string, got, want map[string]bool) []string {
	var out []string
	for member := range got {
		if !want[member] {
			out = append(out, fmt.Sprintf("%s holds %s %s the toolchain does not name", id, class, member))
		}
	}
	for member := range want {
		if !got[member] {
			out = append(out, fmt.Sprintf("%s misses toolchain %s %s", id, class, member))
		}
	}
	return out
}

// verifyEvidenceState checks the source/type evidence a package's disposition
// requires is present — for every package, not only roots.
func verifyEvidenceState(pkg *source.Package, expectation *universeExpectation) []string {
	id := pkg.ID().String()
	switch expectation.disposition {
	case source.DispositionBuiltinUniverse:
		return nil
	case source.DispositionUnsafeIntrinsic:
		if pkg.Types() == nil {
			return []string{id + " unsafe record lacks type evidence"}
		}
		return nil
	}
	var out []string
	if pkg.Types() == nil {
		out = append(out, id+" lacks type evidence")
	}
	// Body-indexed type information follows the evidence-depth partition:
	// present exactly when the package retains full-semantic units.
	if pkg.RetainsFullSemantic() && pkg.TypesInfo() == nil {
		out = append(out, id+" retains full-semantic units without type information")
	}
	if !pkg.RetainsFullSemantic() && pkg.TypesInfo() != nil {
		out = append(out, id+" retains type information without full-semantic units")
	}
	if expectation.disposition == source.DispositionOrdinarySource {
		for _, file := range pkg.Files() {
			if _, hasFull := file.FullSyntax(); hasFull {
				continue
			}
			if _, mixed := file.Evidence().(source.MixedUnits); mixed {
				continue
			}
			// ContractOnly is valid for non-full depths and cgo originals;
			// a full-semantic unit inside a ContractOnly file is caught by
			// admission. Here we require only that cgo originals are
			// toolchain-named.
			if file.CgoOriginal() && !expectation.cgoSources[file.ID().String()] {
				out = append(out, id+" file "+file.ID().String()+" claims a cgo view difference the toolchain does not name")
			}
		}
	}
	return out
}

// verifyFileVersions independently derives each file's effective language
// version from the module go directive plus the file's raw //go:build
// constraint (parsed with go/build/constraint, not the producer's evidence)
// and joins it against the producer's per-file version.
func verifyFileVersions(pkg *source.Package, expectation *universeExpectation, overlay map[string][]byte, goroot string) []string {
	var out []string
	base := ""
	if expectation.moduleGo != "" {
		base = "go" + expectation.moduleGo
	}
	if base == "" {
		switch pkg.ID().Owner().Class() {
		case identity.OwnerStandardLibrary, identity.OwnerToolchain:
			// Reserved owners' base version is the GOROOT source go.mod
			// directive — read independently from raw bytes.
			base = gorootGoDirective(goroot)
		}
	}
	for _, file := range pkg.Files() {
		if file.EffectiveGoVersion() == "" {
			// cgo originals and intrinsic files carry no checked version.
			continue
		}
		expected := base
		if fromConstraint := fileConstraintVersion(file.Path(), overlay); fromConstraint != "" {
			expected = fromConstraint
		}
		if expected == "" {
			continue
		}
		if got := file.EffectiveGoVersion(); got != expected {
			out = append(out, fmt.Sprintf("%s effective version %q vs independent %q", file.ID(), got, expected))
		}
	}
	return out
}

// gorootGoDirective reads the go directive of GOROOT/src/go.mod from raw
// bytes, independent of the producer.
func gorootGoDirective(goroot string) string {
	raw, err := os.ReadFile(filepath.Join(goroot, "src", "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if version, found := strings.CutPrefix(trimmed, "go "); found {
			return "go" + strings.TrimSpace(version)
		}
	}
	return ""
}

// fileConstraintVersion extracts the go-version bound of a file's //go:build
// line from raw bytes, independent of the producer's parse.
func fileConstraintVersion(path string, overlay map[string][]byte) string {
	raw, overlaid := overlay[path]
	if !overlaid {
		var err error
		raw, err = os.ReadFile(path)
		if err != nil {
			return ""
		}
	}
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "package ") {
			break
		}
		if !constraint.IsGoBuild(trimmed) {
			continue
		}
		expr, err := constraint.Parse(trimmed)
		if err != nil {
			return ""
		}
		return constraint.GoVersion(expr)
	}
	return ""
}

// verifyTypeGraphCoherence proves every import edge resolves to the identical
// *types.Package object stored on the imported record: one coherent go/types
// graph, never mixed loads.
func verifyTypeGraphCoherence(ws *source.Workspace) []string {
	byPath := map[string]*source.Package{}
	for _, pkg := range ws.Packages() {
		byPath[pkg.ID().ImportPath()] = pkg
	}
	var out []string
	for _, pkg := range ws.Packages() {
		if pkg.Types() == nil {
			continue
		}
		for _, imported := range pkg.Types().Imports() {
			record, tracked := byPath[imported.Path()]
			if !tracked || record.Types() == nil {
				continue
			}
			if record.Types() != imported {
				out = append(out, fmt.Sprintf("type-graph incoherence: %s imports %s as a distinct types.Package object",
					pkg.ID(), imported.Path()))
			}
		}
	}
	return out
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
