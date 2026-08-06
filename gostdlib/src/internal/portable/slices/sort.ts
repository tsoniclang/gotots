import type { Awaitable, int64 } from "@gotots/gostdlib/internal/scalars.js";
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
  const middle = Math.floor(values.length / 2);
  const left = await sortValues(values.slice(0, middle), compare);
  const right = await sortValues(values.slice(middle), compare);
  const result: E[] = [];
  const leftIterator = left[Symbol.iterator]();
  const rightIterator = right[Symbol.iterator]();
  let leftStep = leftIterator.next();
  let rightStep = rightIterator.next();
  while (!leftStep.done && !rightStep.done) {
    if (await callComparison(compare, leftStep.value, rightStep.value) <= 0n) {
      result.push(leftStep.value);
      leftStep = leftIterator.next();
    } else {
      result.push(rightStep.value);
      rightStep = rightIterator.next();
    }
  }
  while (!leftStep.done) {
    result.push(leftStep.value);
    leftStep = leftIterator.next();
  }
  while (!rightStep.done) {
    result.push(rightStep.value);
    rightStep = rightIterator.next();
  }
  return result;
}
