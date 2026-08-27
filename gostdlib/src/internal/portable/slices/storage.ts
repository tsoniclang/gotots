import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import {
  type CopyValue,
  type FromContainerStorage,
  readElement,
  storeElement,
  type ToContainerStorage,
  type Zero,
} from "./capabilities.js";

export function clearTail<E, EStorage>(
  values: RuntimeSlice<EStorage>,
  start: number,
  copyElement: CopyValue<E>,
  toStorage: ToContainerStorage<E, EStorage>,
  zeroElement: Zero<E>,
): void {
  for (let index = start; index < values.length; index += 1) {
    storeElement(
      values,
      index,
      zeroElement(),
      copyElement,
      toStorage,
    );
  }
}

export function logicalValues<E, EStorage>(
  source: RuntimeSlice<EStorage>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
): E[] {
  const result: E[] = [];
  for (let index = 0; index < source.length; index += 1) {
    result.push(readElement(source, index, copyElement, fromStorage));
  }
  return result;
}

export function allocateSlice<E, EStorage>(
  length: number,
  capacity: number,
  toStorage: ToContainerStorage<E, EStorage>,
  zeroElement: Zero<E>,
): RuntimeSlice<EStorage> {
  const result = RuntimeSlice.make<EStorage>(
    capacity,
    capacity,
    toStorage(zeroElement()),
  );
  for (let index = 0; index < capacity; index += 1) {
    result.set(index, toStorage(zeroElement()));
  }
  return result.slice(0, length, null);
}

export function growthCapacity(current: number, required: number): number {
  let capacity = current === 0 ? 1 : current * 2;
  while (capacity < required) {
    capacity *= 2;
  }
  if (!Number.isSafeInteger(capacity)) {
    GoPanic.raiseRuntime("len out of range");
  }
  return capacity;
}

export function copyLogicalRange<E, EStorage>(
  source: RuntimeSlice<EStorage>,
  start: number,
  end: number,
  target: RuntimeSlice<EStorage>,
  targetStart: number,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
): void {
  for (let read = start; read < end; read += 1) {
    storeElement(
      target,
      targetStart + read - start,
      readElement(source, read, copyElement, fromStorage),
      copyElement,
      toStorage,
    );
  }
}

export function validateRange(
  length: number,
  start: number,
  end: number,
): void {
  if (
    !Number.isSafeInteger(start)
    || !Number.isSafeInteger(end)
    || start < 0
    || end < start
    || end > length
  ) {
    GoPanic.raiseRuntime("slice bounds out of range");
  }
}

export function validateIndex(
  length: number,
  index: number,
  allowEnd: boolean,
): void {
  const upper = allowEnd ? length : length - 1;
  if (!Number.isSafeInteger(index) || index < 0 || index > upper) {
    GoPanic.raiseRuntime("slice bounds out of range");
  }
}
