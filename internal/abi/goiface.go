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
// names qualify, as in runtime messages) and its method table mapping
// method names onto the generated method functions.
export interface GoRtti {
  readonly d: string;
  readonly m: Readonly<Record<string, Function>>;
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
  const fn = boxed.r.m[method] as Function;
  return fn(boxed.v, ...args);
}

export function goIfaceIs(i: GoIface, r: GoRtti): boolean {
  return i !== undefined && i.r === r;
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
