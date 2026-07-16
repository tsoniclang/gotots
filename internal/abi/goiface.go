package abi

// goifaceSource is the interface-value carrier: a box pairing a shared
// per-type rtti object with the concrete value, undefined for the nil
// interface. Identity tests compare rtti objects — each is a single ESM
// export — so dynamic-type checks never consult spellings; spellings
// exist only for Go's exact panic messages.
const goifaceSource = `// Go interface-value carrier: boxed (rtti, value) pairs with exact
// nil-interface vs nil-pointer-in-interface distinction, method-table
// dispatch, and Go's interface-conversion panic messages.
import { GoPanic, goPanicNil } from "./gopanic.js";

// GoRtti identifies one concrete type: its Go display spelling (package
// names qualify, as in runtime messages), its method table mapping
// method names onto the generated method functions, and — for external
// named types — the canonical identity that routes method dispatch
// through the external-contract registry.
export interface GoRtti {
  // d is the Go runtime display of the dynamic type; it is the switch
  // discriminant identity (rtti objects are interned per canonical
  // type, so === comparison is the type token test). There is no method
  // table — dispatch is a generated exhaustive token switch of direct
  // calls.
  readonly d: string;
  // c states comparability: true (equality defined), false (Go panics),
  // or absent (unknown — external contract — equality fails closed).
  readonly c?: boolean;
  // e is the exact equality of a comparable non-primitive value type.
  readonly e?: (a: unknown, b: unknown) => boolean;
  // p marks a pointer type: its boxed values compare by identity.
  readonly p?: boolean;
  // ms is the type's sorted method-name list — data used only for the
  // missing-method diagnostic of a failed interface assertion, never for
  // dispatch (dispatch is a generated exhaustive token switch).
  readonly ms?: readonly string[];
  // x is an external type's canonical contract identity (diagnostics).
  readonly x?: string;
}

// Composite and external rttis intern per canonical type identity, so
// rtti comparison stays object identity across every module.
const compositeRttis = new Map<string, GoRtti>();

export function goRttiComposite(key: string, rtti: GoRtti): GoRtti {
  let interned = compositeRttis.get(key);
  if (interned === undefined) {
    interned = rtti;
    compositeRttis.set(key, rtti);
  }
  return interned;
}

// GoIfaceBox pairs the dynamic type with the concrete value.
export interface GoIfaceBox {
  readonly r: GoRtti;
  readonly v: unknown;
}

export type GoIface = GoIfaceBox | undefined;

export function goIfaceBox(r: GoRtti, v: unknown): GoIfaceBox {
  return { r, v };
}


export function goIfaceIs(i: GoIface, r: GoRtti): boolean {
  return i !== undefined && i.r === r;
}

// Interface equality by Go's rule: equal dynamic types, then equal
// values by that type's own equality. Uncomparable dynamic types panic
// with Go's exact message; unknown (external) comparability fails
// closed rather than guessing.
export function goIfaceEqual(a: GoIface, b: GoIface): boolean {
  if (a === undefined || b === undefined) {
    return a === b;
  }
  if (a.r !== b.r) {
    return false;
  }
  return ifaceValueEqual(a.r, a.v, b.v);
}

// Interface-against-concrete equality with === value carriers.
export function goIfaceEqualPrim(i: GoIface, r: GoRtti, v: unknown): boolean {
  return i !== undefined && i.r === r && i.v === v;
}

// Interface-against-concrete equality through the dynamic type's own
// exact equality (comparable structs and arrays).
export function goIfaceEqualVia(i: GoIface, r: GoRtti, v: unknown): boolean {
  return i !== undefined && i.r === r && (r.e as (a: unknown, b: unknown) => boolean)(i.v, v);
}

function ifaceValueEqual(r: GoRtti, a: unknown, b: unknown): boolean {
  if (r.c === false) {
    throw new GoPanic("runtime error: comparing uncomparable type " + r.d);
  }
  if (r.c === undefined) {
    throw new GoPanic("GOTOTS_EXTERNAL_UNIMPLEMENTED: equality of " + (r.x ?? r.d));
  }
  if (r.e !== undefined) {
    return r.e(a, b);
  }
  // Comparable without a value equality: primitive carriers and pointer
  // identities, both exact under ===.
  return a === b;
}

// x.(T), panic form: both messages name the static source interface.
export function goIfaceAssert(i: GoIface, r: GoRtti, sourceDisplay: string): unknown {
  if (i === undefined) {
    throw new GoPanic("interface conversion: " + sourceDisplay + " is nil, not " + r.d);
  }
  if (i.r !== r) {
    throw new GoPanic("interface conversion: " + sourceDisplay + " is " + i.r.d + ", not " + r.d);
  }
  return i.v;
}

// x.(I) as a static token-membership test: the dynamic type must be one
// of the closed set of tokens that implement the target interface
// (resolved at generation from the whole-unit type universe).
export function goIfaceAssertSet(i: GoIface, tokens: readonly GoRtti[], required: readonly string[], sourceDisplay: string, targetDisplay: string): GoIfaceBox {
  if (i === undefined) {
    throw new GoPanic("interface conversion: " + sourceDisplay + " is nil, not " + targetDisplay);
  }
  for (const token of tokens) {
    if (i.r === token) return i;
  }
  const missing = missingMethod(i.r, required);
  throw new GoPanic("interface conversion: " + i.r.d + " is not " + targetDisplay +
    (missing === undefined ? "" : ": missing method " + missing));
}

// missingMethod returns the first required method the dynamic type lacks
// (Go reports methods in the interface's sorted method order).
function missingMethod(r: GoRtti, required: readonly string[]): string | undefined {
  const have = r.ms;
  if (have === undefined) return undefined;
  for (const method of required) {
    if (!have.includes(method)) return method;
  }
  return undefined;
}

// x.(I), comma-ok form: undefined (the nil interface) fills the miss.
export function goIfaceLookupSet(i: GoIface, tokens: readonly GoRtti[]): [GoIface, boolean] {
  if (i === undefined) return [undefined, false];
  for (const token of tokens) {
    if (i.r === token) return [i, true];
  }
  return [undefined, false];
}

// x.(T), comma-ok form: the zero value fills the miss.
export function goIfaceLookup<T>(i: GoIface, r: GoRtti, zero: T): readonly [T, boolean] {
  if (i === undefined || i.r !== r) {
    return [zero, false];
  }
  return [i.v as T, true];
}

function rtti(d: string): GoRtti {
  // Every predeclared basic type is comparable, and === is its exact
  // equality (floats keep NaN and signed-zero semantics). Predeclared
  // types are methodless.
  return { d, c: true, ms: [] };
}

// Predeclared types' rttis (methodless).
export const goRtti$bool = rtti("bool");
export const goRtti$string = rtti("string");
export const goRtti$int = rtti("int");
export const goRtti$int8 = rtti("int8");
export const goRtti$int16 = rtti("int16");
export const goRtti$int32 = rtti("int32");
export const goRtti$int64 = rtti("int64");
export const goRtti$uint = rtti("uint");
export const goRtti$uint8 = rtti("uint8");
export const goRtti$uint16 = rtti("uint16");
export const goRtti$uint32 = rtti("uint32");
export const goRtti$uint64 = rtti("uint64");
export const goRtti$uintptr = rtti("uintptr");
export const goRtti$float32 = rtti("float32");
export const goRtti$float64 = rtti("float64");
`
