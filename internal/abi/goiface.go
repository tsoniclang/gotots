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

export function goRttiComposite(key: string, rtti: GoRtti): GoRtti {
  let interned = compositeRttis.get(key);
  if (interned === undefined) {
    interned = rtti;
    compositeRttis.set(key, rtti);
  }
  return interned;
}

// GoCompositeBox is the open-composite union member: slices, maps,
// functions, arrays, and pointers to unnamed types box under the
// disjoint "c:"-prefixed discriminant namespace; their payloads
// re-emerge only through token-checked assertions (ADR-0004).
export type GoCompositeBox = GoBox<` + "`c:${string}`" + `, unknown, Record<never, never>>;

// GoAnyBox is the helper-facing supertype of every union member: the
// helpers below read only the token r (equality, assertion membership)
// and never recover the payload — construction-bound functions on the
// token (r.e) and the member's own vtable are the only value readers.
export type GoAnyBox = GoBox<string, unknown, object>;

// GoIface is the nil-or-box helper-facing shape; generated interface
// positions spell their exact closed member unions.
export type GoIface = GoAnyBox | undefined;

export function goIfaceBox<K extends string, V, M>(k: K, r: GoRtti, v: V, m: M): GoBox<K, V, M> {
  return { k, r, v, m };
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
export function goIfaceAssertSet<T extends GoAnyBox>(i: GoIface, tokens: readonly GoRtti[], required: readonly string[], sourceDisplay: string, targetDisplay: string): T {
  if (i === undefined) {
    throw new GoPanic("interface conversion: " + sourceDisplay + " is nil, not " + targetDisplay);
  }
  for (const token of tokens) {
    // Membership of the token set is the runtime proof of the target
    // union; the conversion is checked, never blind.
    if (i.r === token) return i as T;
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

// x.(interface{}): the empty interface is universal — every non-nil
// dynamic value passes; only nil misses.
export function goIfaceAssertAny<T extends GoAnyBox>(i: GoIface, sourceDisplay: string, targetDisplay: string): T {
  if (i === undefined) {
    throw new GoPanic("interface conversion: " + sourceDisplay + " is nil, not " + targetDisplay);
  }
  return i as T;
}

export function goIfaceLookupAny<T extends GoAnyBox>(i: GoIface): [T | undefined, boolean] {
  if (i === undefined) return [undefined, false];
  return [i as T, true];
}

// x.(I), comma-ok form: undefined (the nil interface) fills the miss.
export function goIfaceLookupSet<T extends GoAnyBox>(i: GoIface, tokens: readonly GoRtti[]): [T | undefined, boolean] {
  if (i === undefined) return [undefined, false];
  for (const token of tokens) {
    if (i.r === token) return [i as T, true];
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
