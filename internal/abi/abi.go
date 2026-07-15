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
const Version = 1

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
		"gopanic.ts": gopanicSource,
		"goints.ts":  gointsSource,
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
