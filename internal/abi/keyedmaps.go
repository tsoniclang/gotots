// The keyed/encoded map ABI sources (struct- and interface-keyed maps).
package abi

const keyedMapsSource = `// Encoded-key maps carry Go map semantics for INTERFACE keys: the key
// encodes through the union's generated $key function — per-member exact
// (pointer identity, scalar values, struct goKey$ composition), a fresh
// unmatchable token per NaN encode (Go's NaN keys are unretrievable), and
// Go's exact "hash of unhashable type" panic for uncomparable dynamic
// members. The carrier stores the original key beside the value so range
// yields it.
export type GoEMap<K, V> = Map<string, [K, V]> | undefined;

export function goEMapMake<K, V>(hint?: unknown): Map<string, [K, V]> {
  void hint;
  return new Map<string, [K, V]>();
}

export function goEMapFrom<K, V>(entries: readonly (readonly [K, V])[], encode: (k: K) => string): Map<string, [K, V]> {
  const out = new Map<string, [K, V]>();
  for (const [key, value] of entries) {
    out.set(encode(key), [key, value]);
  }
  return out;
}

export function goEMapGet<K, V>(m: GoEMap<K, V>, key: K, zero: V, encode: (k: K) => string): V {
  if (m === undefined) return zero;
  const entry = m.get(encode(key));
  return entry === undefined ? zero : entry[1];
}

export function goEMapLookup<K, V>(m: GoEMap<K, V>, key: K, zero: V, encode: (k: K) => string): readonly [V, boolean] {
  if (m === undefined) return [zero, false];
  const entry = m.get(encode(key));
  return entry === undefined ? [zero, false] : [entry[1], true];
}

export function goEMapSet<K, V>(m: GoEMap<K, V>, key: K, value: V, encode: (k: K) => string): void {
  if (m === undefined) goPanicNilMapWrite();
  m.set(encode(key), [key, value]);
}

export function goEMapDelete<K, V>(m: GoEMap<K, V>, key: K, encode: (k: K) => string): void {
  if (m === undefined) return;
  m.delete(encode(key));
}

export function goEMapLen<K, V>(m: GoEMap<K, V>): bigint {
  return m === undefined ? 0n : BigInt(m.size);
}

export function goEMapValues<K, V>(m: GoEMap<K, V>): readonly (readonly [K, V])[] {
  if (m === undefined) return [];
  return Array.from(m.values());
}

export function goEMapClear<K, V>(m: GoEMap<K, V>): void {
  if (m === undefined) return;
  m.clear();
}

// goKeyFloat encodes one float key component with Go's exact float-key
// equality: -0 folds onto +0; every NaN encode yields a FRESH token no
// later encode matches (unretrievable, undeletable, still counted).
let nanKeyCounter = 0;
export function goKeyFloat(v: number): string {
  if (Number.isNaN(v)) {
    nanKeyCounter++;
    return "fn#" + String(nanKeyCounter);
  }
  if (v === 0) return "f0";
  return "f" + String(v);
}

// Typed key-component encoders: one per statically known scalar
// carrier — strings length-prefixed (injective under any composition),
// integers by canonical token, booleans by t/f. Floats go through
// goKeyFloat's exact NaN/signed-zero rule.
export function goKeyString(v: string): string {
  return "s" + String(v.length) + ":" + v;
}
export function goKeyInt(v: bigint | number): string {
  return "i" + String(v);
}
export function goKeyBool(v: boolean): string {
  return v ? "t" : "f";
}

// goKeyUnreachable is the machine-claimed invariant stop of a key
// operation the admission chain proves uncallable (every keyed use of
// the binding is statically rejected before emission): reaching it is a
// compiler-invariant break, never Go semantics. Plain Error, never a Go
// panic surface.
export function goKeyUnreachable(display: string): never {
  throw new Error("gotots invariant: statically unreachable key operation for " + display);
}

// goKeyUnhashable is Go's exact runtime panic for an uncomparable dynamic
// map key.
export function goKeyUnhashable(display: string): never {
  throw new GoPanic("runtime error: hash of unhashable type " + display);
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

export function goKMapValues<K extends GoKeyed, V>(m: GoKeyedMap<K, V>): readonly (readonly [K, V])[] {
  if (m === undefined) return [];
  return Array.from(m.values());
}

export function goKMapClear<K extends GoKeyed, V>(m: GoKeyedMap<K, V>): void {
  if (m !== undefined) m.clear();
}
`
