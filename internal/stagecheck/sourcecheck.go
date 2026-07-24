// Package stagecheck holds the blocking per-stage independent verifiers. Each
// pipeline stage's artifact is verified here before any downstream stage may
// consume it. Verifiers never reuse the producer's classifier, canonicalizer,
// resolver, or builder: they extract their observed side independently (the
// selected go binary directly, or the toolchain's own AST walker) and join by
// exact canonical identity, reporting both one-sided differences.
package stagecheck

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/build/constraint"
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
// Standard and Goroot are corroborating facts; std/cmd set membership is the
// authoritative classification input.
type goListPackage struct {
	ImportPath      string
	Dir             string
	Standard        bool
	Goroot          bool
	DepOnly         bool
	GoFiles         []string
	CgoFiles        []string
	CompiledGoFiles []string
	CFiles          []string
	CXXFiles        []string
	MFiles          []string
	HFiles          []string
	FFiles          []string
	SFiles          []string
	SwigFiles       []string
	SwigCXXFiles    []string
	SysoFiles       []string
	EmbedFiles      []string
	EmbedPatterns   []string
	Imports         []string
	Module          *struct {
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
	root          bool
	cgoSources    map[identity.FileID]bool
	checkedView   bool
	provenance    source.Provenance
	acquisition   source.Acquisition
	disposition   source.LanguageDisposition
	moduleGo      string
	imports       map[string]bool
	files         map[identity.FileID]bool
	filePaths     map[identity.FileID]string
	inputs        map[string]bool
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
	if err := verifyToolchainEvidence(ws, req); err != nil {
		return fail(err.Error())
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
	args := []string{"list", "-compiled", "-deps", "-json"}
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
		record, ok := records[path]
		if !ok {
			return
		}
		for _, imported := range record.Imports {
			if imported == "C" {
				// With -compiled, go list names the generated cgo package's
				// actual direct imports alongside the source-only marker C.
				continue
			}
			visit(imported)
		}
	}
	for _, path := range order {
		if !records[path].DepOnly {
			visit(path)
		}
	}
	expected := map[identity.PackageID]*universeExpectation{}
	for _, path := range order {
		pkg := records[path]
		if !reachable[path] {
			continue
		}
		if len(pkg.GoFiles) == 0 && len(pkg.CgoFiles) == 0 && pkg.ImportPath != "unsafe" {
			continue
		}
		expectation, id, err := deriveExpectation(
			pkg,
			stdSet,
			cmdSet,
			ws.Toolchain().GOROOT(),
			req.Dir,
			req.Overlay,
		)
		if err != nil {
			return fail(err.Error())
		}
		if _, dup := expected[id]; dup {
			return fail(
				fmt.Sprintf("duplicate toolchain package identity %s", id),
			)
		}
		expected[id] = expectation
	}
	builtinID, err := identity.NewPackageID(
		identity.LanguagePseudoOwner(), "builtin",
	)
	if err != nil {
		return fail(err.Error())
	}
	expected[builtinID] = &universeExpectation{
		provenance:    source.ProvenanceLanguagePseudo,
		acquisition:   source.AcquisitionGOROOT,
		disposition:   source.DispositionBuiltinUniverse,
		imports:       map[string]bool{},
		files:         map[identity.FileID]bool{},
		filePaths:     map[identity.FileID]string{},
		inputs:        map[string]bool{},
		embedPatterns: map[string]bool{},
		cgoSources:    map[identity.FileID]bool{},
	}
	// Exact join, both directions, with bounded one-sided identity evidence.
	problems := newProblemSet()
	matched := map[identity.PackageID]bool{}
	for _, pkg := range ws.Packages() {
		id := pkg.ID()
		expectation, wanted := expected[id]
		if !wanted {
			problems.addf("workspace-only package %s", id)
			continue
		}
		matched[id] = true
		if pkg.RequestedRoot() != expectation.root {
			problems.addf(
				"%s root=%v vs independent %v",
				id, pkg.RequestedRoot(), expectation.root,
			)
		}
		if pkg.Provenance() != expectation.provenance {
			problems.addf(
				"%s provenance %s vs independent %s",
				id, pkg.Provenance(), expectation.provenance,
			)
		}
		if pkg.Acquisition() != expectation.acquisition {
			problems.addf(
				"%s acquisition %s vs independent %s",
				id, pkg.Acquisition(), expectation.acquisition,
			)
		}
		if pkg.Disposition() != expectation.disposition {
			problems.addf(
				"%s disposition %s vs independent %s",
				id, pkg.Disposition(), expectation.disposition,
			)
		}
		if pkg.ModuleGoVersion() != expectation.moduleGo {
			problems.addf(
				"%s module go %q vs independent %q",
				id, pkg.ModuleGoVersion(), expectation.moduleGo,
			)
		}
		joinSet(
			id,
			"import",
			stringSet(pkg.Imports()),
			expectation.imports,
			problems,
		)
		joinFileIDSet(
			id, "file", fileIDSet(pkg), expectation.files, problems,
		)
		joinSet(
			id, "supplemental-input", inputSet(pkg),
			expectation.inputs, problems,
		)
		joinSet(
			id, "embed-pattern", stringSet(pkg.EmbedPatterns()),
			expectation.embedPatterns, problems,
		)
		verifyCheckedViewState(pkg, expectation, problems)
		verifyFileDigests(pkg, expectation, req.Overlay, problems)
		verifyFileVersions(
			pkg, expectation, req.Overlay,
			ws.Toolchain().GOROOT(), problems,
		)
	}
	for id := range expected {
		if !matched[id] {
			problems.addf("toolchain-only package %s", id)
		}
	}
	if !problems.empty() {
		return fail(problems.summary("universe exact join failed"))
	}
	if err := verifyResolutionFingerprint(ws, req); err != nil {
		return fail(err.Error())
	}
	return nil
}

func verifyFileDigests(
	pkg *source.Package,
	expectation *universeExpectation,
	overlay map[string][]byte,
	problems *problemSet,
) {
	for _, file := range pkg.Files() {
		path := expectation.filePaths[file.ID()]
		if path == "" {
			continue
		}
		raw, overlaid := overlay[path]
		if !overlaid {
			var err error
			raw, err = os.ReadFile(path)
			if err != nil {
				problems.addf(
					"%s independently unreadable: %v", file.ID(), err,
				)
				continue
			}
		}
		if file.ByteDigest() != source.SourceSpanHash(sha256.Sum256(raw)) {
			problems.addf(
				"%s selected-byte digest mismatch", file.ID(),
			)
		}
		if file.Overlaid() != overlaid {
			problems.addf(
				"%s overlay=%t vs independent %t",
				file.ID(), file.Overlaid(), overlaid,
			)
		}
	}
}

// verifyCheckedViewState checks every finalized cgo classification against the
// independently resolved toolchain file set. Checker hydration is transient
// Stage-1 evidence and deliberately does not survive in source.Workspace.
func verifyCheckedViewState(
	pkg *source.Package,
	expectation *universeExpectation,
	problems *problemSet,
) {
	id := pkg.ID()
	if pkg.HasCheckedView() != expectation.checkedView {
		problems.addf(
			"%s checked-view=%t vs independent %t",
			id, pkg.HasCheckedView(), expectation.checkedView,
		)
	}
	if expectation.disposition == source.DispositionOrdinarySource {
		actual := map[identity.FileID]bool{}
		for _, file := range pkg.Files() {
			if file.CgoOriginal() {
				actual[file.ID()] = true
			}
			// A cgo original claims a checked-view difference; the toolchain
			// must name it.
			if file.CgoOriginal() && !expectation.cgoSources[file.ID()] {
				problems.addf(
					"%s file %s claims a cgo view difference the toolchain does not name",
					id,
					file.ID(),
				)
			}
		}
		joinFileIDSet(
			id, "cgo-source", actual,
			expectation.cgoSources, problems,
		)
	}
}

// verifyFileVersions independently derives each file's effective language
// version from the module go directive plus the file's raw //go:build
// constraint (parsed with go/build/constraint, not the producer's evidence)
// and joins it against the producer's per-file version.
func verifyFileVersions(
	pkg *source.Package,
	expectation *universeExpectation,
	overlay map[string][]byte,
	goroot string,
	problems *problemSet,
) {
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
		path := expectation.filePaths[file.ID()]
		if path == "" {
			problems.addf(
				"%s has no independent acquisition path", file.ID(),
			)
			continue
		}
		if fromConstraint := fileConstraintVersion(path, overlay); fromConstraint != "" {
			expected = fromConstraint
		}
		if expected == "" {
			continue
		}
		if got := file.EffectiveGoVersion(); got != expected {
			problems.addf(
				"%s effective version %q vs independent %q",
				file.ID(), got, expected,
			)
		}
	}
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
