import type { GoRecovery } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { bool, int64 } from "@gotots/runtime/scalars.js";

import { Seq } from "../../iter.js";
import type { OrderedValue } from "../portable/cmp/ordered.js";
import {
  AppendSeqCooperative,
  CollectCooperative,
  ContainsFuncCooperative,
  DeleteFuncCooperative,
  IndexFuncCooperative,
  SortFuncCooperative,
  SortStableFuncCooperative,
  SortedCooperative,
  ValuesCooperative,
} from "../portable/slices/cooperative.js";

type CooperativeSequence<T> = Seq<
  T,
  ((
    yieldValue: ((value: T) => Promise<bool>) | undefined,
  ) => void | Promise<void>) | undefined
>;

type CooperativePredicate<T> = (
  value: T,
  recovery?: GoRecovery,
) => Promise<bool>;

type CooperativeComparison<T> = (
  left: T,
  right: T,
  recovery?: GoRecovery,
) => Promise<int64>;

export function SlicesAppendSeqCooperative<
  Slice extends RuntimeSlice<Element>,
  Element,
>(
  source: Slice,
  sequence: CooperativeSequence<Element>,
  recovery?: GoRecovery,
): Promise<Slice>;
export async function SlicesAppendSeqCooperative<Element>(
  source: RuntimeSlice<Element>,
  sequence: CooperativeSequence<Element>,
  _recovery?: GoRecovery,
): Promise<RuntimeSlice<Element>> {
  return AppendSeqCooperative(source, sequence);
}

export function SlicesAppendSeqFullyCooperative<
  Slice extends RuntimeSlice<Element>,
  Element,
>(
  source: Slice,
  sequence: CooperativeSequence<Element>,
  recovery?: GoRecovery,
): Promise<Slice>;
export async function SlicesAppendSeqFullyCooperative<Element>(
  source: RuntimeSlice<Element>,
  sequence: CooperativeSequence<Element>,
  _recovery?: GoRecovery,
): Promise<RuntimeSlice<Element>> {
  return AppendSeqCooperative(source, sequence);
}

export async function SlicesCollectCooperative<
  Element,
  ElementStorage = Element,
>(
  sequence: CooperativeSequence<Element>,
  _recovery?: GoRecovery,
): Promise<RuntimeSlice<Element>> {
  return CollectCooperative(sequence);
}

export async function SlicesCollectFullyCooperative<
  Element,
  ElementStorage = Element,
>(
  sequence: CooperativeSequence<Element>,
  _recovery?: GoRecovery,
): Promise<RuntimeSlice<Element>> {
  return CollectCooperative(sequence);
}

export async function SlicesContainsFuncCooperative<
  Slice extends RuntimeSlice<Element>,
  Element,
>(
  source: Slice,
  predicate: CooperativePredicate<Element> | undefined,
  _recovery?: GoRecovery,
): Promise<bool> {
  return ContainsFuncCooperative(source, predicate);
}

export function SlicesDeleteFuncCooperative<
  Slice extends RuntimeSlice<Element>,
  Element,
>(
  source: Slice,
  predicate: CooperativePredicate<Element> | undefined,
  recovery?: GoRecovery,
): Promise<Slice>;
export async function SlicesDeleteFuncCooperative<Element>(
  source: RuntimeSlice<Element>,
  predicate: CooperativePredicate<Element> | undefined,
  _recovery?: GoRecovery,
): Promise<RuntimeSlice<Element>> {
  return DeleteFuncCooperative(source, predicate);
}

export async function SlicesIndexFuncCooperative<
  Slice extends RuntimeSlice<Element>,
  Element,
>(
  source: Slice,
  predicate: CooperativePredicate<Element> | undefined,
  _recovery?: GoRecovery,
): Promise<int64> {
  return IndexFuncCooperative(source, predicate);
}

export async function SlicesSortFuncCooperative<
  Slice extends RuntimeSlice<Element>,
  Element,
>(
  source: Slice,
  compare: CooperativeComparison<Element> | undefined,
  _recovery?: GoRecovery,
): Promise<void> {
  return SortFuncCooperative(source, compare);
}

export async function SlicesSortStableFuncCooperative<
  Slice extends RuntimeSlice<Element>,
  Element,
>(
  source: Slice,
  compare: CooperativeComparison<Element> | undefined,
  _recovery?: GoRecovery,
): Promise<void> {
  return SortStableFuncCooperative(source, compare);
}

export async function SlicesSortedCooperative<
  Element extends OrderedValue,
  ElementStorage = Element,
>(
  sequence: CooperativeSequence<Element>,
  _recovery?: GoRecovery,
): Promise<RuntimeSlice<Element>> {
  return SortedCooperative(sequence);
}

export async function SlicesSortedFullyCooperative<
  Element extends OrderedValue,
  ElementStorage = Element,
>(
  sequence: CooperativeSequence<Element>,
  _recovery?: GoRecovery,
): Promise<RuntimeSlice<Element>> {
  return SortedCooperative(sequence);
}

export function SlicesValuesCooperative<
  Slice extends RuntimeSlice<Element>,
  Element,
>(
  source: Slice,
  _recovery?: GoRecovery,
): CooperativeSequence<Element> {
  return ValuesCooperative(source);
}

export function SlicesValuesFullyCooperative<
  Slice extends RuntimeSlice<Element>,
  Element,
>(
  source: Slice,
  _recovery?: GoRecovery,
): CooperativeSequence<Element> {
  return ValuesCooperative(source);
}
