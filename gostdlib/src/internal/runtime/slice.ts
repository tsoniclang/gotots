import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring, uint8 } from "@gotots/gostdlib/internal/scalars.js";

export function sliceValues<T>(source: RuntimeSlice<T>): T[] {
  const values: T[] = [];
  for (let index = 0; index < source.length; index += 1) {
    values.push(source.get(index));
  }
  return values;
}

export function stringSlice(values: readonly string[]): RuntimeSlice<gostring> {
  return RuntimeSlice.literal([...values]);
}

export function byteSlice(values: Uint8Array | readonly number[]): RuntimeSlice<uint8> {
  return RuntimeSlice.literal(Array.from(values));
}

export function bytes(source: RuntimeSlice<uint8>): Uint8Array {
  return Uint8Array.from(sliceValues(source));
}

export function writeBytes(target: RuntimeSlice<uint8>, source: Uint8Array | readonly number[]): number {
  const count = Math.min(target.length, source.length);
  for (let index = 0; index < count; index += 1) {
    target.set(index, source[index] ?? 0);
  }
  return count;
}
