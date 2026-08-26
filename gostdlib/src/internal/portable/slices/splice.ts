import { GoPanic } from "@gotots/runtime/panic.js";
import type { int64 } from "@gotots/gostdlib/internal/scalars.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import { hostInteger } from "../../host-integer.js";
import {
  type Convert,
  type CopyValue,
  type FromContainerStorage,
  readElement,
  storeElement,
  type ToContainerStorage,
  type Zero,
} from "./capabilities.js";
import {
  allocateSlice,
  clearTail,
  copyLogicalRange,
  growthCapacity,
  logicalValues,
  validateIndex,
  validateRange,
} from "./storage.js";

export function Insert<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  fromSlice: Convert<RuntimeSlice<EStorage>, S>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  zeroElement: Zero<E>,
  source: S,
  index: int64,
  inserted: RuntimeSlice<EStorage>,
): S {
  const values = toSlice(source);
  const hostIndex = hostInteger(index);
  validateIndex(values.length, hostIndex, true);
  if (inserted.length === 0) {
    return source;
  }
  const insertedValues = logicalValues(
    inserted,
    copyElement,
    fromStorage,
  );
  const nextLength = values.length + insertedValues.length;
  if (!Number.isSafeInteger(nextLength)) {
    GoPanic.raiseRuntime("len out of range");
  }
  const allocated = nextLength > values.capacity;
  const result = allocated
    ? allocateSlice(
      nextLength,
      growthCapacity(values.capacity, nextLength),
      toStorage,
      zeroElement,
    )
    : values.slice(0, nextLength, null);
  if (allocated) {
    copyLogicalRange(
      values,
      0,
      hostIndex,
      result,
      0,
      copyElement,
      fromStorage,
      toStorage,
    );
    copyLogicalRange(
      values,
      hostIndex,
      values.length,
      result,
      hostIndex + insertedValues.length,
      copyElement,
      fromStorage,
      toStorage,
    );
  } else {
    for (let read = values.length - 1; read >= hostIndex; read -= 1) {
      storeElement(
        result,
        read + insertedValues.length,
        readElement(values, read, copyElement, fromStorage),
        copyElement,
        toStorage,
      );
    }
  }
  for (const [offset, value] of insertedValues.entries()) {
    storeElement(result, hostIndex + offset, value, copyElement, toStorage);
  }
  return fromSlice(result);
}

export function Replace<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  fromSlice: Convert<RuntimeSlice<EStorage>, S>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  zeroElement: Zero<E>,
  source: S,
  start: int64,
  end: int64,
  replacement: RuntimeSlice<EStorage>,
): S {
  const values = toSlice(source);
  const startIndex = hostInteger(start);
  const endIndex = hostInteger(end);
  validateRange(values.length, startIndex, endIndex);
  const replacementValues = logicalValues(
    replacement,
    copyElement,
    fromStorage,
  );
  if (startIndex === endIndex && replacementValues.length === 0) {
    return source;
  }
  const removed = endIndex - startIndex;
  const delta = replacementValues.length - removed;
  const nextLength = values.length + delta;
  if (!Number.isSafeInteger(nextLength)) {
    GoPanic.raiseRuntime("len out of range");
  }
  const allocated = nextLength > values.capacity;
  const result = allocated
    ? allocateSlice(
      nextLength,
      growthCapacity(values.capacity, nextLength),
      toStorage,
      zeroElement,
    )
    : values.slice(0, nextLength, null);
  if (allocated) {
    copyLogicalRange(
      values,
      0,
      startIndex,
      result,
      0,
      copyElement,
      fromStorage,
      toStorage,
    );
    copyLogicalRange(
      values,
      endIndex,
      values.length,
      result,
      startIndex + replacementValues.length,
      copyElement,
      fromStorage,
      toStorage,
    );
  } else if (delta > 0) {
    for (let read = values.length - 1; read >= endIndex; read -= 1) {
      storeElement(
        result,
        read + delta,
        readElement(values, read, copyElement, fromStorage),
        copyElement,
        toStorage,
      );
    }
  } else if (delta < 0) {
    for (let read = endIndex; read < values.length; read += 1) {
      storeElement(
        result,
        read + delta,
        readElement(values, read, copyElement, fromStorage),
        copyElement,
        toStorage,
      );
    }
  }
  for (const [offset, value] of replacementValues.entries()) {
    storeElement(result, startIndex + offset, value, copyElement, toStorage);
  }
  if (nextLength < values.length) {
    clearTail(values, nextLength, copyElement, toStorage, zeroElement);
  }
  return fromSlice(result);
}
