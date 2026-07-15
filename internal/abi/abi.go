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
const Version = 2

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
	}
}

const gopanicSource = `// The Go panic carrier: distinguishes translated Go panics from host
// errors and carries the exact Go runtime-error message.
export class GoPanic extends Error {
  goMessage: string;
  constructor(message: string) {
    super(message);
    this.name = "GoPanic";
    this.goMessage = message;
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
// UTF-8 string semantics.
import { GoPanic, goPanicNil, goPanicNilMapWrite } from "./gopanic.js";

// A Go map: undefined is the nil map.
export type GoMap<K, V> = Map<K, V> | undefined;

export function goNilCheck<T>(x: T | undefined): T {
  if (x === undefined) goPanicNil();
  return x;
}

export function goMapMake<K, V>(): Map<K, V> {
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

// len(string) is the UTF-8 byte length; JS string length counts UTF-16
// code units. Surrogate pairs (one code point, 4 UTF-8 bytes) and lone
// surrogates (encoded as 3-byte replacement U+FFFD by Go's conversion,
// but valid strings never contain them) are handled by code-point math.
export function goStringLen(s: string): bigint {
  let bytes = 0;
  for (let index = 0; index < s.length; index++) {
    const code = s.charCodeAt(index);
    if (code < 0x80) bytes += 1;
    else if (code < 0x800) bytes += 2;
    else if (code >= 0xd800 && code < 0xdc00 && index + 1 < s.length) {
      const next = s.charCodeAt(index + 1);
      if (next >= 0xdc00 && next < 0xe000) {
        bytes += 4;
        index++;
      } else {
        bytes += 3;
      }
    } else bytes += 3;
  }
  return BigInt(bytes);
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
`

const gosliceSource = `// The exact Go slice carrier: backing array, offset, length, and
// capacity, with undefined as the nil slice. Aliasing, capacity reuse on
// append, reslicing, and Go's exact bounds panics are all preserved.
// Indices arrive as the static carrier of their Go type (number or
// bigint); JS mixed comparisons are exact, and conversion to an array
// offset happens only after the bounds proof.
import { GoPanic } from "./gopanic.js";

export class GoSlice<T> {
  backing: T[];
  offset: number;
  length: number;
  capacity: number;
  constructor(backing: T[], offset: number, length: number, capacity: number) {
    this.backing = backing;
    this.offset = offset;
    this.length = length;
    this.capacity = capacity;
  }
}

// undefined is the nil slice (len 0, cap 0).
export type GoSliceValue<T> = GoSlice<T> | undefined;

type GoIndex = number | bigint;

function panicIndex(index: GoIndex, length: number): never {
  throw new GoPanic("runtime error: index out of range [" + String(index) + "] with length " + String(length));
}

export function goSliceFrom<T>(values: T[]): GoSlice<T> {
  return new GoSlice(values, 0, values.length, values.length);
}

export function goSliceMake<T>(length: GoIndex, capacity: GoIndex, zero: T): GoSlice<T> {
  if (length < 0 || capacity < 0 || length > capacity) {
    throw new GoPanic("runtime error: makeslice: len out of range");
  }
  const cap = Number(capacity);
  const backing: T[] = new Array(cap);
  for (let index = 0; index < cap; index++) {
    backing[index] = zero;
  }
  return new GoSlice(backing, 0, Number(length), cap);
}

export function goSliceLen<T>(s: GoSliceValue<T>): bigint {
  return s === undefined ? 0n : BigInt(s.length);
}

export function goSliceCap<T>(s: GoSliceValue<T>): bigint {
  return s === undefined ? 0n : BigInt(s.capacity);
}

export function goSliceGet<T>(s: GoSliceValue<T>, index: GoIndex): T {
  const length = s === undefined ? 0 : s.length;
  if (index < 0 || index >= length) panicIndex(index, length);
  const slice = s as GoSlice<T>;
  return slice.backing[slice.offset + Number(index)] as T;
}

export function goSliceSet<T>(s: GoSliceValue<T>, index: GoIndex, value: T): void {
  const length = s === undefined ? 0 : s.length;
  if (index < 0 || index >= length) panicIndex(index, length);
  const slice = s as GoSlice<T>;
  slice.backing[slice.offset + Number(index)] = value;
}

// GoStructValue is the generated contract of every struct-value class:
// deep copy along the value-struct spine, in-place field overwrite, and
// a fresh zero value.
export interface GoStructValue<T> {
  goClone$(): T;
  goSet$(other: T): void;
}

// make([]T, len[, cap]) for struct elements: every element is a distinct
// fresh zero instance.
export function goSliceMakeStruct<T>(length: GoIndex, capacity: GoIndex, zero: () => T): GoSlice<T> {
  if (length < 0 || capacity < 0 || length > capacity) {
    throw new GoPanic("runtime error: makeslice: len out of range");
  }
  const cap = Number(capacity);
  const backing: T[] = new Array(cap);
  for (let index = 0; index < cap; index++) {
    backing[index] = zero();
  }
  return new GoSlice(backing, 0, Number(length), cap);
}

// s[i] = v for struct elements: Go overwrites the element's memory, so
// the stored instance's fields are copied in place and every alias of
// the element observes the store.
export function goSliceSetStruct<T extends GoStructValue<T>>(s: GoSliceValue<T>, index: GoIndex, value: T): void {
  const length = s === undefined ? 0 : s.length;
  if (index < 0 || index >= length) panicIndex(index, length);
  const slice = s as GoSlice<T>;
  (slice.backing[slice.offset + Number(index)] as T).goSet$(value);
}

// s[low:high]: 0 <= low <= high <= cap; shares backing storage. A nil
// slice permits only [0:0].
export function goSliceSlice<T>(s: GoSliceValue<T>, low: GoIndex, high: GoIndex): GoSliceValue<T> {
  const capacity = s === undefined ? 0 : s.capacity;
  if (high > capacity || high < 0) {
    throw new GoPanic("runtime error: slice bounds out of range [:" + String(high) + "] with capacity " + String(capacity));
  }
  if (low < 0 || low > high) {
    throw new GoPanic("runtime error: slice bounds out of range [" + String(low) + ":" + String(high) + "]");
  }
  if (s === undefined) {
    return undefined; // only [0:0] reaches here; nil stays nil
  }
  return new GoSlice(s.backing, s.offset + Number(low), Number(high) - Number(low), s.capacity - Number(low));
}

// append: reuses backing storage when capacity allows (aliasing is
// observable); otherwise allocates. Go leaves the grown capacity
// implementation-defined; this carrier uses the documented doubling /
// 1.25x progression without allocator size-class rounding, and grown
// capacity is not a differential target.
export function goSliceAppend<T>(s: GoSliceValue<T>, values: T[]): GoSlice<T> {
  const length = s === undefined ? 0 : s.length;
  const needed = length + values.length;
  if (s !== undefined && needed <= s.capacity) {
    for (let index = 0; index < values.length; index++) {
      s.backing[s.offset + length + index] = values[index] as T;
    }
    return new GoSlice(s.backing, s.offset, needed, s.capacity);
  }
  let capacity = s === undefined ? 0 : s.capacity;
  if (capacity === 0) capacity = needed;
  while (capacity < needed) {
    capacity = capacity < 256 ? capacity * 2 : capacity + Math.trunc((capacity + 3 * 256) / 4);
  }
  const backing: T[] = new Array(capacity);
  if (s !== undefined) {
    for (let index = 0; index < length; index++) {
      backing[index] = s.backing[s.offset + index] as T;
    }
  }
  for (let index = 0; index < values.length; index++) {
    backing[length + index] = values[index] as T;
  }
  return new GoSlice(backing, 0, needed, capacity);
}

// append(s, source...): source's current elements are read first, so a
// self-append with capacity reuse stays exact.
export function goSliceAppendSlice<T>(s: GoSliceValue<T>, source: GoSliceValue<T>): GoSlice<T> {
  const values: T[] = [];
  const length = source === undefined ? 0 : source.length;
  for (let index = 0; index < length; index++) {
    const src = source as GoSlice<T>;
    values.push(src.backing[src.offset + index] as T);
  }
  return goSliceAppend(s, values);
}

// The struct-element spread clones each copied value: Go copies struct
// values into the destination's storage.
export function goSliceAppendSliceStruct<T extends GoStructValue<T>>(s: GoSliceValue<T>, source: GoSliceValue<T>): GoSlice<T> {
  const values: T[] = [];
  const length = source === undefined ? 0 : source.length;
  for (let index = 0; index < length; index++) {
    const src = source as GoSlice<T>;
    values.push((src.backing[src.offset + index] as T).goClone$());
  }
  return goSliceAppend(s, values);
}

// copy(dst, src): min(len) elements with memmove semantics — the source
// window is read completely before any write, so overlapping ranges on
// shared backing stay exact.
export function goSliceCopy<T>(dst: GoSliceValue<T>, src: GoSliceValue<T>): bigint {
  const dstLength = dst === undefined ? 0 : dst.length;
  const srcLength = src === undefined ? 0 : src.length;
  const count = dstLength < srcLength ? dstLength : srcLength;
  const staged: T[] = [];
  for (let index = 0; index < count; index++) {
    const source = src as GoSlice<T>;
    staged.push(source.backing[source.offset + index] as T);
  }
  for (let index = 0; index < count; index++) {
    const destination = dst as GoSlice<T>;
    destination.backing[destination.offset + index] = staged[index] as T;
  }
  return BigInt(count);
}

// The struct-element copy stages clones and overwrites destination
// elements in place, so element aliases observe the store like Go's
// memory write.
export function goSliceCopyStruct<T extends GoStructValue<T>>(dst: GoSliceValue<T>, src: GoSliceValue<T>): bigint {
  const dstLength = dst === undefined ? 0 : dst.length;
  const srcLength = src === undefined ? 0 : src.length;
  const count = dstLength < srcLength ? dstLength : srcLength;
  const staged: T[] = [];
  for (let index = 0; index < count; index++) {
    const source = src as GoSlice<T>;
    staged.push((source.backing[source.offset + index] as T).goClone$());
  }
  for (let index = 0; index < count; index++) {
    const destination = dst as GoSlice<T>;
    (destination.backing[destination.offset + index] as T).goSet$(staged[index] as T);
  }
  return BigInt(count);
}
`
