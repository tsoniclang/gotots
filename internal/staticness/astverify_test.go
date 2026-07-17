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

// TestASTVerifierFollowsAliases proves the symbol-resolved walk catches
// an aliased banned helper — the audit's exact bypass:
// const invoke = goIfaceCall; invoke(box, "Read", args).
func TestASTVerifierFollowsAliases(t *testing.T) {
	files := map[string]string{
		"language-abi/legacy.ts": `
export function goIfaceCall(i: unknown, m: string, a: unknown[]): unknown {
  return undefined;
}
`,
		"core/pkg/package.ts": `
import { goIfaceCall } from "../../language-abi/legacy.ts";
const invoke = goIfaceCall;
export function f(box: unknown): unknown {
  return invoke(box, "Read", []);
}
`,
	}
	report, err := VerifyAST(files, typescriptDir(t))
	if err != nil {
		t.Fatal(err)
	}
	caught := false
	for _, v := range report.Violations {
		if v.Pattern == "erased-dispatch-helper" && v.File == "core/pkg/package.ts" {
			caught = true
		}
	}
	if !caught {
		t.Errorf("aliased banned-helper invocation not caught: %+v", report.Violations)
	}
}

// TestASTVerifierMechanismBySymbolResolution proves mechanism discovery
// derives from resolved declaring files, not identifier text: a local
// declaration named goIfaceBox is not a mechanism use, while an imported
// ABI symbol is, and the ABI's own definition is not a use site.
func TestASTVerifierMechanismBySymbolResolution(t *testing.T) {
	files := map[string]string{
		"language-abi/goslice.ts": `
export class GoSlice<T> { constructor(readonly backing: T[]) {} }
export function goSliceFrom<T>(v: T[]): GoSlice<T> { return new GoSlice(v); }
`,
		"core/user/package.ts": `
import { goSliceFrom } from "../../language-abi/goslice.ts";
export const s = goSliceFrom([1n]);
`,
		"core/localname/package.ts": `
function goIfaceBox(x: number): number { return x; }
export const y = goIfaceBox(1);
`,
	}
	report, err := VerifyAST(files, typescriptDir(t))
	if err != nil {
		t.Fatal(err)
	}
	users := report.Mechanisms["slice-carrier"]
	if len(users) != 1 || users[0] != "core/user/package.ts" {
		t.Errorf("slice-carrier use sites = %v; want exactly the importing module", users)
	}
	if len(report.Mechanisms["interface-box"]) != 0 {
		t.Errorf("local goIfaceBox declaration must not register a mechanism: %v", report.Mechanisms)
	}
}

// TestASTVerifierErasedCalleeType proves an any-typed callee (erased
// dispatch through a value) receives no positive disposition.
func TestASTVerifierErasedCalleeType(t *testing.T) {
	files := map[string]string{
		"core/pkg/package.ts": `
export function f(dispatch: any): unknown {
  return dispatch("method", []);
}
`,
	}
	report, err := VerifyAST(files, typescriptDir(t))
	if err != nil {
		t.Fatal(err)
	}
	caught := false
	for _, v := range report.Violations {
		if v.Pattern == "erased-callee-type" {
			caught = true
		}
	}
	if !caught {
		t.Errorf("any-typed callee not rejected: %+v", report.Violations)
	}
}

// TestASTVerifierPassesDirectCode proves clean generated shapes carry
// zero verdicts and mechanism detection is identifier-resolved.
func TestASTVerifierPassesDirectCode(t *testing.T) {
	files := map[string]string{
		"language-abi/goslice.ts": `
export class GoSlice<T> { constructor(readonly backing: T[]) {} }
export function goSliceFrom<T>(v: T[]): GoSlice<T> { return new GoSlice(v); }
export function goSliceLen<T>(s: GoSlice<T>): bigint { return BigInt(s.backing.length); }
`,
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

// TestASTVerifierErasedPayloadInCore proves the structural checks catch
// the generic erased form the old literal search missed: a
// GoBox<string, unknown, object> type reference, a .v recovery cast, and
// a GoAnyBox reference — each in generated core.
func TestASTVerifierErasedPayloadInCore(t *testing.T) {
	files := map[string]string{
		"language-abi/goiface.ts": `
export interface GoRtti { readonly d: string }
export interface GoBox<K extends string, V, M> { readonly k: K; readonly r: GoRtti; readonly v: V; readonly m: M }
export type GoAnyBox = GoBox<string, unknown, object>;
`,
		"core/pkg/package.ts": `
import * as goif$ from "../../language-abi/goiface.ts";
export function f(b: goif$.GoBox<string, unknown, object>): unknown {
  return (b as goif$.GoAnyBox).v as string;
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
	for _, want := range []string{"erased-iface-payload", "erased-anybox-in-core"} {
		if !found[want] {
			t.Errorf("missed %s: %+v", want, report.Violations)
		}
	}
}

// TestASTVerifierCatchesLocalUnqualifiedGoBox isolates the ABI-erasure
// class: the ABI's OWN goiface.ts declares GoAnyBox as an UNQUALIFIED
// GoBox<..., unknown, ...>. That must be detected as erased-iface-payload
// exactly like the qualified goif$.GoBox form in generated modules.
func TestASTVerifierCatchesLocalUnqualifiedGoBox(t *testing.T) {
	files := map[string]string{
		"language-abi/goiface.ts": `
export interface GoRtti { readonly d: string }
export type GoBox<K extends string, V, M> = { readonly k: K; readonly r: GoRtti; readonly v: V; readonly m: M };
export type GoAnyBox = GoBox<string, unknown, object>;
`,
		"core/pkg/package.ts": `
import * as goif$ from "../../language-abi/goiface.js";
export type X = goif$.GoBox<"a", unknown, object>;
`,
	}
	report, err := VerifyAST(files, typescriptDir(t))
	if err != nil {
		t.Fatal(err)
	}
	local, qualified := 0, 0
	for _, v := range report.Violations {
		if v.Pattern != "erased-iface-payload" && v.Pattern != "abi:erased-iface-payload" {
			continue
		}
		if v.File == "language-abi/goiface.ts" {
			local++
		} else {
			qualified++
		}
	}
	if local == 0 {
		t.Errorf("the ABI's own unqualified GoBox<..., unknown, ...> was not detected as an erased payload")
	}
	if qualified == 0 {
		t.Errorf("the qualified goif$.GoBox<..., unknown, ...> was not detected as an erased payload")
	}
}

// TestASTVerifierErasedPayloadBlindSpots proves the extended GoBox-payload
// erasure check catches the top types the old any/unknown-only test
// missed: the object keyword, the empty object literal {}, and an alias
// resolving to any — while a concrete exact payload is NOT flagged.
func TestASTVerifierErasedPayloadBlindSpots(t *testing.T) {
	files := map[string]string{
		"language-abi/goiface.ts": `
export interface GoRtti { readonly d: string }
export type GoBox<K extends string, V, M> = { readonly k: K; readonly r: GoRtti; readonly v: V; readonly m: M };
`,
		"core/pkg/package.ts": `
import * as goif$ from "../../language-abi/goiface.ts";
type AliasAny = any;
export type ObjPayload = goif$.GoBox<"a", object, Record<never, never>>;
export type EmptyPayload = goif$.GoBox<"b", {}, Record<never, never>>;
export type AliasPayload = goif$.GoBox<"c", AliasAny, Record<never, never>>;
`,
		"core/clean/package.ts": `
import * as goif$ from "../../language-abi/goiface.ts";
export interface Exact { readonly n: bigint }
export type ConcretePayload = goif$.GoBox<"d", Exact, Record<never, never>>;
`,
	}
	report, err := VerifyAST(files, typescriptDir(t))
	if err != nil {
		t.Fatal(err)
	}
	erasedFiles := map[string]int{}
	for _, v := range report.Violations {
		if v.Pattern == "erased-iface-payload" || v.Pattern == "abi:erased-iface-payload" {
			erasedFiles[v.File]++
		}
	}
	// object, {}, and alias-to-any — three erased payloads in the pkg file.
	if erasedFiles["core/pkg/package.ts"] < 3 {
		t.Errorf("expected object/{}/alias-any payloads all flagged; got %d in core/pkg: %+v",
			erasedFiles["core/pkg/package.ts"], report.Violations)
	}
	// The concrete exact payload must NOT be flagged.
	if erasedFiles["core/clean/package.ts"] != 0 {
		t.Errorf("concrete exact payload wrongly flagged as erased: %+v", report.Violations)
	}
}

// TestASTVerifierAliasHiddenErasedPayload proves the mandate's Phase-1.6
// example is caught: a type alias to GoBox, instantiated with an erased
// payload, whose surface name is NOT "GoBox". Detection resolves the type
// structurally, so the alias layer cannot hide the erasure.
func TestASTVerifierAliasHiddenErasedPayload(t *testing.T) {
	files := map[string]string{
		"language-abi/goiface.ts": `
export interface GoRtti { readonly d: string }
export type GoBox<K extends string, V, M> = { readonly k: K; readonly r: GoRtti; readonly v: V; readonly m: M };
`,
		"core/pkg/package.ts": `
import * as goif$ from "../../language-abi/goiface.ts";
type Methods = Record<never, never>;
type Wrapped<V> = goif$.GoBox<"x", V, Methods>;
export type Erased = Wrapped<object>;
export interface Exact { readonly n: bigint }
export type Fine = Wrapped<Exact>;
`,
	}
	report, err := VerifyAST(files, typescriptDir(t))
	if err != nil {
		t.Fatal(err)
	}
	erased := 0
	for _, v := range report.Violations {
		if v.Pattern == "erased-iface-payload" || v.Pattern == "abi:erased-iface-payload" {
			erased++
		}
	}
	// Exactly the alias-hidden erased instantiation (Wrapped<object>); the
	// concrete Wrapped<Exact> and the Wrapped<V> alias definition are fine.
	if erased == 0 {
		t.Errorf("alias-hidden erased payload (Wrapped<object>) not caught: %+v", report.Violations)
	}
}

// TestASTVerifierGenericPassthroughNotFlagged proves a GoBox<K, V, M>
// whose payload is a bound type PARAMETER of the enclosing generic (the
// box constructor and the type's own definition) is NOT flagged: V is the
// exact payload at each instantiation, not an erased top. Erasure is
// caught only at concrete instantiations (GoBox<"k", object, M>).
func TestASTVerifierGenericPassthroughNotFlagged(t *testing.T) {
	files := map[string]string{
		"language-abi/goiface.ts": `
export interface GoRtti { readonly d: string }
export interface GoBox<K extends string, V, M> { readonly k: K; readonly r: GoRtti; readonly v: V; readonly m: M }
export function goIfaceBox<K extends string, V, M>(k: K, r: GoRtti, v: V, m: M): GoBox<K, V, M> {
  return { k, r, v, m };
}
`,
	}
	report, err := VerifyAST(files, typescriptDir(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range report.Violations {
		if v.Pattern == "erased-iface-payload" || v.Pattern == "abi:erased-iface-payload" {
			t.Errorf("parametric passthrough GoBox<K, V, M> wrongly flagged as erased payload: %+v", v)
		}
	}
}
