// The slice and fixed-array runtime: the exact Go slice carrier and the
// native-array value helpers (clone, in-place set, bounds-checked element
// access, equality, and array-backed slice views).
package abi

const gosliceSource = `// The exact Go slice carrier: backing array, offset, length, and
// capacity, with undefined as the nil slice. Aliasing, capacity reuse on
// append, reslicing, and Go's exact bounds panics are all preserved.
// Indices arrive as the static carrier of their Go type (number or
// bigint); JS mixed comparisons are exact, and conversion to an array
// offset happens only after the bounds proof.
import { GoPanic } from "./gopanic.js";
import { type GoCell } from "./goruntime.js";

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
  // Go's runtime omits the length for a negative index.
  if (index < 0) {
    throw new GoPanic("runtime error: index out of range [" + String(index) + "]");
  }
  throw new GoPanic("runtime error: index out of range [" + String(index) + "] with length " + String(length));
}

// goSliceElemCell is the aliasing cell over one slice slot: reads and
// writes through the cell ARE reads and writes of the slot, exactly
// &s[i] for value-carrier elements.
export function goSliceElemCell<T>(s: GoSliceValue<T>, index: GoIndex): GoCell<T> {
  const backing = s;
  const at = index;
  return {
    get v(): T {
      return goSliceGet(backing, at);
    },
    set v(value: T) {
      goSliceSet(backing, at, value);
    },
  };
}

// goSliceClearWith zeroes every element in place (Go's clear(s)):
// value-copy carriers store through their set operation so aliases of
// the elements observe the zeroing.
export function goSliceClearWith<T>(s: GoSliceValue<T>, zero: () => T, set: ((d: T, s: T) => void) | undefined): void {
  if (s === undefined) return;
  const length = Number(goSliceLen(s));
  for (let index = 0; index < length; index++) {
    if (set === undefined) {
      goSliceSet(s, BigInt(index), zero());
    } else {
      set(goSliceGet(s, BigInt(index)), zero());
    }
  }
}

export function goSliceFrom<T>(values: T[]): GoSlice<T> {
  return new GoSlice(values, 0, values.length, values.length);
}

export function goSliceMake<T>(length: GoIndex, capacity: GoIndex | undefined, zero: T): GoSlice<T> {
  if (capacity === undefined) capacity = length; // make([]T, n): length evaluated once
  goMakeCheck(length, capacity);
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
export function goSliceMakeStruct<T>(length: GoIndex, capacity: GoIndex | undefined, zero: () => T): GoSlice<T> {
  if (capacity === undefined) capacity = length; // make([]T, n): length evaluated once
  goMakeCheck(length, capacity);
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
// slice permits only [0:0]. An undefined high bound is len(s), computed
// from the single evaluation of the operand.
export function goSliceSlice<T>(s: GoSliceValue<T>, low: GoIndex, high: GoIndex | undefined): GoSliceValue<T> {
  if (high === undefined) high = s === undefined ? 0n : BigInt(s.length);
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

// s[low:high:max]: 0 <= low <= high <= max <= cap; shares backing storage
// but caps the result at max-low, so a later append reallocates once the
// limited capacity is exhausted instead of overwriting the operand's
// tail. high and max are both mandatory in Go's three-index form.
export function goSliceSlice3<T>(s: GoSliceValue<T>, low: GoIndex, high: GoIndex, max: GoIndex): GoSliceValue<T> {
  const capacity = s === undefined ? 0 : s.capacity;
  if (max > capacity || max < 0) {
    throw new GoPanic("runtime error: slice bounds out of range [::" + String(max) + "] with capacity " + String(capacity));
  }
  if (high > max || high < 0) {
    throw new GoPanic("runtime error: slice bounds out of range [:" + String(high) + ":" + String(max) + "]");
  }
  if (low < 0 || low > high) {
    throw new GoPanic("runtime error: slice bounds out of range [" + String(low) + ":" + String(high) + ":]");
  }
  if (s === undefined) {
    return undefined; // only [0:0:0] reaches here; nil stays nil
  }
  return new GoSlice(s.backing, s.offset + Number(low), Number(high) - Number(low), Number(max) - Number(low));
}

// append: returns the very slice value when nothing is appended (a nil
// slice stays nil); reuses backing storage when capacity allows
// (aliasing is observable); otherwise allocates, copying the old
// elements and zero-filling the region between the new length and the
// grown capacity so extended reslices read exact zeros. Go leaves the
// grown capacity implementation-defined; this carrier uses the
// documented doubling / 1.25x progression without allocator size-class
// rounding, and grown capacity is not a differential target.
export function goSliceAppend<T>(s: GoSliceValue<T>, values: T[], zero: T): GoSliceValue<T> {
  if (values.length === 0) return s;
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
  for (let index = needed; index < capacity; index++) {
    backing[index] = zero;
  }
  return new GoSlice(backing, 0, needed, capacity);
}

// The struct-element append overwrites reused-capacity slots in place
// (element aliases observe the store, exactly like Go's memory write),
// clones old elements into a grown backing (Go copies values, never
// shares them), and fills the grown tail with distinct fresh zeros. The
// incoming values are already fresh copies from their binding sites.
export function goSliceAppendStruct<T extends GoStructValue<T>>(s: GoSliceValue<T>, values: T[], zero: () => T): GoSliceValue<T> {
  if (values.length === 0) return s;
  const length = s === undefined ? 0 : s.length;
  const needed = length + values.length;
  if (s !== undefined && needed <= s.capacity) {
    for (let index = 0; index < values.length; index++) {
      const slot = s.backing[s.offset + length + index] as T | undefined;
      if (slot === undefined) {
        s.backing[s.offset + length + index] = values[index] as T;
      } else {
        slot.goSet$(values[index] as T);
      }
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
      backing[index] = (s.backing[s.offset + index] as T).goClone$();
    }
  }
  for (let index = 0; index < values.length; index++) {
    backing[length + index] = values[index] as T;
  }
  for (let index = needed; index < capacity; index++) {
    backing[index] = zero();
  }
  return new GoSlice(backing, 0, needed, capacity);
}

// append(s, source...): source's current elements are read first, so a
// self-append with capacity reuse stays exact; an empty source returns
// the very slice value.
export function goSliceAppendSlice<T>(s: GoSliceValue<T>, source: GoSliceValue<T>, zero: T): GoSliceValue<T> {
  const values: T[] = [];
  const length = source === undefined ? 0 : source.length;
  for (let index = 0; index < length; index++) {
    const src = source as GoSlice<T>;
    values.push(src.backing[src.offset + index] as T);
  }
  return goSliceAppend(s, values, zero);
}

// Fixed arrays are value types carried as native arrays. Copies happen
// at Go copy boundaries (cloneElem composes for struct and nested-array
// elements); whole-value stores overwrite elements in place so slices
// over the array and element aliases observe them; bounds panic with
// Go's exact message.
export function goArrayClone<T>(a: T[], cloneElem: ((v: T) => T) | undefined): T[] {
  if (cloneElem === undefined) {
    return a.slice();
  }
  const out: T[] = new Array(a.length);
  for (let index = 0; index < a.length; index++) {
    out[index] = cloneElem(a[index] as T);
  }
  return out;
}

export function goArraySetAll<T>(dst: T[], src: T[], setElem: ((d: T, s: T) => void) | undefined): void {
  for (let index = 0; index < dst.length; index++) {
    if (setElem === undefined) {
      dst[index] = src[index] as T;
    } else {
      setElem(dst[index] as T, src[index] as T);
    }
  }
}

// makeslice's exact panics: a negative length is "len out of range";
// a negative capacity or capacity below length is "cap out of range".
export function goMakeCheck(length: GoIndex, capacity: GoIndex): void {
  if (length < 0) {
    throw new GoPanic("runtime error: makeslice: len out of range");
  }
  if (capacity < 0 || capacity < length) {
    throw new GoPanic("runtime error: makeslice: cap out of range");
  }
}

// make([]T, len) for a native-array region: validated, filled with fresh
// zeros. Capacity equals length (native arrays carry no separate cap).
export function goNativeMakeLen<T>(length: GoIndex, zero: () => T): T[] {
  goMakeCheck(length, length);
  return goArrayZero(Number(length), zero);
}

// make([]T, len, cap) for a native-array region: capacity is validated
// and evaluated (its only observable effects) but not stored.
export function goNativeMakeCap<T>(length: GoIndex, capacity: GoIndex, zero: () => T): T[] {
  goMakeCheck(length, capacity);
  return goArrayZero(Number(length), zero);
}

export function goArrayZero<T>(length: number, makeElem: () => T): T[] {
  const out: T[] = new Array(length);
  for (let index = 0; index < length; index++) {
    out[index] = makeElem();
  }
  return out;
}

export function goArrayLen<T>(a: T[]): bigint {
  return BigInt(a.length);
}

export function goArrayGet<T>(a: T[], index: GoIndex): T {
  if (index < 0 || index >= a.length) panicIndex(index, a.length);
  return a[Number(index)] as T;
}

export function goArrayElemSet<T>(a: T[], index: GoIndex, value: T): void {
  if (index < 0 || index >= a.length) panicIndex(index, a.length);
  a[Number(index)] = value;
}

export function goArrayElemSetStruct<T extends GoStructValue<T>>(a: T[], index: GoIndex, value: T): void {
  if (index < 0 || index >= a.length) panicIndex(index, a.length);
  (a[Number(index)] as T).goSet$(value);
}

// s[i] = v for fixed-array elements: Go overwrites the element's
// memory, so the stored array's slots are copied in place and every
// alias of the element observes the store.
export function goSliceSetArray<T>(s: GoSliceValue<T[]>, index: GoIndex, value: T[], setElem: ((d: T, s: T) => void) | undefined): void {
  const length = s === undefined ? 0 : s.length;
  if (index < 0 || index >= length) panicIndex(index, length);
  const slice = s as GoSlice<T[]>;
  goArraySetAll(slice.backing[slice.offset + Number(index)] as T[], value, setElem);
}

// Element-wise equality in index order with a composed element
// comparison (nested structs, arrays, and interfaces).
export function goArrayEqualWith<T>(a: T[], b: T[], eq: (x: T, y: T) => boolean): boolean {
  for (let index = 0; index < a.length; index++) {
    if (!eq(a[index] as T, b[index] as T)) {
      return false;
    }
  }
  return true;
}

// Element-wise equality in index order for element carriers whose Go
// equality is exact identity/scalar equality.
export function goArrayEqual<T>(a: T[], b: T[]): boolean {
  for (let index = 0; index < a.length; index++) {
    if (a[index] !== b[index]) {
      return false;
    }
  }
  return true;
}

// arr[low:high] over an addressable array: a slice view sharing the
// array's storage; capacity runs to the array's end.
export function goSliceFromArray<T>(a: T[], low: GoIndex, high: GoIndex | undefined): GoSliceValue<T> {
  if (high === undefined) high = BigInt(a.length);
  if (high > a.length || high < 0) {
    throw new GoPanic("runtime error: slice bounds out of range [:" + String(high) + "] with capacity " + String(a.length));
  }
  if (low < 0 || low > high) {
    throw new GoPanic("runtime error: slice bounds out of range [" + String(low) + ":" + String(high) + "]");
  }
  return new GoSlice(a, Number(low), Number(high) - Number(low), a.length - Number(low));
}

// The struct-element spread clones each copied value: Go copies struct
// values into the destination's storage.
export function goSliceAppendSliceStruct<T extends GoStructValue<T>>(s: GoSliceValue<T>, source: GoSliceValue<T>, zero: () => T): GoSliceValue<T> {
  const values: T[] = [];
  const length = source === undefined ? 0 : source.length;
  for (let index = 0; index < length; index++) {
    const src = source as GoSlice<T>;
    values.push((src.backing[src.offset + index] as T).goClone$());
  }
  return goSliceAppendStruct(s, values, zero);
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
// ---- Type-parameter (factory-driven) element semantics. A generic body
// receives per-type-parameter factories: clone (total; identity for
// non-copy carriers) and set (undefined ⇢ the carrier stores by slot
// assignment; a function ⇢ the carrier overwrites in place so element
// aliases observe the store). These helpers are the single element-store
// and copy semantics for []T / [N]T under a type parameter T, exactly
// mirroring the struct forms above.

// s[i] = v for a type-parameter element.
export function goSliceSetWith<T>(s: GoSliceValue<T>, index: GoIndex, value: T, setElem: ((d: T, s: T) => void) | undefined): void {
  const length = s === undefined ? 0 : s.length;
  if (index < 0 || index >= length) panicIndex(index, length);
  const slice = s as GoSlice<T>;
  if (setElem === undefined) {
    slice.backing[slice.offset + Number(index)] = value;
  } else {
    setElem(slice.backing[slice.offset + Number(index)] as T, value);
  }
}

// a[i] = v for a type-parameter element of a fixed array.
export function goArrayElemSetWith<T>(a: T[], index: GoIndex, value: T, setElem: ((d: T, s: T) => void) | undefined): void {
  if (index < 0 || index >= a.length) panicIndex(index, a.length);
  if (setElem === undefined) {
    a[Number(index)] = value;
  } else {
    setElem(a[Number(index)] as T, value);
  }
}

// append(s, values...) for type-parameter elements: reused-capacity slots
// overwrite in place through setElem (element aliases observe the store),
// growth clones old elements through cloneElem (Go copies values), and
// the grown tail fills with distinct fresh zeros. The incoming values are
// already fresh copies from their binding sites.
export function goSliceAppendWith<T>(s: GoSliceValue<T>, values: T[], zero: () => T, cloneElem: (v: T) => T, setElem: ((d: T, s: T) => void) | undefined): GoSliceValue<T> {
  if (values.length === 0) return s;
  const length = s === undefined ? 0 : s.length;
  const needed = length + values.length;
  if (s !== undefined && needed <= s.capacity) {
    for (let index = 0; index < values.length; index++) {
      const at = s.offset + length + index;
      const slot = s.backing[at] as T | undefined;
      if (setElem === undefined || slot === undefined) {
        s.backing[at] = values[index] as T;
      } else {
        setElem(slot, values[index] as T);
      }
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
      backing[index] = cloneElem(s.backing[s.offset + index] as T);
    }
  }
  for (let index = 0; index < values.length; index++) {
    backing[length + index] = values[index] as T;
  }
  for (let index = needed; index < capacity; index++) {
    backing[index] = zero();
  }
  return new GoSlice(backing, 0, needed, capacity);
}

// append(s, source...) for type-parameter elements: source's current
// elements are staged as copies first, so a self-append with capacity
// reuse stays exact.
export function goSliceAppendSliceWith<T>(s: GoSliceValue<T>, source: GoSliceValue<T>, zero: () => T, cloneElem: (v: T) => T, setElem: ((d: T, s: T) => void) | undefined): GoSliceValue<T> {
  const values: T[] = [];
  const length = source === undefined ? 0 : source.length;
  for (let index = 0; index < length; index++) {
    const src = source as GoSlice<T>;
    values.push(cloneElem(src.backing[src.offset + index] as T));
  }
  return goSliceAppendWith(s, values, zero, cloneElem, setElem);
}

// copy(dst, src) for type-parameter elements: overlapping ranges stage
// copies first (Go's memmove semantics), then store through the element
// rule.
export function goSliceCopyWith<T>(dst: GoSliceValue<T>, src: GoSliceValue<T>, cloneElem: (v: T) => T, setElem: ((d: T, s: T) => void) | undefined): bigint {
  const dstLength = dst === undefined ? 0 : dst.length;
  const srcLength = src === undefined ? 0 : src.length;
  const count = dstLength < srcLength ? dstLength : srcLength;
  const staged: T[] = [];
  for (let index = 0; index < count; index++) {
    const source = src as GoSlice<T>;
    staged.push(cloneElem(source.backing[source.offset + index] as T));
  }
  for (let index = 0; index < count; index++) {
    const destination = dst as GoSlice<T>;
    if (setElem === undefined) {
      destination.backing[destination.offset + index] = staged[index] as T;
    } else {
      setElem(destination.backing[destination.offset + index] as T, staged[index] as T);
    }
  }
  return BigInt(count);
}
`
