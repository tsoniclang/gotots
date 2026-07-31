import { GoPanic } from "@gotots/runtime/panic.js";
import type { bool, int64 } from "@gotots/runtime/scalars.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import { Compare as compareOrdered } from "../cmp/ordered.js";
import type { OrderedValue } from "../cmp/ordered.js";

type Predicate<T> = ((value: T) => bool) | undefined;
type Equality<L, R> = ((left: L, right: R) => bool) | undefined;
type Comparison<L, R> = ((left: L, right: R) => int64) | undefined;

export function BinarySearch<T extends OrderedValue>(
  source: RuntimeSlice<T>,
  target: T,
): [int64, bool] {
  let low = 0;
  let high = source.length;
  while (low < high) {
    const middle = Math.floor((low + high) / 2);
    if (compareOrdered(source.get(middle), target) < 0) {
      low = middle + 1;
    } else {
      high = middle;
    }
  }
  return [
    low,
    low < source.length && compareOrdered(source.get(low), target) === 0,
  ];
}

export function BinarySearchFunc<T, Target>(
  source: RuntimeSlice<T>,
  target: Target,
  compare: Comparison<T, Target>,
): [int64, bool] {
  let low = 0;
  let high = source.length;
  while (low < high) {
    const middle = Math.floor((low + high) / 2);
    if (callComparison(compare, source.get(middle), target) < 0) {
      low = middle + 1;
    } else {
      high = middle;
    }
  }
  return [
    low,
    low < source.length && callComparison(compare, source.get(low), target) === 0,
  ];
}

export function Compare<T extends OrderedValue>(
  left: RuntimeSlice<T>,
  right: RuntimeSlice<T>,
): int64 {
  const count = Math.min(left.length, right.length);
  for (let index = 0; index < count; index += 1) {
    const result = compareOrdered(left.get(index), right.get(index));
    if (result !== 0) {
      return result;
    }
  }
  return compareLengths(left.length, right.length);
}

export function CompareFunc<L, R>(
  left: RuntimeSlice<L>,
  right: RuntimeSlice<R>,
  compare: Comparison<L, R>,
): int64 {
  const count = Math.min(left.length, right.length);
  for (let index = 0; index < count; index += 1) {
    const result = callComparison(compare, left.get(index), right.get(index));
    if (result !== 0) {
      return result;
    }
  }
  return compareLengths(left.length, right.length);
}

export function Contains<T>(source: RuntimeSlice<T>, target: T): bool {
  return Index(source, target) >= 0;
}

export function ContainsFunc<T>(source: RuntimeSlice<T>, predicate: Predicate<T>): bool {
  return IndexFunc(source, predicate) >= 0;
}

export function Equal<T>(left: RuntimeSlice<T>, right: RuntimeSlice<T>): bool {
  if (left.length !== right.length) {
    return false;
  }
  for (let index = 0; index < left.length; index += 1) {
    if (left.get(index) !== right.get(index)) {
      return false;
    }
  }
  return true;
}

export function EqualFunc<L, R>(
  left: RuntimeSlice<L>,
  right: RuntimeSlice<R>,
  equal: Equality<L, R>,
): bool {
  if (left.length !== right.length) {
    return false;
  }
  for (let index = 0; index < left.length; index += 1) {
    if (!callEquality(equal, left.get(index), right.get(index))) {
      return false;
    }
  }
  return true;
}

export function Index<T>(source: RuntimeSlice<T>, target: T): int64 {
  for (let index = 0; index < source.length; index += 1) {
    if (source.get(index) === target) {
      return index;
    }
  }
  return -1;
}

export function IndexFunc<T>(source: RuntimeSlice<T>, predicate: Predicate<T>): int64 {
  for (let index = 0; index < source.length; index += 1) {
    if (callPredicate(predicate, source.get(index))) {
      return index;
    }
  }
  return -1;
}

export function callPredicate<T>(predicate: Predicate<T>, value: T): bool {
  if (predicate === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return predicate(value);
}

export function callEquality<L, R>(
  equal: Equality<L, R>,
  left: L,
  right: R,
): bool {
  if (equal === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return equal(left, right);
}

export function callComparison<L, R>(
  compare: Comparison<L, R>,
  left: L,
  right: R,
): int64 {
  if (compare === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return compare(left, right);
}

function compareLengths(left: number, right: number): int64 {
  if (left < right) {
    return -1;
  }
  if (left > right) {
    return 1;
  }
  return 0;
}
