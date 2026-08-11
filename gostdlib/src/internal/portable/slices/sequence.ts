import { GoPanic } from "@gotots/runtime/panic.js";
import type { Awaitable, bool, int64 } from "@gotots/gostdlib/internal/scalars.js";
import { hostInteger } from "../../host-integer.js";
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
import { callComparison } from "./read.js";
import { sortValues } from "./sort.js";

type Comparison<T> = ((left: T, right: T) => Awaitable<int64>) | undefined;

export async function AppendSeq<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  fromSlice: Convert<RuntimeSlice<EStorage>, S>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  zeroElement: Zero<E>,
  source: S,
  sequence: Seq<E>,
): Promise<S> {
  const appended: E[] = [];
  await runSequence(sequence, (value): bool => {
    appended.push(copyElement(value));
    return true;
  });
  if (appended.length === 0) {
    return source;
  }
  const values = toSlice(source);
  const nextLength = values.length + appended.length;
  if (nextLength <= values.capacity) {
    const stored = appended.map(
      (value): EStorage => toStorage(copyElement(value)),
    );
    return fromSlice(values.append(toStorage(zeroElement()), stored));
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

export async function Collect<E, EStorage>(
  copyElement: CopyValue<E>,
  toStorage: ToContainerStorage<E, EStorage>,
  sequence: Seq<E>,
): Promise<RuntimeSlice<EStorage>> {
  const values: EStorage[] = [];
  await runSequence(sequence, (value): bool => {
    values.push(toStorage(copyElement(value)));
    return true;
  });
  return values.length === 0
    ? RuntimeSlice.nil<EStorage>()
    : RuntimeSlice.literal(values);
}

export async function Sorted<E, EStorage>(
  less: BinaryLess<E>,
  copyElement: CopyValue<E>,
  equal: EqualValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  sequence: Seq<E>,
): Promise<RuntimeSlice<EStorage>> {
  const result = await Collect(copyElement, toStorage, sequence);
  const values = logicalValues(result, copyElement, fromStorage);
  values.sort(
    (left, right): number => hostInteger(orderedCompare(less, equal, left, right)),
  );
  return RuntimeSlice.literal(
    values.map((value): EStorage => toStorage(copyElement(value))),
  );
}

export async function SortedFunc<E, EStorage>(
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  sequence: Seq<E>,
  compare: Comparison<E>,
): Promise<RuntimeSlice<EStorage>> {
  const result = await Collect(copyElement, toStorage, sequence);
  const values = await sortValues(
    logicalValues(result, copyElement, fromStorage),
    compare,
  );
  return RuntimeSlice.literal(
    values.map((value): EStorage => toStorage(copyElement(value))),
  );
}

export function Values<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  source: S,
): Seq<E> {
  const values = toSlice(source);
  return new Seq<E>(async (yieldValue): Promise<void> => {
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
  });
}

function logicalValues<E, EStorage>(
  source: RuntimeSlice<EStorage>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
): E[] {
  const values: E[] = [];
  for (let index = 0; index < source.length; index += 1) {
    values.push(readElement(
      source,
      index,
      copyElement,
      fromStorage,
    ));
  }
  return values;
}

async function runSequence<T>(
  sequence: Seq<T>,
  yieldValue: (value: T) => Awaitable<bool>,
): Promise<void> {
  if (sequence.value === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  await sequence.value(yieldValue);
}
