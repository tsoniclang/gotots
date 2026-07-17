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
// DropFirst removes a function's receiver parameter from its tuple —
// the exact typing of promoted vtable adapters.
export type DropFirst<T extends readonly unknown[]> = T extends readonly [unknown, ...infer R] ? R : never;

// GoBox is one member of a closed interface union: the literal
// discriminant k narrows the payload v and vtable m to their exact
// member types — dispatch never recovers a payload from an erased type.
export interface GoBox<K extends string, V, M> {
  readonly k: K;
  readonly r: GoRtti;
  readonly v: V;
  readonly m: M;
}

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

// IfaceBox is the helper-facing shape of an interface box: the rtti token
// r plus the payload v. Every generated interface position spells its
// EXACT closed member union; this shape is used only by the finite
// equality/assertion helpers below, which read the token r and compare
// values through the type's OWN construction-bound equality (r.e) — the
// payload is never a GoBox<...,unknown,...> and is never recovered into a
// typed position. TypeScript cannot infer a single payload type from a
// discriminated union argument, so a structural supertype (not a generic)
// is used.
type IfaceBox = { readonly r: GoRtti; readonly v: unknown };

// goPanicConversionIface is Go's failed interface-to-interface
// assertion panic: the dynamic type first, then the first missing
// method (from the token's method-name data).
export function goPanicConversionIface(i: IfaceBox | undefined, sourceDisplay: string, targetDisplay: string, required: readonly string[]): never {
  if (i === undefined) {
    throw new GoPanic("interface conversion: " + sourceDisplay + " is nil, not " + targetDisplay);
  }
  let missing = "";
  const have = i.r.ms;
  if (have !== undefined) {
    for (const method of required) {
      if (!have.includes(method)) { missing = ": missing method " + method; break; }
    }
  }
  throw new GoPanic("interface conversion: " + i.r.d + " is not " + targetDisplay + missing);
}

export function goRttiComposite(key: string, rtti: GoRtti): GoRtti {
  let interned = compositeRttis.get(key);
  if (interned === undefined) {
    interned = rtti;
    compositeRttis.set(key, rtti);
  }
  return interned;
}

export function goIfaceBox<K extends string, V, M>(k: K, r: GoRtti, v: V, m: M): GoBox<K, V, M> {
  return { k, r, v, m };
}


export function goIfaceIs(i: IfaceBox | undefined, r: GoRtti): boolean {
  return i !== undefined && i.r === r;
}

// Interface equality by Go's rule: equal dynamic types, then equal
// values by that type's own equality. Uncomparable dynamic types panic
// with Go's exact message; unknown (external) comparability fails
// closed rather than guessing. Generic over the boxes' exact member
// types so no payload is erased.
export function goIfaceEqual(a: IfaceBox | undefined, b: IfaceBox | undefined): boolean {
  if (a === undefined || b === undefined) {
    return a === b;
  }
  if (a.r !== b.r) {
    return false;
  }
  return ifaceValueEqual(a.r, a.v, b.v);
}

// Interface-against-concrete equality with === value carriers.
export function goIfaceEqualPrim(i: IfaceBox | undefined, r: GoRtti, v: unknown): boolean {
  return i !== undefined && i.r === r && i.v === v;
}

// Interface-against-concrete equality through the dynamic type's own
// exact equality (comparable structs and arrays); r.e is the type's
// construction-bound equality.
export function goIfaceEqualVia(i: IfaceBox | undefined, r: GoRtti, v: unknown): boolean {
  return i !== undefined && i.r === r && r.e !== undefined && r.e(i.v, v);
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

// Assertions (x.(T), both panic and comma-ok forms) emit INLINE at the
// use site as literal-discriminant narrowing that reads the exact member
// payload with no cast (ADR-0004); no payload-recovering helper exists.

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
