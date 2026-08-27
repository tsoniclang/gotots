import { GoPanic } from "@gotots/runtime/panic.js";
import type { bool, int64 } from "@gotots/gostdlib/internal/scalars.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import { integerFromHost } from "../../host-integer.js";

import {
  type BinaryLess,
  compareLengths,
  type Convert,
  type CopyValue,
  type EqualValue,
  type FromContainerStorage,
  orderedCompare,
  readElement,
} from "./capabilities.js";

type Predicate<T> = ((value: T) => bool) | undefined;
type Equality<L, R> = ((left: L, right: R) => bool) | undefined;
type Comparison<L, R> = ((left: L, right: R) => int64) | undefined;

export function BinarySearch<S, E, EStorage>(
  less: BinaryLess<E>,
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  equal: EqualValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  source: S,
  target: E,
): [int64, bool] {
  const values = toSlice(source);
  let low = 0;
  let high = values.length;
  while (low < high) {
    const middle = Math.floor((low + high) / 2);
    const value = readElement(
      values,
      middle,
      copyElement,
      fromStorage,
    );
    if (orderedCompare(less, equal, value, target) < 0n) {
      low = middle + 1;
    } else {
      high = middle;
    }
  }
  return [
    integerFromHost(low),
    low < values.length
      && orderedCompare(
        less,
        equal,
        readElement(values, low, copyElement, fromStorage),
        target,
      ) === 0n,
  ];
}

export function BinarySearchFunc<S, E, EStorage, Target>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  source: S,
  target: Target,
  compare: Comparison<E, Target>,
): [int64, bool] {
  if (compare === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  const values = toSlice(source);
  let low = 0;
  let high = values.length;
  while (low < high) {
    const middle = Math.floor((low + high) / 2);
    if (
      compare(
        readElement(
          values,
          middle,
          copyElement,
          fromStorage,
        ),
        target,
      ) < 0n
    ) {
      low = middle + 1;
    } else {
      high = middle;
    }
  }
  return [
    integerFromHost(low),
    low < values.length
      && compare(
        readElement(values, low, copyElement, fromStorage),
        target,
      ) === 0n,
  ];
}

export function Compare<S, E, EStorage>(
  less: BinaryLess<E>,
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  equal: EqualValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  left: S,
  right: S,
): int64 {
  const leftValues = toSlice(left);
  const rightValues = toSlice(right);
  const count = Math.min(leftValues.length, rightValues.length);
  for (let index = 0; index < count; index += 1) {
    const result = orderedCompare(
      less,
      equal,
      readElement(leftValues, index, copyElement, fromStorage),
      readElement(rightValues, index, copyElement, fromStorage),
    );
    if (result !== 0n) {
      return result;
    }
  }
  return compareLengths(leftValues.length, rightValues.length);
}

export function CompareFunc<
  S1,
  S2,
  E1,
  E1Storage,
  E2,
  E2Storage,
>(
  leftSlice: Convert<S1, RuntimeSlice<E1Storage>>,
  rightSlice: Convert<S2, RuntimeSlice<E2Storage>>,
  copyLeft: CopyValue<E1>,
  copyRight: CopyValue<E2>,
  fromLeftStorage: FromContainerStorage<E1, E1Storage>,
  fromRightStorage: FromContainerStorage<E2, E2Storage>,
  left: S1,
  right: S2,
  compare: Comparison<E1, E2>,
): int64 {
  if (compare === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  const leftValues = leftSlice(left);
  const rightValues = rightSlice(right);
  const count = Math.min(leftValues.length, rightValues.length);
  for (let index = 0; index < count; index += 1) {
    const result = compare(
      readElement(leftValues, index, copyLeft, fromLeftStorage),
      readElement(rightValues, index, copyRight, fromRightStorage),
    );
    if (result !== 0n) {
      return result;
    }
  }
  return compareLengths(leftValues.length, rightValues.length);
}

export function Contains<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  equal: EqualValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  source: S,
  target: E,
): bool {
  return indexBy(
    toSlice(source),
    copyElement,
    equal,
    fromStorage,
    target,
  ) >= 0n;
}

export function ContainsFunc<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  source: S,
  predicate: Predicate<E>,
): bool {
  return indexFuncBy(
    toSlice(source),
    copyElement,
    fromStorage,
    predicate,
  ) >= 0n;
}

export function Equal<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  equal: EqualValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  left: S,
  right: S,
): bool {
  const leftValues = toSlice(left);
  const rightValues = toSlice(right);
  if (leftValues.length !== rightValues.length) {
    return false;
  }
  for (let index = 0; index < leftValues.length; index += 1) {
    if (
      !equal(
        readElement(leftValues, index, copyElement, fromStorage),
        readElement(rightValues, index, copyElement, fromStorage),
      )
    ) {
      return false;
    }
  }
  return true;
}

export function EqualFunc<
  S1,
  S2,
  E1,
  E1Storage,
  E2,
  E2Storage,
>(
  leftSlice: Convert<S1, RuntimeSlice<E1Storage>>,
  rightSlice: Convert<S2, RuntimeSlice<E2Storage>>,
  copyLeft: CopyValue<E1>,
  copyRight: CopyValue<E2>,
  fromLeftStorage: FromContainerStorage<E1, E1Storage>,
  fromRightStorage: FromContainerStorage<E2, E2Storage>,
  left: S1,
  right: S2,
  equal: Equality<E1, E2>,
): bool {
  if (equal === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  const leftValues = leftSlice(left);
  const rightValues = rightSlice(right);
  if (leftValues.length !== rightValues.length) {
    return false;
  }
  for (let index = 0; index < leftValues.length; index += 1) {
    if (
      !equal(
        readElement(leftValues, index, copyLeft, fromLeftStorage),
        readElement(rightValues, index, copyRight, fromRightStorage),
      )
    ) {
      return false;
    }
  }
  return true;
}

export function Index<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  equal: EqualValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  source: S,
  target: E,
): int64 {
  return indexBy(
    toSlice(source),
    copyElement,
    equal,
    fromStorage,
    target,
  );
}

export function IndexFunc<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  source: S,
  predicate: Predicate<E>,
): int64 {
  return indexFuncBy(
    toSlice(source),
    copyElement,
    fromStorage,
    predicate,
  );
}

function indexBy<E, EStorage>(
  source: RuntimeSlice<EStorage>,
  copyElement: CopyValue<E>,
  equal: EqualValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  target: E,
): int64 {
  for (let index = 0; index < source.length; index += 1) {
    if (equal(
      readElement(source, index, copyElement, fromStorage),
      target,
    )) {
      return integerFromHost(index);
    }
  }
  return -1n;
}

function indexFuncBy<E, EStorage>(
  source: RuntimeSlice<EStorage>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  predicate: Predicate<E>,
): int64 {
  if (predicate === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  for (let index = 0; index < source.length; index += 1) {
	if (
	  predicate(
		readElement(
          source,
          index,
          copyElement,
          fromStorage,
        ),
      )
    ) {
      return integerFromHost(index);
    }
  }
  return -1n;
}
