package staticness

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func typescriptDir(t *testing.T) string {
	_, thisFile, _, _ := runtime.Caller(0)
	repo := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	dir := filepath.Join(repo, "product", "node_modules", "typescript")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("pinned typescript module: %v", err)
	}
	return dir
}

// TestASTVerifierCatchesStructuralForms proves the typed-AST walk
// rejects erased dispatch in shapes text scanning misses: multiline
// spellings, aliased helpers, computed members.
func TestASTVerifierCatchesStructuralForms(t *testing.T) {
	files := map[string]string{
		"core/pkg/package.ts": `
const table:
  Record<
    string,
    Function
  > = {};
function f(obj: { [k: string]: () => void }, name: string) {
  obj
    [name]
    ();
}
const g = goIfaceCall;
function h(i: unknown) {
  return (g as unknown as (a: unknown, b: string, c: unknown[]) => unknown)(i, "M", []);
}
`,
	}
	report, err := VerifyAST(files, typescriptDir(t))
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, v := range report.Violations {
		found[v.Pattern] = true
	}
	for _, want := range []string{"string-function-registry", "computed-member-call", "erased-function-type"} {
		if !found[want] {
			t.Errorf("AST verifier missed %s; got %+v", want, report.Violations)
		}
	}
}

// TestASTVerifierPassesDirectCode proves clean generated shapes carry
// zero verdicts and mechanism detection is identifier-resolved.
func TestASTVerifierPassesDirectCode(t *testing.T) {
	files := map[string]string{
		"core/pkg/package.ts": `
import * as gosl$ from "../../language-abi/goslice.js";
export function f(): bigint {
  const s = gosl$.goSliceFrom([1n]);
  return gosl$.goSliceLen(s);
}
// goIfaceCall mentioned only in a comment is not a call.
const record = { M: (x: bigint): bigint => x };
export const viaTypedProperty = record.M(2n);
`,
	}
	report, err := VerifyAST(files, typescriptDir(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Violations) != 0 {
		t.Errorf("clean code flagged: %+v", report.Violations)
	}
	if files := report.Mechanisms["slice-carrier"]; len(files) != 1 {
		t.Errorf("slice-carrier mechanism not derived from identifiers: %+v", report.Mechanisms)
	}
}
