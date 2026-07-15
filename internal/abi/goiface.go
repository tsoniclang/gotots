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
import { goExternalCall } from "./goextern.js";

// GoRtti identifies one concrete type: its Go display spelling (package
// names qualify, as in runtime messages), its method table mapping
// method names onto the generated method functions, and — for external
// named types — the canonical identity that routes method dispatch
// through the external-contract registry.
export interface GoRtti {
  readonly d: string;
  readonly m: Readonly<Record<string, Function>>;
  readonly x?: string;
  // p marks a pointer type: its boxed values compare by identity.
  readonly p?: boolean;
}

// Composite and external rttis intern per canonical type identity, so
// rtti comparison stays object identity across every module.
const compositeRttis = new Map<string, GoRtti>();

export function goRttiComposite(key: string, display: string, externId: string | undefined): GoRtti {
  let rtti = compositeRttis.get(key);
  if (rtti === undefined) {
    rtti = externId === undefined ? { d: display, m: {} } : { d: display, m: {}, x: externId };
    compositeRttis.set(key, rtti);
  }
  return rtti;
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
  if (fn === undefined && boxed.r.x !== undefined) {
    return goExternalCall(boxed.r.x + "." + method, [boxed.v, ...args]);
  }
  return (fn as Function)(boxed.v, ...args);
}

export function goIfaceIs(i: GoIface, r: GoRtti): boolean {
  return i !== undefined && i.r === r;
}

// Interface equality: equal dynamic types and equal values. Pointer
// dynamic values compare by identity; struct values with a canonical
// key compare field-wise through it; every other object-carried
// dynamic value has no reviewed equality yet and fails closed with a
// stable diagnostic (Go panics only for genuinely uncomparable types).
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

// Interface-against-concrete equality through the canonical struct key.
export function goIfaceEqualKey(i: GoIface, r: GoRtti, v: { goKey$(): string }): boolean {
  return i !== undefined && i.r === r && (i.v as { goKey$(): string }).goKey$() === v.goKey$();
}

function ifaceValueEqual(r: GoRtti, a: unknown, b: unknown): boolean {
  if (a === b) {
    return true; // identity covers pointers and every equal primitive
  }
  if (typeof a !== "object" || a === null || typeof b !== "object" || b === null) {
    return false;
  }
  if (r.p === true) {
    return false; // distinct pointer identities
  }
  const keyed = a as { goKey$?: () => string };
  if (typeof keyed.goKey$ === "function") {
    return keyed.goKey$() === (b as { goKey$(): string }).goKey$();
  }
  throw new GoPanic("GOTOTS_UNIMPLEMENTED: interface equality over " + r.d);
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
export function goIfaceAssertIface(i: GoIface, methods: string[], sourceDisplay: string, targetDisplay: string): GoIfaceBox {
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
export function goIfaceLookupIface(i: GoIface, methods: string[]): [GoIface, boolean] {
  if (i === undefined || ifaceMissingMethod(i.r, methods) !== undefined) {
    return [undefined, false];
  }
  return [i, true];
}

function ifaceMissingMethod(rtti: GoRtti, methods: string[]): string | undefined {
  if (rtti.x !== undefined) {
    throw new GoPanic("GOTOTS_EXTERNAL_UNIMPLEMENTED: method set of " + rtti.x);
  }
  for (const method of methods) {
    if (rtti.m[method] === undefined) return method;
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
  return { d, m: {} };
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
