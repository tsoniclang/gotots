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

// IfaceToken is the diagnostic-facing shape of an interface box: the rtti
// token r. This shape is used ONLY by the assertion-panic
// diagnostic below, which reads the token's display and method-name data;
// it never touches a payload, so no interface value is ever widened to an
// erased payload type. Interface equality and dispatch are generated
// inline per exact union member (per-member narrowing), not through any
// payload-erasing helper.
type IfaceToken = { readonly r: GoRtti };

// goPanicConversionIface is Go's failed interface-to-interface
// assertion panic: the dynamic type first, then the first missing
// method (from the token's method-name data).
export function goPanicConversionIface(i: IfaceToken | undefined, sourceDisplay: string, targetDisplay: string, required: readonly string[]): never {
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

// goPanicUncomparable is Go's panic when two interface values share an
// uncomparable dynamic type: the exact runtime message, the type display
// read from the box's own rtti token (never an erased payload).
export function goPanicUncomparable(display: string): never {
  throw new GoPanic("runtime error: comparing uncomparable type " + display);
}

// goPanicExternalEq fails closed when an external value's comparability is
// unknown (its fields are unreviewed): equality is not guessed.
export function goPanicExternalEq(display: string): never {
  throw new GoPanic("GOTOTS_EXTERNAL_UNIMPLEMENTED: equality of " + display);
}


// Interface equality (a == b) and interface-against-concrete equality
// emit INLINE at the use site as a per-union equality function / literal
// narrowing that touches only exact member payloads (=== for pointers and
// primitive carriers, the type's own goEq$ for comparable value structs,
// element-wise for comparable value arrays, an exact runtime panic for an
// uncomparable dynamic type, and fail-closed for an external value): no
// payload is ever recovered through an erased type, so no equality helper
// over an unknown payload exists.

// Assertions (x.(T), both panic and comma-ok forms) emit INLINE at the
// use site as literal-discriminant narrowing that reads the exact member
// payload with no cast (ADR-0004); no payload-recovering helper exists.

function rtti(d: string): GoRtti {
  // A predeclared basic type's rtti is a pure identity+display token;
  // it is methodless, and its interface equality is generated inline.
  return { d, ms: [] };
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
