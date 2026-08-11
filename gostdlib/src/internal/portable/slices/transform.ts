import { GoPanic } from "@gotots/runtime/panic.js";
import type { Awaitable, bool, int64 } from "@gotots/gostdlib/internal/scalars.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import { hostInteger } from "../../host-integer.js";

import { sliceValues } from "../../runtime/slice.js";
import {
  callPredicate,
  callPredicateSynchronous,
} from "./read.js";
import {
  type Convert,
  type CopyValue,
  type EqualValue,
  type FromContainerStorage,
  readElement,
  storeElement,
  type ToContainerStorage,
  type Zero,
} from "./capabilities.js";

type Predicate<T> = ((value: T) => Awaitable<bool>) | undefined;
type Equality<T> = ((left: T, right: T) => Awaitable<bool>) | undefined;
type SynchronousPredicate<T> = ((value: T) => bool) | undefined;
type SynchronousEquality<T> = ((left: T, right: T) => bool) | undefined;

export function Clip<T>(source: RuntimeSlice<T>): RuntimeSlice<T> {
  return source.slice(0, source.length, source.length);
}

export function Clone<T>(source: RuntimeSlice<T>): RuntimeSlice<T> {
  return source.isNil()
    ? RuntimeSlice.nil<T>()
    : RuntimeSlice.literal(sliceValues(source));
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
  return compactSynchronous(
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

export async function CompactFunc<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  fromSlice: Convert<RuntimeSlice<EStorage>, S>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  zeroElement: Zero<E>,
  source: S,
  equal: Equality<E>,
): Promise<S> {
  const values = toSlice(source);
  if (values.length < 2) {
    return source;
  }
  if (equal === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  for (let duplicate = 1; duplicate < values.length; duplicate += 1) {
    if (await equal(
      readElement(values, duplicate, copyElement, fromStorage),
      readElement(values, duplicate - 1, copyElement, fromStorage),
    )) {
      let write = duplicate;
      for (let read = duplicate + 1; read < values.length; read += 1) {
        if (!await equal(
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

export function CompactFuncSynchronous<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  fromSlice: Convert<RuntimeSlice<EStorage>, S>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  zeroElement: Zero<E>,
  source: S,
  equal: SynchronousEquality<E>,
): S {
  const values = toSlice(source);
  if (values.length < 2) {
    return source;
  }
  if (equal === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return compactSynchronous(
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

export function Concat<T>(
  sources: RuntimeSlice<RuntimeSlice<T>>,
): RuntimeSlice<T> {
  const result: T[] = [];
  for (let sourceIndex = 0; sourceIndex < sources.length; sourceIndex += 1) {
    const source = sources.get(sourceIndex);
    for (let valueIndex = 0; valueIndex < source.length; valueIndex += 1) {
      result.push(source.get(valueIndex));
    }
  }
  return result.length === 0
    ? RuntimeSlice.nil<T>()
    : RuntimeSlice.literal(result);
}

export function Delete<T>(
  source: RuntimeSlice<T>,
  start: int64,
  end: int64,
): RuntimeSlice<T> {
  const startIndex = hostInteger(start);
  const endIndex = hostInteger(end);
  validateRange(source.length, startIndex, endIndex);
  if (start === end) {
    return source;
  }
  const values = sliceValues(source);
  values.splice(startIndex, endIndex - startIndex);
  return resultLike(source, values);
}

export async function DeleteFunc<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  fromSlice: Convert<RuntimeSlice<EStorage>, S>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  zeroElement: Zero<E>,
  source: S,
  predicate: Predicate<E>,
): Promise<S> {
  const values = toSlice(source);
  let write = 0;
  for (let read = 0; read < values.length; read += 1) {
    const value = fromStorage(values.get(read));
    if (!await callPredicate(predicate, value)) {
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

export function DeleteFuncSynchronous<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  fromSlice: Convert<RuntimeSlice<EStorage>, S>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  zeroElement: Zero<E>,
  source: S,
  predicate: SynchronousPredicate<E>,
): S {
  const values = toSlice(source);
  let write = 0;
  for (let read = 0; read < values.length; read += 1) {
    const value = fromStorage(values.get(read));
    if (!callPredicateSynchronous(predicate, value)) {
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

export function Insert<T>(
  source: RuntimeSlice<T>,
  index: int64,
  values: RuntimeSlice<T>,
): RuntimeSlice<T> {
  const hostIndex = hostInteger(index);
  validateIndex(source.length, hostIndex, true);
  if (values.length === 0) {
    return source;
  }
  const result = sliceValues(source);
  result.splice(hostIndex, 0, ...sliceValues(values));
  return RuntimeSlice.literal(result);
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

export function Repeat<T>(
  source: RuntimeSlice<T>,
  count: int64,
): RuntimeSlice<T> {
  if (count < 0n) {
    GoPanic.raiseRuntime("the result of (len(x) * count) overflows");
  }
  const hostCount = hostInteger(count);
  const resultLength = source.length * hostCount;
  if (!Number.isSafeInteger(resultLength)) {
    GoPanic.raiseRuntime("the result of (len(x) * count) overflows");
  }
  const values: T[] = [];
  for (let repetition = 0; repetition < hostCount; repetition += 1) {
    values.push(...sliceValues(source));
  }
  return RuntimeSlice.literal(values);
}

export function Replace<T>(
  source: RuntimeSlice<T>,
  start: int64,
  end: int64,
  replacement: RuntimeSlice<T>,
): RuntimeSlice<T> {
  const startIndex = hostInteger(start);
  const endIndex = hostInteger(end);
  validateRange(source.length, startIndex, endIndex);
  const values = sliceValues(source);
  values.splice(startIndex, endIndex - startIndex, ...sliceValues(replacement));
  return resultLike(source, values);
}

export function Reverse<T>(source: RuntimeSlice<T>): void {
  for (
    let left = 0, right = source.length - 1;
    left < right;
    left += 1, right -= 1
  ) {
    const leftValue = source.get(left);
    source.set(left, source.get(right));
    source.set(right, leftValue);
  }
}

function compactSynchronous<S, E, EStorage>(
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

function clearTail<E, EStorage>(
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

function resultLike<T>(
  source: RuntimeSlice<T>,
  values: T[],
): RuntimeSlice<T> {
  if (values.length === 0 && source.isNil()) {
    return RuntimeSlice.nil<T>();
  }
  return RuntimeSlice.literal(values);
}

function validateRange(length: number, start: number, end: number): void {
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

function validateIndex(length: number, index: number, allowEnd: boolean): void {
  const upper = allowEnd ? length : length - 1;
  if (!Number.isSafeInteger(index) || index < 0 || index > upper) {
    GoPanic.raiseRuntime("slice bounds out of range");
  }
}
