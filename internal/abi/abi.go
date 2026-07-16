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
const Version = 11

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

const gopanicSource = `// The Go panic carrier: distinguishes translated Go panics from host
// errors and carries the exact Go runtime-error message.
export class GoPanic extends Error {
  // value retains the exact typed panic payload through deferred
  // execution and recovery; format lazily produces the printed message
  // at the point the runtime would finally print it (after all defers).
  readonly value: unknown;
  private readonly format: () => string;
  private computed: string | undefined;
  constructor(message: string, value?: unknown, format?: () => string) {
    super(message);
    this.name = "GoPanic";
    this.value = value === undefined ? message : value;
    this.format = format ?? (() => message);
  }
  get goMessage(): string {
    if (this.computed === undefined) {
      this.computed = this.format();
    }
    return this.computed;
  }
}

export function goPanicDivide(): never {
  throw new GoPanic("runtime error: integer divide by zero");
}

export function goPanicShift(): never {
  throw new GoPanic("runtime error: negative shift amount");
}

export function goPanicNil(): never {
  throw new GoPanic("runtime error: invalid memory address or nil pointer dereference");
}

export function goPanicNilMapWrite(): never {
  throw new GoPanic("assignment to entry in nil map");
}
`

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

// goPanicRangeExit is the exact runtime panic when a range-over-func
// sequence keeps yielding after the loop body stopped iteration.
export function goPanicRangeExit(): never {
  throw new GoPanic("runtime error: range function continued iteration after function for loop body returned false");
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

// panic(err): the message is the error's dynamic Error() result — the
// %v of an error value — dispatched here; a nil error panics like
// panic(nil).
export function goPanicError(err: { r: { d: string; m: Readonly<Record<string, Function>>; x?: string }; v: unknown } | undefined, errorKey: string): never {
  if (err === undefined) {
    throw new GoPanic("panic called with nil argument");
  }
  const fn = err.r.m[errorKey] as Function | undefined;
  if (fn === undefined) {
    throw new GoPanic("GOTOTS_EXTERNAL_UNIMPLEMENTED: " + (err.r.x ?? err.r.d) + ".Error");
  }
  // The typed error value is retained; its message formats lazily, so a
  // deferred function that mutates the error is reflected exactly as Go
  // prints the panic after defers run.
  throw new GoPanic("panic", err, () => String((fn as Function)(err.v)));
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
// object identity in insertion order, "u" for nil.
export function goKeyId(x: object | undefined): string {
  if (x === undefined) return "u";
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

export function goKMapMake<K extends GoKeyed, V>(hint?: unknown): Map<string, [K, V]> {
  void hint;
  return new Map<string, [K, V]>();
}

export function goKMapFrom<K extends GoKeyed, V>(entries: readonly (readonly [K, V])[]): Map<string, [K, V]> {
  const out = new Map<string, [K, V]>();
  for (const [key, value] of entries) {
    out.set(key.goKey$(), [key, value]);
  }
  return out;
}

export function goKMapGet<K extends GoKeyed, V>(m: GoKeyedMap<K, V>, key: K, zero: V): V {
  if (m === undefined) return zero;
  const entry = m.get(key.goKey$());
  return entry === undefined ? zero : entry[1];
}

export function goKMapLookup<K extends GoKeyed, V>(m: GoKeyedMap<K, V>, key: K, zero: V): [V, boolean] {
  if (m === undefined) return [zero, false];
  const entry = m.get(key.goKey$());
  return entry === undefined ? [zero, false] : [entry[1], true];
}

export function goKMapSet<K extends GoKeyed, V>(m: GoKeyedMap<K, V>, key: K, value: V): void {
  if (m === undefined) goPanicNilMapWrite();
  (m as Map<string, [K, V]>).set(key.goKey$(), [key, value]);
}

export function goKMapDelete<K extends GoKeyed, V>(m: GoKeyedMap<K, V>, key: K): void {
  if (m !== undefined) m.delete(key.goKey$());
}

export function goKMapClear<K extends GoKeyed, V>(m: GoKeyedMap<K, V>): void {
  if (m !== undefined) m.clear();
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

const gointsSource = `// Exact Go integer semantics. Signed and unsigned 8/16/32-bit values are
// carried as JS numbers canonicalized to their Go range; 64-bit values are
// carried as bigint (BigInt.asIntN / BigInt.asUintN canonical ranges).
import { goPanicDivide, goPanicShift } from "./gopanic.js";

export type GoInt8 = number;
export type GoInt16 = number;
export type GoInt32 = number;
export type GoUint8 = number;
export type GoUint16 = number;
export type GoUint32 = number;
export type GoInt64 = bigint;
export type GoInt = bigint;
export type GoUint64 = bigint;
export type GoUint = bigint;
export type GoUintptr = bigint;

export function goInt8(x: number): number { return (x << 24) >> 24; }
export function goInt16(x: number): number { return (x << 16) >> 16; }
export function goInt32(x: number): number { return x | 0; }
export function goUint8(x: number): number { return x & 0xff; }
export function goUint16(x: number): number { return x & 0xffff; }
export function goUint32(x: number): number { return x >>> 0; }
export function goInt64(x: bigint): bigint { return BigInt.asIntN(64, x); }
export function goUint64(x: bigint): bigint { return BigInt.asUintN(64, x); }

// Multiplication: 32-bit operands overflow double precision, so exact
// 32-bit lanes use Math.imul; narrower widths fit exactly.
export function goInt8Mul(a: number, b: number): number { return goInt8(a * b); }
export function goInt16Mul(a: number, b: number): number { return goInt16(a * b); }
export function goInt32Mul(a: number, b: number): number { return Math.imul(a, b); }
export function goUint8Mul(a: number, b: number): number { return goUint8(a * b); }
export function goUint16Mul(a: number, b: number): number { return goUint16(a * b); }
export function goUint32Mul(a: number, b: number): number { return Math.imul(a, b) >>> 0; }
export function goInt64Mul(a: bigint, b: bigint): bigint { return BigInt.asIntN(64, a * b); }
export function goUint64Mul(a: bigint, b: bigint): bigint { return BigInt.asUintN(64, a * b); }

// Division truncates toward zero (matching JS Math.trunc and bigint /);
// division by zero is the exact Go runtime panic; the most negative value
// divided by -1 wraps.
export function goInt8Div(a: number, b: number): number { if (b === 0) goPanicDivide(); return goInt8(Math.trunc(a / b)); }
export function goInt16Div(a: number, b: number): number { if (b === 0) goPanicDivide(); return goInt16(Math.trunc(a / b)); }
export function goInt32Div(a: number, b: number): number { if (b === 0) goPanicDivide(); return goInt32(Math.trunc(a / b)); }
export function goUint8Div(a: number, b: number): number { if (b === 0) goPanicDivide(); return goUint8(Math.trunc(a / b)); }
export function goUint16Div(a: number, b: number): number { if (b === 0) goPanicDivide(); return goUint16(Math.trunc(a / b)); }
export function goUint32Div(a: number, b: number): number { if (b === 0) goPanicDivide(); return goUint32(Math.trunc(a / b)); }
export function goInt64Div(a: bigint, b: bigint): bigint { if (b === 0n) goPanicDivide(); return BigInt.asIntN(64, a / b); }
export function goUint64Div(a: bigint, b: bigint): bigint { if (b === 0n) goPanicDivide(); return BigInt.asUintN(64, a / b); }

// Remainder keeps the dividend's sign in both Go and JS.
export function goInt8Rem(a: number, b: number): number { if (b === 0) goPanicDivide(); return goInt8(a % b); }
export function goInt16Rem(a: number, b: number): number { if (b === 0) goPanicDivide(); return goInt16(a % b); }
export function goInt32Rem(a: number, b: number): number { if (b === 0) goPanicDivide(); return goInt32(a % b); }
export function goUint8Rem(a: number, b: number): number { if (b === 0) goPanicDivide(); return goUint8(a % b); }
export function goUint16Rem(a: number, b: number): number { if (b === 0) goPanicDivide(); return goUint16(a % b); }
export function goUint32Rem(a: number, b: number): number { if (b === 0) goPanicDivide(); return goUint32(a % b); }
export function goInt64Rem(a: bigint, b: bigint): bigint { if (b === 0n) goPanicDivide(); return BigInt.asIntN(64, a % b); }
export function goUint64Rem(a: bigint, b: bigint): bigint { if (b === 0n) goPanicDivide(); return BigInt.asUintN(64, a % b); }

// Shifts: a negative count is the exact Go panic; a count at or beyond the
// width shifts everything out (sign-filling for signed right shifts). The
// clamp keeps JS's 32-bit shift-count masking from being observable.
function goShiftCount(n: number): number { if (n < 0) goPanicShift(); return n; }
export function goShiftCountFromBig(n: bigint): number { if (n < 0n) goPanicShift(); return n > 64n ? 64 : Number(n); }

export function goInt8Shl(a: number, n: number): number { n = goShiftCount(n); return n >= 8 ? 0 : goInt8(a << n); }
export function goInt16Shl(a: number, n: number): number { n = goShiftCount(n); return n >= 16 ? 0 : goInt16(a << n); }
export function goInt32Shl(a: number, n: number): number { n = goShiftCount(n); return n >= 32 ? 0 : goInt32(a << n); }
export function goUint8Shl(a: number, n: number): number { n = goShiftCount(n); return n >= 8 ? 0 : goUint8(a << n); }
export function goUint16Shl(a: number, n: number): number { n = goShiftCount(n); return n >= 16 ? 0 : goUint16(a << n); }
export function goUint32Shl(a: number, n: number): number { n = goShiftCount(n); return n >= 32 ? 0 : goUint32(a << n); }
export function goInt8Shr(a: number, n: number): number { n = goShiftCount(n); return n >= 8 ? (a < 0 ? -1 : 0) : goInt8(a >> n); }
export function goInt16Shr(a: number, n: number): number { n = goShiftCount(n); return n >= 16 ? (a < 0 ? -1 : 0) : goInt16(a >> n); }
export function goInt32Shr(a: number, n: number): number { n = goShiftCount(n); return n >= 32 ? (a < 0 ? -1 : 0) : goInt32(a >> n); }
export function goUint8Shr(a: number, n: number): number { n = goShiftCount(n); return n >= 8 ? 0 : goUint8(a >>> n); }
export function goUint16Shr(a: number, n: number): number { n = goShiftCount(n); return n >= 16 ? 0 : goUint16(a >>> n); }
export function goUint32Shr(a: number, n: number): number { n = goShiftCount(n); return n >= 32 ? 0 : a >>> n; }
export function goInt64Shl(a: bigint, n: number): bigint { n = goShiftCount(n); return n >= 64 ? 0n : BigInt.asIntN(64, a << BigInt(n)); }
export function goInt64Shr(a: bigint, n: number): bigint { n = goShiftCount(n); return n >= 64 ? (a < 0n ? -1n : 0n) : a >> BigInt(n); }
export function goUint64Shl(a: bigint, n: number): bigint { n = goShiftCount(n); return n >= 64 ? 0n : BigInt.asUintN(64, a << BigInt(n)); }
export function goUint64Shr(a: bigint, n: number): bigint { n = goShiftCount(n); return n >= 64 ? 0n : a >> BigInt(n); }

// 64-bit lane arithmetic and bitwise operations.
export function goInt64Add(a: bigint, b: bigint): bigint { return BigInt.asIntN(64, a + b); }
export function goInt64Sub(a: bigint, b: bigint): bigint { return BigInt.asIntN(64, a - b); }
export function goUint64Add(a: bigint, b: bigint): bigint { return BigInt.asUintN(64, a + b); }
export function goUint64Sub(a: bigint, b: bigint): bigint { return BigInt.asUintN(64, a - b); }
export function goInt64And(a: bigint, b: bigint): bigint { return BigInt.asIntN(64, a & b); }
export function goInt64Or(a: bigint, b: bigint): bigint { return BigInt.asIntN(64, a | b); }
export function goInt64Xor(a: bigint, b: bigint): bigint { return BigInt.asIntN(64, a ^ b); }
export function goInt64AndNot(a: bigint, b: bigint): bigint { return BigInt.asIntN(64, a & ~b); }
export function goUint64And(a: bigint, b: bigint): bigint { return BigInt.asUintN(64, a & b); }
export function goUint64Or(a: bigint, b: bigint): bigint { return BigInt.asUintN(64, a | b); }
export function goUint64Xor(a: bigint, b: bigint): bigint { return BigInt.asUintN(64, a ^ b); }
export function goUint64AndNot(a: bigint, b: bigint): bigint { return BigInt.asUintN(64, a & ~b); }
export function goInt64Neg(a: bigint): bigint { return BigInt.asIntN(64, -a); }
export function goUint64Neg(a: bigint): bigint { return BigInt.asUintN(64, -a); }
export function goInt64Not(a: bigint): bigint { return BigInt.asIntN(64, ~a); }
export function goUint64Not(a: bigint): bigint { return BigInt.asUintN(64, ~a); }

// Width and family conversions between canonical carriers.
export function goInt8FromBig(x: bigint): number { return Number(BigInt.asIntN(8, x)); }
export function goInt16FromBig(x: bigint): number { return Number(BigInt.asIntN(16, x)); }
export function goInt32FromBig(x: bigint): number { return Number(BigInt.asIntN(32, x)); }
export function goUint8FromBig(x: bigint): number { return Number(BigInt.asUintN(8, x)); }
export function goUint16FromBig(x: bigint): number { return Number(BigInt.asUintN(16, x)); }
export function goUint32FromBig(x: bigint): number { return Number(BigInt.asUintN(32, x)); }
export function goInt64FromNumber(x: number): bigint { return BigInt.asIntN(64, BigInt(x)); }
export function goUint64FromNumber(x: number): bigint { return BigInt.asUintN(64, BigInt(x)); }

// float64 to signed integers: truncation toward zero; NaN and
// out-of-range inputs produce the pinned amd64 conversion values
// (cvttsd2si's integer-indefinite results).
export function goInt64FromFloat(f: number): bigint {
  if (Number.isNaN(f) || f >= 9223372036854775808 || f < -9223372036854775808) {
    return -9223372036854775808n;
  }
  return BigInt(Math.trunc(f));
}

export function goInt32FromFloat(f: number): number {
  if (Number.isNaN(f) || f >= 2147483648 || f < -2147483648) {
    return -2147483648;
  }
  return Math.trunc(f);
}
`
