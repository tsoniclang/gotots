import type { GoRecovery } from "@gotots/runtime/panic.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { bool, int64 } from "@gotots/runtime/scalars.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import { Seq } from "../iter/sequence.js";
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
  type Zero,
} from "./capabilities.js";

type AsyncPredicate<T> = (
  (value: T, recovery?: GoRecovery) => Promise<bool>
) | undefined;
type AsyncComparison<T> = (
  (left: T, right: T, recovery?: GoRecovery) => Promise<int64>
) | undefined;
type CooperativeYield<T> = (
  value: T,
  recovery?: GoRecovery,
) => Promise<bool>;
export type CooperativeSequence<T> = Seq<
  T,
  ((
    yieldValue: CooperativeYield<T> | undefined,
    recovery?: GoRecovery,
  ) => Promise<void>) | undefined
>;

export async function AppendSeqCooperative<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  fromSlice: Convert<RuntimeSlice<EStorage>, S>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  zeroElement: Zero<E>,
  source: S,
  sequence: CooperativeSequence<E>,
): Promise<S> {
  const appended = await collectLogical(sequence, copyElement);
  if (appended.length === 0) {
    return source;
  }
  const values = toSlice(source);
  const nextLength = values.length + appended.length;
  if (nextLength <= values.capacity) {
    return fromSlice(values.append(
      toStorage(zeroElement()),
      appended.map((value): EStorage => toStorage(copyElement(value))),
    ));
  }
  let nextCapacity = values.capacity === 0 ? 1 : values.capacity * 2;
  while (nextCapacity < nextLength) {
    nextCapacity *= 2;
  }
  const result = RuntimeSlice.make<EStorage>(
    nextLength,
    nextCapacity,
    toStorage(zeroElement()),
  );
  const initialized = result.slice(0, nextCapacity, null);
  for (let index = 0; index < nextCapacity; index += 1) {
    initialized.set(index, toStorage(zeroElement()));
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
  for (const [index, value] of appended.entries()) {
    storeElement(
      result,
      values.length + index,
      value,
      copyElement,
      toStorage,
    );
  }
  return fromSlice(result);
}

export async function CollectCooperative<E, EStorage>(
  copyElement: CopyValue<E>,
  toStorage: ToContainerStorage<E, EStorage>,
  sequence: CooperativeSequence<E>,
): Promise<RuntimeSlice<EStorage>> {
  const values = await collectLogical(sequence, copyElement);
  return values.length === 0
    ? RuntimeSlice.nil<EStorage>()
    : RuntimeSlice.literal(
      values.map((value): EStorage => toStorage(copyElement(value))),
    );
}

export async function ContainsFuncCooperative<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  source: S,
  predicate: AsyncPredicate<E>,
): Promise<bool> {
  return await IndexFuncCooperative(
    toSlice,
    copyElement,
    fromStorage,
    source,
    predicate,
  ) >= 0;
}

export async function DeleteFuncCooperative<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  fromSlice: Convert<RuntimeSlice<EStorage>, S>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  zeroElement: Zero<E>,
  source: S,
  predicate: AsyncPredicate<E>,
): Promise<S> {
  const values = toSlice(source);
  let write = 0;
  for (let read = 0; read < values.length; read += 1) {
    const value = readElement(values, read, copyElement, fromStorage);
    if (!await callPredicate(predicate, value)) {
      storeElement(values, write, value, copyElement, toStorage);
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

export async function IndexFuncCooperative<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  source: S,
  predicate: AsyncPredicate<E>,
): Promise<int64> {
  const values = toSlice(source);
  for (let index = 0; index < values.length; index += 1) {
    if (
      await callPredicate(
        predicate,
        readElement(values, index, copyElement, fromStorage),
      )
    ) {
      return index;
    }
  }
  return -1;
}

export async function SortFuncCooperative<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  source: S,
  compare: AsyncComparison<E>,
): Promise<void> {
  const target = toSlice(source);
  const values = await mergeSort(
    logicalValues(target, copyElement, fromStorage),
    compare,
  );
  for (const [index, value] of values.entries()) {
    storeElement(target, index, value, copyElement, toStorage);
  }
}

export async function SortStableFuncCooperative<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  source: S,
  compare: AsyncComparison<E>,
): Promise<void> {
  await SortFuncCooperative(
    toSlice,
    copyElement,
    fromStorage,
    toStorage,
    source,
    compare,
  );
}

export async function SortedCooperative<E, EStorage>(
  less: BinaryLess<E>,
  copyElement: CopyValue<E>,
  equal: EqualValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  sequence: CooperativeSequence<E>,
): Promise<RuntimeSlice<EStorage>> {
  const result = await CollectCooperative(copyElement, toStorage, sequence);
  const values = logicalValues(result, copyElement, fromStorage);
  values.sort(
    (left, right): number => orderedCompare(less, equal, left, right),
  );
  return RuntimeSlice.literal(
    values.map((value): EStorage => toStorage(copyElement(value))),
  );
}

export function ValuesCooperative<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  source: S,
): CooperativeSequence<E> {
  const values = toSlice(source);
  return new Seq<E, CooperativeSequence<E>["value"]>(
    async (yieldValue): Promise<void> => {
      if (yieldValue === undefined) {
        GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
      }
      for (let index = 0; index < values.length; index += 1) {
        if (!await yieldValue(
          readElement(values, index, copyElement, fromStorage),
        )) {
          return;
        }
      }
    },
  );
}

function logicalValues<E, EStorage>(
  source: RuntimeSlice<EStorage>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
): E[] {
  const values: E[] = [];
  for (let index = 0; index < source.length; index += 1) {
    values.push(readElement(source, index, copyElement, fromStorage));
  }
  return values;
}

async function collectLogical<E>(
  sequence: CooperativeSequence<E>,
  copyElement: CopyValue<E>,
): Promise<E[]> {
  const implementation = sequence.value;
  if (implementation === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  const values: E[] = [];
  await implementation(async (value): Promise<bool> => {
    values.push(copyElement(value));
    return true;
  });
  return values;
}

async function callPredicate<E>(
  predicate: AsyncPredicate<E>,
  value: E,
): Promise<bool> {
  if (predicate === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return predicate(value);
}

async function callComparison<E>(
  compare: AsyncComparison<E>,
  left: E,
  right: E,
): Promise<int64> {
  if (compare === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return compare(left, right);
}

async function mergeSort<E>(
  values: readonly E[],
  compare: AsyncComparison<E>,
): Promise<E[]> {
  if (values.length < 2) {
    return [...values];
  }
  const middle = Math.floor(values.length / 2);
  const left = await mergeSort(values.slice(0, middle), compare);
  const right = await mergeSort(values.slice(middle), compare);
  const merged: E[] = [];
  const leftIterator = left.values();
  const rightIterator = right.values();
  let leftResult = leftIterator.next();
  let rightResult = rightIterator.next();
  while (!leftResult.done && !rightResult.done) {
    if (await callComparison(
      compare,
      leftResult.value,
      rightResult.value,
    ) <= 0) {
      merged.push(leftResult.value);
      leftResult = leftIterator.next();
    } else {
      merged.push(rightResult.value);
      rightResult = rightIterator.next();
    }
  }
  while (!leftResult.done) {
    merged.push(leftResult.value);
    leftResult = leftIterator.next();
  }
  while (!rightResult.done) {
    merged.push(rightResult.value);
    rightResult = rightIterator.next();
  }
  return merged;
}
