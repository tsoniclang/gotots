import type { GoRecovery } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { bool, int64 } from "@gotots/runtime/scalars.js";

import {
  AppendSeqCooperative,
  CollectCooperative,
  type CooperativeSequence,
  ContainsFuncCooperative,
  DeleteFuncCooperative,
  IndexFuncCooperative,
  SortFuncCooperative,
  SortStableFuncCooperative,
  SortedCooperative,
  ValuesCooperative,
} from "../portable/slices/cooperative.js";
import type {
  BinaryLess,
  Convert,
  CopyValue,
  EqualValue,
  FromContainerStorage,
  ToContainerStorage,
  Zero,
} from "../portable/slices/capabilities.js";

type CooperativePredicate<E> = (
  value: E,
  recovery?: GoRecovery,
) => Promise<bool>;
type CooperativeComparison<E> = (
  left: E,
  right: E,
  recovery?: GoRecovery,
) => Promise<int64>;

export async function SlicesAppendSeqCooperative<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  fromSlice: Convert<RuntimeSlice<EStorage>, S>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  zeroElement: Zero<E>,
  source: S,
  sequence: CooperativeSequence<E>,
  _recovery?: GoRecovery,
): Promise<S> {
  return AppendSeqCooperative(
    toSlice,
    fromSlice,
    copyElement,
    fromStorage,
    toStorage,
    zeroElement,
    source,
    sequence,
  );
}

export async function SlicesAppendSeqFullyCooperative<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  fromSlice: Convert<RuntimeSlice<EStorage>, S>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  zeroElement: Zero<E>,
  source: S,
  sequence: CooperativeSequence<E>,
  _recovery?: GoRecovery,
): Promise<S> {
  return AppendSeqCooperative(
    toSlice,
    fromSlice,
    copyElement,
    fromStorage,
    toStorage,
    zeroElement,
    source,
    sequence,
  );
}

export async function SlicesCollectCooperative<E, EStorage>(
  copyElement: CopyValue<E>,
  toStorage: ToContainerStorage<E, EStorage>,
  sequence: CooperativeSequence<E>,
  _recovery?: GoRecovery,
): Promise<RuntimeSlice<EStorage>> {
  return CollectCooperative(copyElement, toStorage, sequence);
}

export async function SlicesCollectFullyCooperative<E, EStorage>(
  copyElement: CopyValue<E>,
  toStorage: ToContainerStorage<E, EStorage>,
  sequence: CooperativeSequence<E>,
  _recovery?: GoRecovery,
): Promise<RuntimeSlice<EStorage>> {
  return CollectCooperative(copyElement, toStorage, sequence);
}

export async function SlicesContainsFuncCooperative<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  source: S,
  predicate: CooperativePredicate<E> | undefined,
  _recovery?: GoRecovery,
): Promise<bool> {
  return ContainsFuncCooperative(
    toSlice,
    copyElement,
    fromStorage,
    source,
    predicate,
  );
}

export async function SlicesDeleteFuncCooperative<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  fromSlice: Convert<RuntimeSlice<EStorage>, S>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  zeroElement: Zero<E>,
  source: S,
  predicate: CooperativePredicate<E> | undefined,
  _recovery?: GoRecovery,
): Promise<S> {
  return DeleteFuncCooperative(
    toSlice,
    fromSlice,
    copyElement,
    fromStorage,
    toStorage,
    zeroElement,
    source,
    predicate,
  );
}

export async function SlicesIndexFuncCooperative<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  source: S,
  predicate: CooperativePredicate<E> | undefined,
  _recovery?: GoRecovery,
): Promise<int64> {
  return IndexFuncCooperative(
    toSlice,
    copyElement,
    fromStorage,
    source,
    predicate,
  );
}

export async function SlicesSortFuncCooperative<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  source: S,
  compare: CooperativeComparison<E> | undefined,
  _recovery?: GoRecovery,
): Promise<void> {
  return SortFuncCooperative(
    toSlice,
    copyElement,
    fromStorage,
    toStorage,
    source,
    compare,
  );
}

export async function SlicesSortStableFuncCooperative<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  source: S,
  compare: CooperativeComparison<E> | undefined,
  _recovery?: GoRecovery,
): Promise<void> {
  return SortStableFuncCooperative(
    toSlice,
    copyElement,
    fromStorage,
    toStorage,
    source,
    compare,
  );
}

export async function SlicesSortedCooperative<E, EStorage>(
  less: BinaryLess<E>,
  copyElement: CopyValue<E>,
  equal: EqualValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  sequence: CooperativeSequence<E>,
  _recovery?: GoRecovery,
): Promise<RuntimeSlice<EStorage>> {
  return SortedCooperative(
    less,
    copyElement,
    equal,
    fromStorage,
    toStorage,
    sequence,
  );
}

export async function SlicesSortedFullyCooperative<E, EStorage>(
  less: BinaryLess<E>,
  copyElement: CopyValue<E>,
  equal: EqualValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  toStorage: ToContainerStorage<E, EStorage>,
  sequence: CooperativeSequence<E>,
  _recovery?: GoRecovery,
): Promise<RuntimeSlice<EStorage>> {
  return SortedCooperative(
    less,
    copyElement,
    equal,
    fromStorage,
    toStorage,
    sequence,
  );
}

export function SlicesValuesCooperative<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  source: S,
  _recovery?: GoRecovery,
): CooperativeSequence<E> {
  return ValuesCooperative(toSlice, copyElement, fromStorage, source);
}

export function SlicesValuesFullyCooperative<S, E, EStorage>(
  toSlice: Convert<S, RuntimeSlice<EStorage>>,
  copyElement: CopyValue<E>,
  fromStorage: FromContainerStorage<E, EStorage>,
  source: S,
  _recovery?: GoRecovery,
): CooperativeSequence<E> {
  return ValuesCooperative(toSlice, copyElement, fromStorage, source);
}
