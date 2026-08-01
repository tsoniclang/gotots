import type { int64 } from "@gotots/runtime/scalars.js";
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

type Comparison<T> = ((left: T, right: T) => int64) | undefined;

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
      (left, right): number => orderedCompare(less, equal, left, right),
    ),
    copyElement,
    toStorage,
  );
}

export function SortFunc<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  source: S,
  compare: Comparison<E>,
): void {
  const values = toSlice(source);
  writeSorted(
    values,
    logicalValues(values, copyElement, fromStorage).sort(
      (left, right): number => callComparison(compare, left, right),
    ),
    copyElement,
    toStorage,
  );
}

export function SortStableFunc<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  source: S,
  compare: Comparison<E>,
): void {
  const values = toSlice(source);
  const indexed = logicalValues(values, copyElement, fromStorage).map(
    (value, index): { readonly value: E; readonly index: number } => ({
      value,
      index,
    }),
  );
  indexed.sort((left, right): number => {
    const result = callComparison(compare, left.value, right.value);
    return result === 0 ? left.index - right.index : result;
  });
  writeSorted(
    values,
    indexed.map((entry): E => entry.value),
    copyElement,
    toStorage,
  );
}

function logicalValues<E, EStorage>(
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

function writeSorted<E, EStorage>(
  target: RuntimeSlice<EStorage>,
  values: readonly E[],
  copyElement: CopyValue<E>,
  toStorage: ToContainerStorage<E, EStorage>,
): void {
  let index = 0;
  for (const value of values) {
    storeElement(target, index, value, copyElement, toStorage);
    index += 1;
  }
}
