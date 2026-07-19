// The strict TypeScript staticness stage: generate the owned corpus,
// stage it with the pinned strict configuration, and let the pinned
// compiler judge it.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

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
	if details, failed := emitterDefectDetails(generated); failed {
		return "fail", details, fmt.Errorf("%d emitter defects (compiler must be fixed; diagnostic placeholders are never acceptable output)", len(generated.EmitterDefects))
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
	// Run the strict typecheck, but DECIDE only after the staticness
	// verifier has run (below): staticness verification is unconditional —
	// it must run regardless of the typecheck result — and a genuine tsc
	// error is a hard failure with NO waiver. There is no diagnostic-text
	// classifier; the emission set is made forward-dependency-closed by
	// the withholding fixed point, so a retained module never imports a
	// withheld one and strict tsc is REQUIRED to reach zero.
	tscOut, tscErr := runInRepo(staging, "node", tscJs, "-p", ".")
	// The staticness verdict comes from the typed-AST verifier (the
	// pinned TypeScript compiler parses every generated file and the AST
	// is walked structurally), so aliases, multiline forms, and renamed
	// equivalents cannot evade it; the text sweep remains as a fast
	// defense-in-depth pre-check. No file-local suppressions exist. It
	// runs whether or not the typecheck passed.
	typescriptModule := filepath.Join(repoDir, "product", "node_modules", "typescript")
	astReport, err := staticness.VerifyAST(generated.Files, typescriptModule)
	if err != nil {
		return "fail", nil, err
	}
	// BOTH results are reported together when EITHER fails — an early
	// return must never conceal the other result. EVERY detected staticness
	// regression fails (no abi: file-class downgrade), and zero strict-tsc
	// errors is REQUIRED (no waiver: the withholding fixed point makes the
	// emitted set forward-dependency-closed).
	astFail := len(astReport.Violations) > 0
	tscFail := tscErr != nil
	if astFail || tscFail {
		var details []string
		if astFail {
			counts := staticness.Counts(astReport.Violations)
			keys := make([]string, 0, len(counts))
			for key := range counts {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			details = append(details, fmt.Sprintf("AST staticness: %d violations", len(astReport.Violations)))
			for _, key := range keys {
				details = append(details, fmt.Sprintf("  %s: %d", key, counts[key]))
			}
			for i, v := range astReport.Violations {
				if i >= 6 {
					details = append(details, fmt.Sprintf("  ... and %d more sites", len(astReport.Violations)-6))
					break
				}
				details = append(details, fmt.Sprintf("  %s:%d %s", v.File, v.Line, v.Pattern))
			}
		} else {
			details = append(details, "AST staticness: 0 violations")
		}
		if tscFail {
			lines := splitLines(tscOut)
			details = append(details, fmt.Sprintf("strict tsc: %d diagnostics", len(lines)))
			for i, l := range lines {
				if i >= 6 {
					details = append(details, fmt.Sprintf("  ... and %d more", len(lines)-6))
					break
				}
				details = append(details, "  "+l)
			}
		} else {
			details = append(details, "strict tsc: 0 diagnostics")
		}
		return "fail", details, fmt.Errorf("gate 10: AST staticness=%d violations, strict tsc=%v", len(astReport.Violations), tscFail)
	}
	// (The former text check for "readonly v: unknown" is deleted: it was
	// dead beneath the structural AST verifier above — which now catches
	// the ABI's own erased GoBox<..., unknown, ...> as a hard failure —
	// and it did not match the generic alias form that actually erases.
	// There is one authoritative staticness path: the typed-AST verifier.)
	//
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
	// The positive per-invocation disposition ledger: every call,
	// construction, member selection, and index classified into a closed
	// class by the whole-program checker walk — an unclassifiable site is
	// a violation (caught above), so reaching here with a NON-EMPTY ledger
	// certifies every invocation positively.
	ledger := astReport.Ledger
	if ledger.Invocations() == 0 {
		return "fail", details, fmt.Errorf("positive-disposition ledger is empty: no invocation was certified")
	}
	details = append(details,
		fmt.Sprintf("positive-disposition ledger: %d invocations (direct-static %d, typed-function-value %d, constructor-static %d, unimplemented-blocking %d), %d member selections, %d typed index sites — zero undisposed",
			ledger.Invocations(), ledger.DirectStatic, ledger.TypedFunctionValue, ledger.ConstructorStatic, ledger.UnimplementedBlocking, ledger.MemberSelection, ledger.TypedIndex))
	return "pass", details, nil
}
