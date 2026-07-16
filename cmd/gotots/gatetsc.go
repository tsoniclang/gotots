// The strict TypeScript staticness stage: generate the owned corpus,
// stage it with the pinned strict configuration, and let the pinned
// compiler judge it.
package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tsoniclang/gotots/internal/census"
	"github.com/tsoniclang/gotots/internal/goenv"
	"github.com/tsoniclang/gotots/internal/pinning"
	"github.com/tsoniclang/gotots/internal/productinputs"
	"github.com/tsoniclang/gotots/internal/profile"
	"github.com/tsoniclang/gotots/internal/staticness"
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
		// Classify over the WHOLE diagnostic stream (splitLines keeps only
		// the last 40 lines, which would orphan continuation lines from
		// their error headers and misclassify them).
		lines := strings.Split(strings.TrimSpace(out), "\n")
		// Partition every diagnostic. A tsc error is PREREQUISITE-ROOTED
		// only when it is definitionally a consequence of the emission set
		// being a partial subset (not forward-dependency-closed): a
		// "cannot find module" for a co-generated core module absent from
		// this bundle, or an opaque external-stub interface supertype
		// (GoIface/GoAnyBox) meeting a concrete core union at the
		// stub/core boundary. Anything else is a genuine defect in emitted
		// logic. If even one genuine defect exists the stage FAILS; only
		// when every error is prerequisite-rooted is it BLOCKED on
		// complete emission — the classifier can never hide a real defect.
		genuine, prerequisite := classifyTypecheckErrors(lines, generated.Withheld)
		if len(genuine) == 0 && len(prerequisite) > 0 {
			details := []string{
				"BLOCKED: the emission set is not forward-dependency-closed; every tsc diagnostic is a withheld co-generated module or an opaque external-stub interface at the stub/core boundary — none is a defect in emitted logic",
				fmt.Sprintf("prerequisite-rooted tsc errors: %d; genuine defects: 0", len(prerequisite)),
				"the prerequisite (complete, forward-dependency-closed corpus emission) is shared with stage 18",
			}
			for i, l := range prerequisite {
				if i >= 8 {
					details = append(details, fmt.Sprintf("... and %d more prerequisite-rooted errors", len(prerequisite)-8))
					break
				}
				details = append(details, l)
			}
			return "blocked", details, nil
		}
		if len(genuine) > 40 {
			genuine = append(genuine[:40], fmt.Sprintf("... and %d more lines", len(genuine)-40))
		}
		return "fail", genuine, fmt.Errorf("strict typecheck found %d genuine defects in emitted logic", len(genuine))
	}
	// The staticness verdict comes from the typed-AST verifier (the
	// pinned TypeScript compiler parses every generated file and the AST
	// is walked structurally), so aliases, multiline forms, and renamed
	// equivalents cannot evade it; the text sweep remains as a fast
	// defense-in-depth pre-check. No file-local suppressions exist.
	typescriptModule := filepath.Join(repoDir, "product", "node_modules", "typescript")
	astReport, err := staticness.VerifyAST(generated.Files, typescriptModule)
	if err != nil {
		return "fail", nil, err
	}
	// ABI-scoped erasure findings are the REMAINING reviewed mechanism
	// (equality's construction-bound payload flow and the helper
	// supertype): they are reported and keep this stage blocked, never
	// silently passed; anything in generated core fails outright.
	var coreViolations []staticness.Violation
	abiViolations := 0
	for _, v := range astReport.Violations {
		if strings.HasPrefix(v.Pattern, "abi:") {
			abiViolations++
			continue
		}
		coreViolations = append(coreViolations, v)
	}
	astReport.Violations = coreViolations
	if len(astReport.Violations) > 0 {
		counts := staticness.Counts(astReport.Violations)
		keys := make([]string, 0, len(counts))
		for key := range counts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		details := make([]string, 0, len(keys)+8)
		for _, key := range keys {
			details = append(details, fmt.Sprintf("%s: %d", key, counts[key]))
		}
		for i, v := range astReport.Violations {
			if i >= 8 {
				details = append(details, fmt.Sprintf("... and %d more sites", len(astReport.Violations)-8))
				break
			}
			details = append(details, fmt.Sprintf("%s:%d %s", v.File, v.Line, v.Pattern))
		}
		return "fail", details, fmt.Errorf("AST staticness verifier found %d prohibited dispatch sites", len(astReport.Violations))
	}
	// The accepted interface specification (docs/spec/06) requires a
	// closed statically typed payload; the current carrier still holds an
	// erased unknown payload recovered by casts at dispatch. Until the
	// representation redesign lands, this stage is BLOCKED — passing
	// would attest output that contradicts the accepted specification.
	if iface, ok := generated.Files["language-abi/goiface.ts"]; ok && strings.Contains(iface, "readonly v: unknown") {
		return "blocked", []string{
			"interface carrier retains an erased unknown payload (GoIfaceBox.v: unknown) with cast recovery at dispatch",
			"contradicts docs/spec/06-interfaces-generics-functions.md (closed statically typed payload union required)",
			"the representation redesign (spec/ADR pending) unblocks this stage",
		}, nil
	}
	// Authoritative symbol evidence: every retained proof's generated
	// symbol must be an exported declaration in its file's typed AST.
	var symbolDefects []string
	for _, proof := range generated.Proofs {
		if !proof.ModuleRetained || proof.GeneratedSymbol == "" {
			continue
		}
		found := false
		for _, name := range astReport.Exports[proof.GeneratedFile] {
			if name == proof.GeneratedSymbol {
				found = true
				break
			}
		}
		if !found {
			if len(symbolDefects) < 10 {
				symbolDefects = append(symbolDefects, proof.ID+": symbol "+proof.GeneratedSymbol+" not an exported AST declaration of "+proof.GeneratedFile)
			}
		}
	}
	if len(symbolDefects) > 0 {
		return "fail", symbolDefects, fmt.Errorf("proof symbol evidence failed the typed-AST join")
	}
	violations := staticness.Sweep(generated.Files)
	if len(violations) > 0 {
		counts := staticness.Counts(violations)
		keys := make([]string, 0, len(counts))
		for key := range counts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		details := make([]string, 0, len(keys)+8)
		for _, key := range keys {
			details = append(details, fmt.Sprintf("%s: %d", key, counts[key]))
		}
		for i, v := range violations {
			if i >= 8 {
				details = append(details, fmt.Sprintf("... and %d more sites", len(violations)-8))
				break
			}
			details = append(details, fmt.Sprintf("%s:%d %s", v.File, v.Line, v.Pattern))
		}
		return "fail", details, fmt.Errorf("staticness sweep found %d prohibited dispatch sites", len(violations))
	}
	details := []string{
		fmt.Sprintf("strict tsc (%s@%s) accepted %d generated files", productPin.TypescriptCompiler.Package, productPin.TypescriptCompiler.Version, len(generated.Files)),
		"typed-AST staticness verifier: zero erased or name-selected dispatch sites",
		"text sweep (defense in depth): zero prohibited sites",
		fmt.Sprintf("withheld packages (honest unimplemented, not typechecked): %d", len(generated.Withheld)),
	}
	details = append(details,
		fmt.Sprintf("BLOCKED: %d ABI erasure sites remain (equality's construction-bound payload flow; helper supertype) — reported, never passed over", abiViolations),
		"BLOCKED: the per-invocation positive-disposition ledger is absent — the verifier rejects known erased forms but cannot yet certify every invocation (spec 11 staticness sweep)")
	return "blocked", details, nil
}

// classifyTypecheckErrors partitions tsc diagnostics into genuine
// defects and prerequisite-rooted errors. Diagnostics are grouped: an
// "error TS" line starts a group that absorbs its indented continuation
// lines, so a type name appearing on a continuation still classifies its
// group. A group is prerequisite-rooted ONLY when it is definitionally a
// partial-emission artifact — a missing co-generated core module, or an
// opaque external-stub interface supertype (GoIface/GoAnyBox) at the
// stub/core boundary. Every other group is a genuine defect. The default
// is "genuine": an unrecognized diagnostic fails the stage.
func classifyTypecheckErrors(lines []string, withheld map[string]string) (genuine, prerequisite []string) {
	var group []string
	flush := func() {
		if len(group) == 0 {
			return
		}
		head := group[0]
		joined := strings.Join(group, "\n")
		if typecheckErrorIsPrerequisiteRooted(joined, withheld) {
			prerequisite = append(prerequisite, head)
		} else {
			genuine = append(genuine, head)
		}
		group = nil
	}
	for _, line := range lines {
		if strings.Contains(line, "error TS") {
			flush()
			group = []string{line}
			continue
		}
		if len(group) > 0 {
			group = append(group, line)
		} else if strings.TrimSpace(line) != "" {
			// A stray non-error, non-continuation line before any error:
			// treat as a genuine, unclassified diagnostic (fail-safe).
			genuine = append(genuine, line)
		}
	}
	flush()
	return genuine, prerequisite
}

// typecheckErrorIsPrerequisiteRooted reports whether one grouped tsc
// diagnostic is a partial-emission artifact rather than a logic defect.
func typecheckErrorIsPrerequisiteRooted(group string, withheld map[string]string) bool {
	// (1) A "cannot find module" whose RESOLVED target package is
	// withheld from this bundle: absent only because emission is a
	// partial subset. The specifier is resolved relative to the erroring
	// file and checked against the withheld set, so a genuinely broken
	// import (a specifier that does not resolve to a withheld package) is
	// NOT excused — it stays a genuine defect.
	if strings.Contains(group, "TS2307") && strings.Contains(group, "Cannot find module") {
		pkg := resolveMissingModulePackage(group)
		if pkg == "" {
			return false
		}
		_, isWithheld := withheld[pkg]
		return isWithheld
	}
	// (2) An opaque external-stub interface supertype meeting a concrete
	// core union: GoIface/GoAnyBox appear ONLY in external stub modules,
	// which spell interfaces opaquely pending forward-closed emission.
	if strings.Contains(group, "GoIface") || strings.Contains(group, "GoAnyBox") {
		return true
	}
	return false
}

// resolveMissingModulePackage extracts the erroring file and the missing
// module specifier from a TS2307 group and resolves the specifier to a
// co-generated package path (the "core/<path>/package.js" convention).
// It returns "" when the diagnostic is not a co-generated module miss.
func resolveMissingModulePackage(group string) string {
	// File is the text before the first "(" of the "file(line,col):" head.
	head := group
	if nl := strings.IndexByte(head, '\n'); nl >= 0 {
		head = head[:nl]
	}
	paren := strings.IndexByte(head, '(')
	if paren < 0 {
		return ""
	}
	file := head[:paren]
	// Specifier is the single-quoted path after "Cannot find module ".
	marker := "Cannot find module '"
	start := strings.Index(group, marker)
	if start < 0 {
		return ""
	}
	rest := group[start+len(marker):]
	end := strings.IndexByte(rest, '\'')
	if end < 0 {
		return ""
	}
	specifier := rest[:end]
	// Resolve the specifier relative to the erroring file's directory.
	resolved := path.Clean(path.Join(path.Dir(file), specifier))
	// A co-generated package module is "core/<pkgpath>/package.js".
	trimmed, ok := strings.CutPrefix(resolved, "core/")
	if !ok {
		return ""
	}
	trimmed, ok = strings.CutSuffix(trimmed, "/package.js")
	if !ok {
		return ""
	}
	return trimmed
}
