import type { bool, int64 } from "@gotots/gostdlib/internal/scalars.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import { integerFromHost } from "../../host-integer.js";

export type BinaryLess<T> = (left: T, right: T) => bool;
export type Convert<Source, Target> = (value: Source) => Target;
export type CopyValue<T> = (value: T) => T;
export type EqualValue<Left, Right = Left> = (
  left: Left,
  right: Right,
) => bool;
export type FromContainerStorage<T, Storage> = (value: Storage) => T;
export type ToContainerStorage<T, Storage> = (value: T) => Storage;
export type Zero<T> = () => T;

export function readElement<T, Storage>(
  source: RuntimeSlice<Storage>,
  index: int64,
  copy: CopyValue<T>,
  fromStorage: FromContainerStorage<T, Storage>,
): T {
  return copy(fromStorage(source.get(index)));
}

export function storeElement<T, Storage>(
  target: RuntimeSlice<Storage>,
  index: int64,
  value: T,
  copy: CopyValue<T>,
  toStorage: ToContainerStorage<T, Storage>,
): void {
  target.set(index, toStorage(copy(value)));
}

export function storedValues<T, Storage>(
  source: RuntimeSlice<Storage>,
  copy: CopyValue<T>,
  fromStorage: FromContainerStorage<T, Storage>,
  toStorage: ToContainerStorage<T, Storage>,
): Storage[] {
  const result: Storage[] = [];
  for (let index = 0; index < source.length; index += 1) {
    result.push(toStorage(readElement(
      source,
      integerFromHost(index),
      copy,
      fromStorage,
    )));
  }
  return result;
}

export function orderedCompare<T>(
  less: BinaryLess<T>,
  equal: EqualValue<T>,
  left: T,
  right: T,
): int64 {
  const leftNaN = !equal(left, left);
  const rightNaN = !equal(right, right);
  if (leftNaN) {
    return rightNaN ? 0n : -1n;
  }
  if (rightNaN) {
    return 1n;
  }
  if (less(left, right)) {
    return -1n;
  }
  if (less(right, left)) {
    return 1n;
  }
  return 0n;
}

export function compareLengths(left: number, right: number): int64 {
  if (left < right) {
    return -1n;
  }
  if (left > right) {
    return 1n;
  }
  return 0n;
}
