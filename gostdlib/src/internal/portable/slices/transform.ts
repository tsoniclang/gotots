import { GoPanic } from "@gotots/runtime/panic.js";
import type { bool, int64 } from "@gotots/gostdlib/internal/scalars.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import { hostInteger } from "../../host-integer.js";

import {
  type Convert,
  type CopyValue,
  type EqualValue,
  type FromContainerStorage,
  readElement,
  storedValues,
  storeElement,
  type ToContainerStorage,
  type Zero,
} from "./capabilities.js";
import {
  allocateSlice,
  clearTail,
  growthCapacity,
  validateRange,
} from "./storage.js";

type Predicate<T> = ((value: T) => bool) | undefined;
type Equality<T> = ((left: T, right: T) => bool) | undefined;

export function Clip<T>(source: RuntimeSlice<T>): RuntimeSlice<T> {
  return source.slice(0, source.length, source.length);
}

export function Clone<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  fromSlice: Convert<RuntimeSlice<EStorage>, S>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  source: S,
): S {
  const values = toSlice(source);
  return values.isNil()
    ? source
    : fromSlice(RuntimeSlice.literal(storedValues(
      values,
      copyElement,
      fromStorage,
      toStorage,
    )));
}

export function Compact<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  fromSlice: Convert<RuntimeSlice<EStorage>, S>,
  copyElement: CopyValue<E>,
  equal: EqualValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  zeroElement: Zero<E>,
  source: S,
): S {
  const values = toSlice(source);
  return compact(
    fromSlice,
    copyElement,
    equal,
    fromStorage,
    toStorage,
    zeroElement,
    source,
    values,
  );
}

export function CompactFunc<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  fromSlice: Convert<RuntimeSlice<EStorage>, S>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  zeroElement: Zero<E>,
  source: S,
  equal: Equality<E>,
): S {
  const values = toSlice(source);
  if (values.length < 2) {
    return source;
  }
  if (equal === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return compact(
    fromSlice,
    copyElement,
    equal,
    fromStorage,
    toStorage,
    zeroElement,
    source,
    values,
  );
}

export function Concat<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  fromSlice: Convert<RuntimeSlice<EStorage>, S>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  zeroElement: Zero<E>,
  sources: RuntimeSlice<S>,
): S {
  let size = 0;
  for (let sourceIndex = 0; sourceIndex < sources.length; sourceIndex += 1) {
    size += toSlice(sources.get(sourceIndex)).length;
    if (!Number.isSafeInteger(size)) {
      GoPanic.raiseRuntime("len out of range");
    }
  }
  if (size === 0) {
    return fromSlice(RuntimeSlice.nil<EStorage>());
  }
  const result = allocateSlice(
    size,
    growthCapacity(0, size),
    toStorage,
    zeroElement,
  );
  let write = 0;
  for (let sourceIndex = 0; sourceIndex < sources.length; sourceIndex += 1) {
    const source = toSlice(sources.get(sourceIndex));
    for (let read = 0; read < source.length; read += 1) {
      storeElement(
        result,
        write,
        readElement(source, read, copyElement, fromStorage),
        copyElement,
        toStorage,
      );
      write += 1;
    }
  }
  return fromSlice(result);
}

export function Delete<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  fromSlice: Convert<RuntimeSlice<EStorage>, S>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  zeroElement: Zero<E>,
  source: S,
  start: int64,
  end: int64,
): S {
  const values = toSlice(source);
  const startIndex = hostInteger(start);
  const endIndex = hostInteger(end);
  validateRange(values.length, startIndex, endIndex);
  if (start === end) {
    return source;
  }
  const removed = endIndex - startIndex;
  for (let read = endIndex; read < values.length; read += 1) {
    storeElement(
      values,
      read - removed,
      readElement(values, read, copyElement, fromStorage),
      copyElement,
      toStorage,
    );
  }
  const nextLength = values.length - removed;
  clearTail(values, nextLength, copyElement, toStorage, zeroElement);
  return fromSlice(values.slice(0, nextLength, null));
}

export function DeleteFunc<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  fromSlice: Convert<RuntimeSlice<EStorage>, S>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  zeroElement: Zero<E>,
  source: S,
  predicate: Predicate<E>,
): S {
  if (predicate === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  const values = toSlice(source);
  let write = 0;
  for (let read = 0; read < values.length; read += 1) {
    const value = fromStorage(values.get(read));
    if (!predicate(value)) {
      storeElement(
        values,
        write,
        value,
        copyElement,
        toStorage,
      );
      write += 1;
    }
  }
  for (let index = write; index < values.length; index += 1) {
    storeElement(
      values,
      index,
      zeroElement(),
      copyElement,
      toStorage,
    );
  }
  return fromSlice(values.slice(0, write, null));
}

export function Grow<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  fromSlice: Convert<RuntimeSlice<EStorage>, S>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  zeroElement: Zero<E>,
  source: S,
  amount: int64,
): S {
  const values = toSlice(source);
  if (amount < 0n) {
    GoPanic.raiseRuntime("cannot be negative");
  }
  const numericAmount = hostInteger(amount);
  const requiredCapacity = values.length + numericAmount;
  if (!Number.isSafeInteger(requiredCapacity)) {
    GoPanic.raiseRuntime("cannot be negative");
  }
  if (requiredCapacity <= values.capacity) {
    return source;
  }
  let nextCapacity = values.capacity === 0 ? 1 : values.capacity * 2;
  while (nextCapacity < requiredCapacity) {
    nextCapacity *= 2;
  }
  if (!Number.isSafeInteger(nextCapacity)) {
    GoPanic.raiseRuntime("cannot be negative");
  }
  const result = RuntimeSlice.make<EStorage>(
    nextCapacity,
    nextCapacity,
    toStorage(zeroElement()),
  );
  for (let index = 0; index < nextCapacity; index += 1) {
    result.set(index, toStorage(zeroElement()));
  }
  for (let index = 0; index < values.length; index += 1) {
    storeElement(
      result,
      index,
      fromStorage(values.get(index)),
      copyElement,
      toStorage,
    );
  }
  return fromSlice(result.slice(0, values.length, null));
}

export function Repeat<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  fromSlice: Convert<RuntimeSlice<EStorage>, S>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  source: S,
  count: int64,
): S {
  if (count < 0n) {
    GoPanic.raiseRuntime("cannot be negative");
  }
  const hostCount = hostInteger(count);
  const values = toSlice(source);
  const resultLength = values.length * hostCount;
  if (!Number.isSafeInteger(resultLength)) {
    GoPanic.raiseRuntime("the result of (len(x) * count) overflows");
  }
  const result: EStorage[] = [];
  for (let repetition = 0; repetition < hostCount; repetition += 1) {
    result.push(...storedValues(
      values,
      copyElement,
      fromStorage,
      toStorage,
    ));
  }
  return fromSlice(RuntimeSlice.literal(result));
}

export function Reverse<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  source: S,
): void {
  const values = toSlice(source);
  for (
    let left = 0, right = values.length - 1;
    left < right;
    left += 1, right -= 1
  ) {
    const leftValue = readElement(values, left, copyElement, fromStorage);
    const rightValue = readElement(values, right, copyElement, fromStorage);
    storeElement(values, left, rightValue, copyElement, toStorage);
    storeElement(values, right, leftValue, copyElement, toStorage);
  }
}

function compact<S, E, EStorage>(
  fromSlice: Convert<RuntimeSlice<EStorage>, S>,
  copyElement: CopyValue<E>,
  equal: EqualValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  zeroElement: Zero<E>,
  source: S,
  values: RuntimeSlice<EStorage>,
): S {
  if (values.length < 2) {
    return source;
  }
  for (let duplicate = 1; duplicate < values.length; duplicate += 1) {
    if (equal(
      readElement(values, duplicate, copyElement, fromStorage),
      readElement(values, duplicate - 1, copyElement, fromStorage),
    )) {
      let write = duplicate;
      for (let read = duplicate + 1; read < values.length; read += 1) {
        if (!equal(
          readElement(values, read, copyElement, fromStorage),
          readElement(values, read - 1, copyElement, fromStorage),
        )) {
          storeElement(
            values,
            write,
            readElement(values, read, copyElement, fromStorage),
            copyElement,
            toStorage,
          );
          write += 1;
        }
      }
      clearTail(values, write, copyElement, toStorage, zeroElement);
      return fromSlice(values.slice(0, write, null));
    }
  }
  return source;
}
