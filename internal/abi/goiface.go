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
  readonly d: string;
  // m maps canonical method identities (name, unexported package
  // qualifier, signature digest) onto statically selected functions —
  // the exact method SET of this dynamic type (value and pointer types
  // carry distinct tables).
  readonly m: Readonly<Record<string, Function>>;
  // c states comparability: true (equality defined), false (Go panics),
  // or absent (unknown — external contract — equality fails closed).
  readonly c?: boolean;
  // e is the exact equality of a comparable non-primitive value type.
  readonly e?: (a: unknown, b: unknown) => boolean;
  // p marks a pointer type: its boxed values compare by identity.
  readonly p?: boolean;
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

// Method dispatch: a nil interface panics exactly like a nil pointer
// dereference.
export function goIfaceCall(i: GoIface, method: string, args: unknown[]): unknown {
  if (i === undefined) goPanicNil();
  const boxed = i as GoIfaceBox;
  const fn = boxed.r.m[method] as Function | undefined;
  if (fn === undefined) {
    // Every table is statically populated at generation; an external
    // dynamic type dispatched beyond its recorded contract fails closed.
    throw new GoPanic("GOTOTS_EXTERNAL_UNIMPLEMENTED: " + (boxed.r.x ?? boxed.r.d) + "." + method);
  }
  return fn(boxed.v, ...args);
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

// x.(I), interface-target panic form: the dynamic type must carry
// every method of the target interface. External dynamic types cannot
// prove their method sets statically, so they fail closed.
export function goIfaceAssertIface(i: GoIface, methods: readonly (readonly [string, string])[], sourceDisplay: string, targetDisplay: string): GoIfaceBox {
  if (i === undefined) {
    throw new GoPanic("interface conversion: " + sourceDisplay + " is nil, not " + targetDisplay);
  }
  const missing = ifaceMissingMethod(i.r, methods);
  if (missing !== undefined) {
    throw new GoPanic("interface conversion: " + i.r.d + " is not " + targetDisplay + ": missing method " + missing);
  }
  return i;
}

// x.(I), comma-ok form: undefined (the nil interface) fills the miss.
export function goIfaceLookupIface(i: GoIface, methods: readonly (readonly [string, string])[]): [GoIface, boolean] {
  if (i === undefined || ifaceMissingMethod(i.r, methods) !== undefined) {
    return [undefined, false];
  }
  return [i, true];
}

// methods pairs the canonical dispatch identity with the source
// spelling for the missing-method panic.
function ifaceMissingMethod(rtti: GoRtti, methods: readonly (readonly [string, string])[]): string | undefined {
  for (const [key, display] of methods) {
    if (rtti.m[key] === undefined) {
      if (rtti.x !== undefined) {
        // The recorded external contract cannot prove the method's
        // absence; deciding either way would guess.
        throw new GoPanic("GOTOTS_EXTERNAL_UNIMPLEMENTED: method set of " + rtti.x);
      }
      return display;
    }
  }
  return undefined;
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
  // equality (floats keep NaN and signed-zero semantics).
  return { d, c: true, m: {} };
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
