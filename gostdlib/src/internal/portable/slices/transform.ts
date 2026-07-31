import { GoPanic } from "@gotots/runtime/panic.js";
import type { bool, int64 } from "@gotots/runtime/scalars.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import { sliceValues } from "../../runtime/slice.js";
import { callEquality, callPredicate } from "./read.js";

type Predicate<T> = ((value: T) => bool) | undefined;
type Equality<T> = ((left: T, right: T) => bool) | undefined;

export function Clip<T>(source: RuntimeSlice<T>): RuntimeSlice<T> {
  return source.slice(0, source.length, source.length);
}

export function Clone<T>(source: RuntimeSlice<T>): RuntimeSlice<T> {
  return source.isNil()
    ? RuntimeSlice.nil<T>()
    : RuntimeSlice.literal(sliceValues(source));
}

export function Compact<T>(source: RuntimeSlice<T>): RuntimeSlice<T> {
  return compactBy(source, (left, right): bool => left === right);
}

export function CompactFunc<T>(
  source: RuntimeSlice<T>,
  equal: Equality<T>,
): RuntimeSlice<T> {
  return compactBy(
    source,
    (left, right): bool => callEquality(equal, left, right),
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
  validateRange(source.length, start, end);
  if (start === end) {
    return source;
  }
  const values = sliceValues(source);
  values.splice(start, end - start);
  return resultLike(source, values);
}

export function DeleteFunc<T>(
  source: RuntimeSlice<T>,
  predicate: Predicate<T>,
): RuntimeSlice<T> {
  const values: T[] = [];
  for (let index = 0; index < source.length; index += 1) {
    const value = source.get(index);
    if (!callPredicate(predicate, value)) {
      values.push(value);
    }
  }
  return resultLike(source, values);
}

export function Insert<T>(
  source: RuntimeSlice<T>,
  index: int64,
  values: RuntimeSlice<T>,
): RuntimeSlice<T> {
  validateIndex(source.length, index, true);
  if (values.length === 0) {
    return source;
  }
  const result = sliceValues(source);
  result.splice(index, 0, ...sliceValues(values));
  return RuntimeSlice.literal(result);
}

export function Repeat<T>(
  source: RuntimeSlice<T>,
  count: int64,
): RuntimeSlice<T> {
  if (!Number.isSafeInteger(count) || count < 0) {
    GoPanic.raiseRuntime("the result of (len(x) * count) overflows");
  }
  const resultLength = source.length * count;
  if (!Number.isSafeInteger(resultLength)) {
    GoPanic.raiseRuntime("the result of (len(x) * count) overflows");
  }
  const values: T[] = [];
  for (let repetition = 0; repetition < count; repetition += 1) {
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
  validateRange(source.length, start, end);
  const values = sliceValues(source);
  values.splice(start, end - start, ...sliceValues(replacement));
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

function compactBy<T>(
  source: RuntimeSlice<T>,
  equal: (left: T, right: T) => bool,
): RuntimeSlice<T> {
  if (source.length < 2) {
    return source;
  }
  let previous = source.get(0);
  const values: T[] = [previous];
  for (let index = 1; index < source.length; index += 1) {
    const value = source.get(index);
    if (!equal(previous, value)) {
      values.push(value);
      previous = value;
    }
  }
  return resultLike(source, values);
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
