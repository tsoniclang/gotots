import { GoPanic } from "@gotots/runtime/panic.js";
import type { bool, int64 } from "@gotots/runtime/scalars.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import { Seq } from "../iter/sequence.js";
import type { OrderedValue } from "../cmp/ordered.js";
import { sliceValues } from "../../runtime/slice.js";
import { Sort } from "./sort.js";

type AsyncPredicate<T> = ((value: T) => Promise<bool>) | undefined;
type AsyncComparison<T> = ((left: T, right: T) => Promise<int64>) | undefined;
type CooperativeSequence<T> = Seq<
  T,
  ((
    yieldValue: ((value: T) => Promise<bool>) | undefined,
  ) => void | Promise<void>) | undefined
>;

export async function AppendSeqCooperative<T>(
  source: RuntimeSlice<T>,
  sequence: CooperativeSequence<T>,
): Promise<RuntimeSlice<T>> {
  const values = await collectValues(sequence);
  if (values.length === 0) {
    return source;
  }
  return RuntimeSlice.literal([...sliceValues(source), ...values]);
}

export async function CollectCooperative<T>(
  sequence: CooperativeSequence<T>,
): Promise<RuntimeSlice<T>> {
  const values = await collectValues(sequence);
  return values.length === 0
    ? RuntimeSlice.nil<T>()
    : RuntimeSlice.literal(values);
}

export async function ContainsFuncCooperative<T>(
  source: RuntimeSlice<T>,
  predicate: AsyncPredicate<T>,
): Promise<bool> {
  return (await IndexFuncCooperative(source, predicate)) >= 0;
}

export async function DeleteFuncCooperative<T>(
  source: RuntimeSlice<T>,
  predicate: AsyncPredicate<T>,
): Promise<RuntimeSlice<T>> {
  const values: T[] = [];
  for (let index = 0; index < source.length; index += 1) {
    const value = source.get(index);
    if (!await callPredicate(predicate, value)) {
      values.push(value);
    }
  }
  if (values.length === 0 && source.isNil()) {
    return RuntimeSlice.nil<T>();
  }
  return RuntimeSlice.literal(values);
}

export async function IndexFuncCooperative<T>(
  source: RuntimeSlice<T>,
  predicate: AsyncPredicate<T>,
): Promise<int64> {
  for (let index = 0; index < source.length; index += 1) {
    if (await callPredicate(predicate, source.get(index))) {
      return index;
    }
  }
  return -1;
}

export async function SortFuncCooperative<T>(
  source: RuntimeSlice<T>,
  compare: AsyncComparison<T>,
): Promise<void> {
  const values = await mergeSort(sliceValues(source), compare);
  let index = 0;
  for (const value of values) {
    source.set(index, value);
    index += 1;
  }
}

export async function SortStableFuncCooperative<T>(
  source: RuntimeSlice<T>,
  compare: AsyncComparison<T>,
): Promise<void> {
  await SortFuncCooperative(source, compare);
}

export async function SortedCooperative<T extends OrderedValue>(
  sequence: CooperativeSequence<T>,
): Promise<RuntimeSlice<T>> {
  const result = await CollectCooperative(sequence);
  Sort(result);
  return result;
}

export function ValuesCooperative<T>(
  source: RuntimeSlice<T>,
): CooperativeSequence<T> {
  return new Seq<T, CooperativeSequence<T>["value"]>(
    async (yieldValue): Promise<void> => {
      if (yieldValue === undefined) {
        GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
      }
      for (let index = 0; index < source.length; index += 1) {
        if (!await yieldValue(source.get(index))) {
          return;
        }
      }
    },
  );
}

async function collectValues<T>(
  sequence: CooperativeSequence<T>,
): Promise<T[]> {
  const implementation = sequence.value;
  if (implementation === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  const values: T[] = [];
  await implementation(async (value): Promise<bool> => {
    values.push(value);
    return true;
  });
  return values;
}

async function callPredicate<T>(
  predicate: AsyncPredicate<T>,
  value: T,
): Promise<bool> {
  if (predicate === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return predicate(value);
}

async function callComparison<T>(
  compare: AsyncComparison<T>,
  left: T,
  right: T,
): Promise<int64> {
  if (compare === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return compare(left, right);
}

async function mergeSort<T>(
  values: readonly T[],
  compare: AsyncComparison<T>,
): Promise<T[]> {
  if (values.length < 2) {
    return [...values];
  }
  const middle = Math.floor(values.length / 2);
  const left = await mergeSort(values.slice(0, middle), compare);
  const right = await mergeSort(values.slice(middle), compare);
  const merged: T[] = [];
  const leftIterator = left[Symbol.iterator]();
  const rightIterator = right[Symbol.iterator]();
  let leftEntry = leftIterator.next();
  let rightEntry = rightIterator.next();
  while (!leftEntry.done && !rightEntry.done) {
    if (await callComparison(compare, leftEntry.value, rightEntry.value) <= 0) {
      merged.push(leftEntry.value);
      leftEntry = leftIterator.next();
    } else {
      merged.push(rightEntry.value);
      rightEntry = rightIterator.next();
    }
  }
  while (!leftEntry.done) {
    merged.push(leftEntry.value);
    leftEntry = leftIterator.next();
  }
  while (!rightEntry.done) {
    merged.push(rightEntry.value);
    rightEntry = rightIterator.next();
  }
  return merged;
}
