import type { Awaitable, int64 } from "@gotots/gostdlib/internal/scalars.js";
import { GoDenseIndex } from "@gotots/runtime/dense-index.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import {
  hostInteger,
  integerFromHost,
} from "../../host-integer.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import {
  type BinaryLess,
  type Convert,
  type CopyValue,
  type EqualValue,
  type FromContainerStorage,
  orderedCompare,
  readElement,
  storeElement,
  type ToContainerStorage,
} from "./capabilities.js";
import { callComparison } from "./read.js";

type Comparison<T> = ((left: T, right: T) => Awaitable<int64>) | undefined;
type SynchronousComparison<T> = ((left: T, right: T) => int64) | undefined;

export function Sort<S, E, EStorage>(
  less: BinaryLess<E>,
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  equal: EqualValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  source: S,
): void {
  const values = toSlice(source);
  writeSorted(
    values,
    logicalValues(values, copyElement, fromStorage).sort(
      (left, right): number => hostInteger(orderedCompare(less, equal, left, right)),
    ),
    copyElement,
    toStorage,
  );
}

export async function SortFunc<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  source: S,
  compare: Comparison<E>,
): Promise<void> {
  const values = toSlice(source);
  writeSorted(
    values,
    await sortValues(
      logicalValues(values, copyElement, fromStorage),
      compare,
    ),
    copyElement,
    toStorage,
  );
}

export async function SortStableFunc<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  source: S,
  compare: Comparison<E>,
): Promise<void> {
  await SortFunc(
    toSlice,
    copyElement,
    fromStorage,
    toStorage,
    source,
    compare,
  );
}

export function SortFuncSynchronous<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  source: S,
  compare: SynchronousComparison<E>,
): void {
  const values = toSlice(source);
  writeSorted(
    values,
    sortValuesSynchronous(
      logicalValues(values, copyElement, fromStorage),
      compare,
    ),
    copyElement,
    toStorage,
  );
}

export function SortStableFuncSynchronous<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  source: S,
  compare: SynchronousComparison<E>,
): void {
  SortFuncSynchronous(
    toSlice,
    copyElement,
    fromStorage,
    toStorage,
    source,
    compare,
  );
}

function logicalValues<E, EStorage>(
  source: RuntimeSlice<EStorage>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
): E[] {
  const result: E[] = [];
  for (let index = 0; index < source.length; index += 1) {
    result.push(readElement(
      source,
      integerFromHost(index),
      copyElement,
      fromStorage,
    ));
  }
  return result;
}

function writeSorted<E, EStorage>(
  target: RuntimeSlice<EStorage>,
  values: readonly E[],
  copyElement: CopyValue<E>,
  toStorage: ToContainerStorage<E, EStorage>,
): void {
  let index = 0;
  for (const value of values) {
    storeElement(
      target,
      integerFromHost(index),
      value,
      copyElement,
      toStorage,
    );
    index += 1;
  }
}

export async function sortValues<E>(
  values: readonly E[],
  compare: Comparison<E>,
): Promise<E[]> {
  if (values.length < 2) {
    return [...values];
  }
  let source = [...values];
  let target = new Array<E>(values.length);
  for (let width = 1; width < values.length; width *= 2) {
    for (let start = 0; start < values.length; start += width * 2) {
      const middle = Math.min(start + width, values.length);
      const end = Math.min(start + width * 2, values.length);
      let left = start;
      let right = middle;
      let output = start;
      while (left < middle && right < end) {
        const leftValue = GoDenseIndex.get(source, left);
        const rightValue = GoDenseIndex.get(source, right);
        if (await callComparison(compare, leftValue, rightValue) <= 0n) {
          target[output] = leftValue;
          left += 1;
        } else {
          target[output] = rightValue;
          right += 1;
        }
        output += 1;
      }
      while (left < middle) {
        target[output] = GoDenseIndex.get(source, left);
        left += 1;
        output += 1;
      }
      while (right < end) {
        target[output] = GoDenseIndex.get(source, right);
        right += 1;
        output += 1;
      }
    }
    [source, target] = [target, source];
  }
  return source;
}

export function sortValuesSynchronous<E>(
  values: readonly E[],
  compare: SynchronousComparison<E>,
): E[] {
  if (values.length < 2) {
    return [...values];
  }
  if (compare === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  let source = [...values];
  let target = new Array<E>(values.length);
  for (let width = 1; width < values.length; width *= 2) {
    for (let start = 0; start < values.length; start += width * 2) {
      const middle = Math.min(start + width, values.length);
      const end = Math.min(start + width * 2, values.length);
      let left = start;
      let right = middle;
      let output = start;
      while (left < middle && right < end) {
        const leftValue = GoDenseIndex.get(source, left);
        const rightValue = GoDenseIndex.get(source, right);
        if (compare(leftValue, rightValue) <= 0n) {
          target[output] = leftValue;
          left += 1;
        } else {
          target[output] = rightValue;
          right += 1;
        }
        output += 1;
      }
      while (left < middle) {
        target[output] = GoDenseIndex.get(source, left);
        left += 1;
        output += 1;
      }
      while (right < end) {
        target[output] = GoDenseIndex.get(source, right);
        right += 1;
        output += 1;
      }
    }
    [source, target] = [target, source];
  }
  return source;
}
