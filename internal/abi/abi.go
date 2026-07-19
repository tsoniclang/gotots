// Package abi owns the generated Go language ABI: the versioned TypeScript
// module implementing exact Go language semantics (integer widths,
// wrapping, division and shift panics, panic carriers) that generated code
// calls. The ABI is generator-owned static output — it carries provenance,
// is byte-deterministic, and uses only erasable TypeScript syntax so the
// pinned Node toolchain can execute it directly during oracle runs.
//
// Helper selection is table-driven: the emitter asks for the helper of one
// (carrier family, operation) pair, so the emitter and the ABI cannot
// drift apart.
package abi

import "fmt"

// Version identifies the ABI contract carried in generated output.
const Version = 32

// Family is the static carrier family of an integer kind.
type Family string

const (
	// FamilyNumberSigned covers int8/int16/int32 carried as JS number.
	FamilyNumberSigned Family = "numberSigned"
	// FamilyNumberUnsigned covers uint8/uint16/uint32 carried as JS number.
	FamilyNumberUnsigned Family = "numberUnsigned"
	// FamilyBigSigned covers int64/int carried as bigint (asIntN range).
	FamilyBigSigned Family = "bigSigned"
	// FamilyBigUnsigned covers uint64/uint/uintptr as bigint (asUintN).
	FamilyBigUnsigned Family = "bigUnsigned"
)

// Wrap returns the canonicalization helper for a number-family width
// (goInt8, goUint32, ...) or the bigint family (goInt64, goUint64).
func Wrap(family Family, bits int) string {
	switch family {
	case FamilyNumberSigned:
		return fmt.Sprintf("goInt%d", bits)
	case FamilyNumberUnsigned:
		return fmt.Sprintf("goUint%d", bits)
	case FamilyBigSigned:
		return "goInt64"
	case FamilyBigUnsigned:
		return "goUint64"
	}
	return ""
}

// Op returns the operation helper name for a family/width/op triple, or ""
// when the emitter should lower the operation directly.
func Op(family Family, bits int, op string) string {
	base := Wrap(family, bits)
	switch op {
	case "div", "rem", "shl", "shr", "mul", "add", "sub", "and", "or", "xor", "andNot", "neg", "not":
		suffix := map[string]string{
			"div": "Div", "rem": "Rem", "shl": "Shl", "shr": "Shr", "mul": "Mul",
			"add": "Add", "sub": "Sub", "and": "And", "or": "Or", "xor": "Xor",
			"andNot": "AndNot", "neg": "Neg", "not": "Not",
		}[op]
		return base + suffix
	}
	return ""
}

// Files returns the complete deterministic ABI module set, keyed by path
// relative to the language-abi root.
func Files() map[string]string {
	return map[string]string{
		"gopanic.ts":   gopanicSource,
		"goints.ts":    gointsSource,
		"goruntime.ts": goruntimeSource,
		"goslice.ts":   gosliceSource,
		"goiface.ts":   goifaceSource,
		"goextern.ts":  goexternSource,
	}
}

const goruntimeSource = `// Go language runtime carriers beyond integers: nil-checked pointer
// access, maps with exact nil/zero/comma-ok/write-panic behavior, and
// byte-string semantics (one code unit per Go byte, canonical).
import { GoPanic, goPanicNil, goPanicNilMapWrite } from "./gopanic.js";

export { goPanicNil } from "./gopanic.js";

// A Go map: undefined is the nil map.
export type GoMap<K, V> = Map<K, V> | undefined;

type GoStrIndex = number | bigint;

// A mutable cell: the carrier of a pointer to a value with no object
// identity of its own (scalars, strings, slices, maps, functions,
// interfaces, nested pointers). The cell is created where the address
// is taken; undefined is the nil pointer.
export type GoCell<T> = { v: T };

// GoPtr is the pointer-to-type-parameter carrier: it resolves PER
// INSTANTIATION to exactly the concrete pointer representation — the
// instance itself for identity carriers (structs, arrays: objects), a
// cell for value carriers (scalars, strings, and every nilable form).
// The non-distributive [T] extends [object] keeps union-typed bindings
// (pointers, interfaces, slices, maps — all nilable spellings) on the
// cell branch as one piece.
export type GoPtr<T> = [T] extends [object] ? T : GoCell<T>;

// GoChanHandle is the opaque carrier of a channel TYPE position: no
// emitted code can construct one (channel operations are outside the
// reviewed subset), so a channel-typed field or variable only ever holds
// the nil channel (undefined).
export interface GoChanHandle {
  readonly goChan$: never;
}

// goPanicRangeExit is the exact runtime panic when a range-over-func
// sequence keeps yielding after the loop body stopped iteration.
// goPanicConversion is Go's failed type-assertion panic with the exact
// dynamic-type, source, and target spellings.
export function goPanicConversion(dynamic: string, sourceDisplay: string, targetDisplay: string): never {
  if (dynamic === "nil") {
    throw new GoPanic("interface conversion: " + sourceDisplay + " is nil, not " + targetDisplay);
  }
  throw new GoPanic("interface conversion: " + sourceDisplay + " is " + dynamic + ", not " + targetDisplay);
}

export function goPanicRangeExit(): never {
  throw new GoPanic("runtime error: range function continued iteration after function for loop body returned false");
}

// goPanicUnreachableType is the closed-world default of a token switch:
// the dynamic type set is complete, so this never fires under correct
// generation; it fails closed rather than silently mis-dispatching.
export function goPanicUnreachableType(display: string): never {
  throw new GoPanic("GOTOTS_UNREACHABLE: dynamic type " + display + " outside the resolved dispatch set");
}

// goIndirect returns its function unchanged: an identity indirection
// that keeps TypeScript's IIFE control-flow analysis from treating an
// always-panicking (provably unreachable) dispatch as making the
// caller's subsequent statements unreachable.
export function goIndirect<F>(f: F): F {
  return f;
}

// goNil is the typed nil initializer of a nilable local: the declared
// carrier type flows through untouched, so control-flow analysis never
// narrows the binding to the undefined literal — a narrowing that would
// hide closure assignments from every later use (the classic tsc
// callback-assignment blind spot) and collapse generic helper inference
// to unknown.
export function goNil<T>(): T {
  return undefined as unknown as T;
}

export function goNilCheck<T>(x: T | undefined): T {
  if (x === undefined) goPanicNil();
  return x;
}

export function goMapMake<K, V>(hint?: unknown): Map<K, V> {
  void hint; // the capacity hint evaluates but changes nothing
  return new Map<K, V>();
}

export function goMapFrom<K, V>(entries: readonly (readonly [K, V])[]): Map<K, V> {
  const out = new Map<K, V>();
  for (const [key, value] of entries) {
    out.set(key, value);
  }
  return out;
}

// Reads of a nil map and missing keys yield the zero value. A stored value
// may itself be undefined (a nil pointer), so presence uses has(), never a
// get() sentinel.
export function goMapGet<K, V>(m: GoMap<K, V>, key: K, zero: V): V {
  if (m === undefined || !m.has(key)) return zero;
  return m.get(key) as V;
}

export function goMapLookup<K, V>(m: GoMap<K, V>, key: K, zero: V): readonly [V, boolean] {
  if (m === undefined || !m.has(key)) return [zero, false];
  return [m.get(key) as V, true];
}

export function goMapSet<K, V>(m: GoMap<K, V>, key: K, value: V): void {
  if (m === undefined) goPanicNilMapWrite();
  m.set(key, value);
}

export function goMapDelete<K, V>(m: GoMap<K, V>, key: K): void {
  if (m === undefined) return;
  m.delete(key);
}

export function goMapLen<K, V>(m: GoMap<K, V>): bigint {
  return m === undefined ? 0n : BigInt(m.size);
}

// Range iteration sources: a nil map ranges zero times. Central helpers
// keep the nil test inside the ABI so a provably nil operand (which
// TypeScript narrows to literal undefined) still typechecks exactly.
export function goMapEntries<K, V>(m: GoMap<K, V>): readonly (readonly [K, V])[] {
  if (m === undefined) return [];
  return Array.from(m.entries());
}

// clear(m): a no-op on nil maps.
export function goMapClear<K, V>(m: GoMap<K, V>): void {
  if (m === undefined) return;
  m.clear();
}

// panic(v) for values whose Go %v formatting coincides with JS String():
// strings, canonical-range and bigint integers, and booleans.
export function goPanicValue(value: string | number | bigint | boolean): never {
  throw new GoPanic(String(value));
}

// goBodyUnimplemented is the body of a materialized-but-not-yet-lowered
// declaration: the signature is exact and typechecks, but the body has an
// operation outside the reviewed subset, so calling it fails closed. It is
// an ANALYSIS placeholder — the package is publication-withheld until the
// body is genuinely lowered — never a silently wrong result. Returns never,
// so a single call satisfies any declared return type.
export function goBodyUnimplemented(id: string): never {
  throw new GoPanic("GOTOTS_BODY_UNIMPLEMENTED: " + id);
}

// goPtrParamSet is the pointee store through an identity-carrier *T under
// a type parameter: the closed instantiation evidence proves every binding
// is an identity carrier with an in-place set, so a missing set operation
// here is a construction-invariant violation, never a silent rebind.
export function goPtrParamSet<T>(p: T | undefined, v: T, setElem: ((d: T, s: T) => void) | undefined): void {
  if (setElem === undefined) {
    throw new GoPanic("GOTOTS_INVARIANT: identity pointee store without a set operation");
  }
  setElem(goNilCheck(p), v);
}

// goEqUnsupported spells the equality operation of a type-parameter
// binding whose carrier is not comparable: Go's type system proves the
// operation unreachable in any admitted body, so reaching it is a
// construction-invariant violation.
export function goEqUnsupported(): never {
  throw new GoPanic("GOTOTS_INVARIANT: equality on a non-comparable type-parameter binding");
}

// panic(err): the message is the error's dynamic Error() result — the
// %v of an error value — dispatched here; a nil error panics like
// panic(nil).
export function goPanicError(err: unknown, format: () => string): never {
  if (err === undefined) {
    throw new GoPanic("panic called with nil argument");
  }
  // The typed error value is retained; the message formats lazily
  // through the closed token-switch dispatch of its Error method, so a
  // deferred mutation is reflected exactly as Go prints it.
  throw new GoPanic("panic", err, format);
}

// len(string) is the UTF-8 byte length; JS string length counts UTF-16
export function goStringLen(s: string): bigint {
  return BigInt(s.length);
}

export function goStringIndex(s: string, index: GoStrIndex): number {
  if (index < 0 || index >= s.length) {
    if (index < 0) {
      throw new GoPanic("runtime error: index out of range [" + String(index) + "]");
    }
    throw new GoPanic("runtime error: index out of range [" + String(index) + "] with length " + String(s.length));
  }
  return s.charCodeAt(Number(index));
}

// s[low:high] by byte offsets: 0 <= low <= high <= len. Mid-rune cuts
// are exact — the carrier is bytes, not text.
export function goStringSlice(s: string, low: GoStrIndex, high: GoStrIndex | undefined): string {
  if (high === undefined) high = BigInt(s.length);
  if (high > s.length || high < 0) {
    throw new GoPanic("runtime error: slice bounds out of range [:" + String(high) + "] with length " + String(s.length));
  }
  if (low < 0 || low > high) {
    throw new GoPanic("runtime error: slice bounds out of range [" + String(low) + ":" + String(high) + "]");
  }
  return s.substring(Number(low), Number(high));
}

// Rune iteration exactly as Go's range: each entry is [starting byte
// offset, decoded rune]; invalid UTF-8 yields U+FFFD advancing one byte.
export function goStringRange(s: string): [bigint, number][] {
  const out: [bigint, number][] = [];
  let i = 0;
  while (i < s.length) {
    const [r, size] = decodeRune(s, i);
    out.push([BigInt(i), r]);
    i += size;
  }
  return out;
}

// Composite map keys: a struct key's class carries goKey$(), a
// deterministic injective encoding of its comparable fields. The keyed
// carrier stores [key, value] under the encoding, so lookups follow Go
// value equality and iteration recovers the stored key.
export interface GoKeyed {
  goKey$(): string;
}

export type GoKeyedMap<K extends GoKeyed, V> = Map<string, [K, V]> | undefined;

const keyIds = new WeakMap<object, string>();
let keyIdCounter = 0;

// goKeyId is the deterministic identity component of a pointer field:
// object identity in insertion order, "u" for nil. The parameter is open
// so identity-carrier type parameters pass without narrowing; every
// non-nil identity carrier is an object (struct, array, external
// handle), so a primitive here is an admission-invariant break.
export function goKeyId(x: unknown): string {
  if (x === undefined) return "u";
  if (x === null || (typeof x !== "object" && typeof x !== "function")) {
    throw new Error("gotots invariant: non-object identity key");
  }
  let id = keyIds.get(x);
  if (id === undefined) {
    keyIdCounter++;
    id = "#" + String(keyIdCounter);
    keyIds.set(x, id);
  }
  return id;
}

// goKeyArray encodes a fixed array field element-wise (the length is
// fixed by the type, so the composition stays injective).
export function goKeyArray<T>(a: T[], encodeElem: (v: T) => string): string {
  let out = "[";
  for (let i = 0; i < a.length; i++) out += encodeElem(a[i] as T) + "|";
  return out + "]";
}
` + keyedMapsSource + `

// Float-keyed maps carry Go's exact float-key semantics. The ONLY
// divergence from JS SameValueZero is NaN: Go's NaN != NaN makes every
// NaN-keyed insert a fresh, unretrievable, undeletable entry that still
// counts and ranges. +0/-0 already coincide under SameValueZero.
export type GoFMap<V> = { m: Map<number, V>; nan: V[] } | undefined;

export function goFMapMake<V>(hint?: unknown): { m: Map<number, V>; nan: V[] } {
  void hint;
  return { m: new Map<number, V>(), nan: [] };
}

export function goFMapFrom<V>(entries: readonly (readonly [number, V])[]): { m: Map<number, V>; nan: V[] } {
  const out = goFMapMake<V>();
  for (const [key, value] of entries) {
    if (Number.isNaN(key)) {
      out.nan.push(value);
    } else {
      out.m.set(key, value);
    }
  }
  return out;
}

export function goFMapGet<V>(fm: GoFMap<V>, key: number, zero: V): V {
  if (fm === undefined || Number.isNaN(key) || !fm.m.has(key)) return zero;
  return fm.m.get(key) as V;
}

export function goFMapLookup<V>(fm: GoFMap<V>, key: number, zero: V): readonly [V, boolean] {
  if (fm === undefined || Number.isNaN(key) || !fm.m.has(key)) return [zero, false];
  return [fm.m.get(key) as V, true];
}

export function goFMapSet<V>(fm: GoFMap<V>, key: number, value: V): void {
  if (fm === undefined) goPanicNilMapWrite();
  if (Number.isNaN(key)) {
    fm.nan.push(value);
  } else {
    fm.m.set(key, value);
  }
}

export function goFMapDelete<V>(fm: GoFMap<V>, key: number): void {
  if (fm === undefined || Number.isNaN(key)) return;
  fm.m.delete(key);
}

export function goFMapLen<V>(fm: GoFMap<V>): bigint {
  return fm === undefined ? 0n : BigInt(fm.m.size + fm.nan.length);
}

export function goFMapEntries<V>(fm: GoFMap<V>): readonly (readonly [number, V])[] {
  if (fm === undefined) return [];
  const out: [number, V][] = Array.from(fm.m.entries());
  for (const value of fm.nan) out.push([NaN, value]);
  return out;
}

export function goFMapClear<V>(fm: GoFMap<V>): void {
  if (fm === undefined) return;
  fm.m.clear();
  fm.nan.length = 0;
}


export function goKMapLen<K extends GoKeyed, V>(m: GoKeyedMap<K, V>): bigint {
  return m === undefined ? 0n : BigInt(m.size);
}

// Ordered min/max over every ordered carrier (numbers, bigints, byte
// strings): NaN propagates and -0 sorts below +0, exactly Go.
export function goMin<T extends number | bigint | string>(values: T[]): T {
  let m = values[0] as T;
  for (let i = 1; i < values.length; i++) {
    const v = values[i] as T;
    if (typeof m === "number" && m !== m) return m;
    if (typeof v === "number" && v !== v) return v;
    if (v < m || (v === m && typeof v === "number" && Object.is(v, -0))) m = v;
  }
  return m;
}

export function goMax<T extends number | bigint | string>(values: T[]): T {
  let m = values[0] as T;
  for (let i = 1; i < values.length; i++) {
    const v = values[i] as T;
    if (typeof m === "number" && m !== m) return m;
    if (typeof v === "number" && v !== v) return v;
    if (v > m || (v === m && typeof m === "number" && Object.is(m, -0))) m = v;
  }
  return m;
}

// string(r): the UTF-8 encoding of one code point; out-of-range and
// surrogate values encode U+FFFD, exactly Go.
export function goStringFromRune(r: number | bigint): string {
  let code = Number(r);
  if (code < 0 || code > 0x10ffff || (code >= 0xd800 && code <= 0xdfff)) code = 0xfffd;
  if (code < 0x80) return String.fromCharCode(code);
  if (code < 0x800) {
    return String.fromCharCode(0xc0 | (code >> 6), 0x80 | (code & 0x3f));
  }
  if (code < 0x10000) {
    return String.fromCharCode(0xe0 | (code >> 12), 0x80 | ((code >> 6) & 0x3f), 0x80 | (code & 0x3f));
  }
  return String.fromCharCode(
    0xf0 | (code >> 18), 0x80 | ((code >> 12) & 0x3f), 0x80 | ((code >> 6) & 0x3f), 0x80 | (code & 0x3f));
}

// string(b) for []byte: a fresh string of the slice's bytes (nil is "").
export function goStringFromBytes(s: { length: number; backing: number[]; offset: number } | undefined): string {
  if (s === undefined) return "";
  let out = "";
  for (let i = 0; i < s.length; i++) {
    out += String.fromCharCode(s.backing[s.offset + i] as number);
  }
  return out;
}

// string(rs) for []rune: the concatenated UTF-8 encodings (nil is "").
export function goStringFromRunes(s: { length: number; backing: number[]; offset: number } | undefined): string {
  if (s === undefined) return "";
  let out = "";
  for (let i = 0; i < s.length; i++) {
    out += goStringFromRune(s.backing[s.offset + i] as number);
  }
  return out;
}

// goStringBytes / goStringRunes return the string's bytes / decoded
// runes as plain arrays for the slice carrier to wrap.
export function goStringBytes(s: string): number[] {
  const out: number[] = new Array(s.length);
  for (let i = 0; i < s.length; i++) out[i] = s.charCodeAt(i);
  return out;
}

export function goStringRunes(s: string): number[] {
  const out: number[] = [];
  let i = 0;
  while (i < s.length) {
    const [r, size] = decodeRune(s, i);
    out.push(r);
    i += size;
  }
  return out;
}

// decodeRune mirrors Go's utf8.DecodeRuneInString over the byte
// carrier: strict continuation checks, overlong/surrogate/out-of-range
// rejection, and [U+FFFD, 1] for every invalid sequence.
function decodeRune(s: string, i: number): [number, number] {
  const b0 = s.charCodeAt(i);
  if (b0 < 0x80) return [b0, 1];
  if (b0 < 0xc2 || b0 > 0xf4) return [0xfffd, 1];
  const remaining = s.length - i;
  const cont = (offset: number): number => {
    if (offset >= remaining) return -1;
    const b = s.charCodeAt(i + offset);
    return b >= 0x80 && b <= 0xbf ? b & 0x3f : -1;
  };
  if (b0 < 0xe0) {
    const c1 = cont(1);
    if (c1 < 0) return [0xfffd, 1];
    return [((b0 & 0x1f) << 6) | c1, 2];
  }
  if (b0 < 0xf0) {
    const c1 = cont(1);
    const c2 = cont(2);
    if (c1 < 0 || c2 < 0) return [0xfffd, 1];
    const r = ((b0 & 0x0f) << 12) | (c1 << 6) | c2;
    if (r < 0x800 || (r >= 0xd800 && r <= 0xdfff)) return [0xfffd, 1];
    return [r, 3];
  }
  const c1 = cont(1);
  const c2 = cont(2);
  const c3 = cont(3);
  if (c1 < 0 || c2 < 0 || c3 < 0) return [0xfffd, 1];
  const r = ((b0 & 0x07) << 18) | (c1 << 12) | (c2 << 6) | c3;
  if (r < 0x10000 || r > 0x10ffff) return [0xfffd, 1];
  return [r, 4];
}
`
