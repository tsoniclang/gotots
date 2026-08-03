import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { Awaitable, bool, int64 } from "@gotots/runtime/scalars.js";

import type { Seq } from "./internal/portable/iter/sequence.js";

type Predicate<Element> = ((value: Element) => Awaitable<bool>) | undefined;
type Equality<Left, Right = Left> = (
  (left: Left, right: Right) => Awaitable<bool>
) | undefined;
type Comparison<Left, Right = Left> = (
  (left: Left, right: Right) => Awaitable<int64>
) | undefined;

export async function AppendSeq<Slice, Element>(
  source: Slice,
  sequence: Seq<Element>,
): Promise<Slice> {
  return specializationRequired("slices.AppendSeq");
}

export function BinarySearch<Slice, Element>(
  source: Slice,
  target: Element,
): [int64, bool] {
  return specializationRequired("slices.BinarySearch");
}

export async function BinarySearchFunc<Slice, Element, Target>(
  source: Slice,
  target: Target,
  compare: Comparison<Element, Target>,
): Promise<[int64, bool]> {
  return specializationRequired("slices.BinarySearchFunc");
}

export function Clip<Slice, Element>(source: Slice): Slice {
  return specializationRequired("slices.Clip");
}

export function Clone<Slice, Element>(source: Slice): Slice {
  return specializationRequired("slices.Clone");
}

export async function Collect<Element>(
  sequence: Seq<Element>,
): Promise<RuntimeSlice<Element>> {
  return specializationRequired("slices.Collect");
}

export function Compact<Slice, Element>(source: Slice): Slice {
  return specializationRequired("slices.Compact");
}

export async function CompactFunc<Slice, Element>(
  source: Slice,
  equal: Equality<Element>,
): Promise<Slice> {
  return specializationRequired("slices.CompactFunc");
}

export function Compare<Slice, Element>(left: Slice, right: Slice): int64 {
  return specializationRequired("slices.Compare");
}

export async function CompareFunc<LeftSlice, RightSlice, Left, Right>(
  left: LeftSlice,
  right: RightSlice,
  compare: Comparison<Left, Right>,
): Promise<int64> {
  return specializationRequired("slices.CompareFunc");
}

export function Concat<Slice, Element>(sources: RuntimeSlice<Slice>): Slice {
  return specializationRequired("slices.Concat");
}

export function Contains<Slice, Element>(
  source: Slice,
  target: Element,
): bool {
  return specializationRequired("slices.Contains");
}

export async function ContainsFunc<Slice, Element>(
  source: Slice,
  predicate: Predicate<Element>,
): Promise<bool> {
  return specializationRequired("slices.ContainsFunc");
}

export function Delete<Slice, Element>(
  source: Slice,
  start: int64,
  end: int64,
): Slice {
  return specializationRequired("slices.Delete");
}

export async function DeleteFunc<Slice, Element>(
  source: Slice,
  predicate: Predicate<Element>,
): Promise<Slice> {
  return specializationRequired("slices.DeleteFunc");
}

export function Equal<Slice, Element>(left: Slice, right: Slice): bool {
  return specializationRequired("slices.Equal");
}

export async function EqualFunc<LeftSlice, RightSlice, Left, Right>(
  left: LeftSlice,
  right: RightSlice,
  equal: Equality<Left, Right>,
): Promise<bool> {
  return specializationRequired("slices.EqualFunc");
}

export function Grow<Slice, Element>(source: Slice, amount: int64): Slice {
  return specializationRequired("slices.Grow");
}

export function Index<Slice, Element>(
  source: Slice,
  target: Element,
): int64 {
  return specializationRequired("slices.Index");
}

export async function IndexFunc<Slice, Element>(
  source: Slice,
  predicate: Predicate<Element>,
): Promise<int64> {
  return specializationRequired("slices.IndexFunc");
}

export function Insert<Slice, Element>(
  source: Slice,
  index: int64,
  values: RuntimeSlice<Element>,
): Slice {
  return specializationRequired("slices.Insert");
}

export function Repeat<Slice, Element>(source: Slice, count: int64): Slice {
  return specializationRequired("slices.Repeat");
}

export function Replace<Slice, Element>(
  source: Slice,
  start: int64,
  end: int64,
  replacement: RuntimeSlice<Element>,
): Slice {
  return specializationRequired("slices.Replace");
}

export function Reverse<Slice, Element>(source: Slice): void {
  return specializationRequired("slices.Reverse");
}

export function Sort<Slice, Element>(source: Slice): void {
  return specializationRequired("slices.Sort");
}

export async function SortFunc<Slice, Element>(
  source: Slice,
  compare: Comparison<Element>,
): Promise<void> {
  return specializationRequired("slices.SortFunc");
}

export async function SortStableFunc<Slice, Element>(
  source: Slice,
  compare: Comparison<Element>,
): Promise<void> {
  return specializationRequired("slices.SortStableFunc");
}

export async function Sorted<Element>(
  sequence: Seq<Element>,
): Promise<RuntimeSlice<Element>> {
  return specializationRequired("slices.Sorted");
}

export async function SortedFunc<Element>(
  sequence: Seq<Element>,
  compare: Comparison<Element>,
): Promise<RuntimeSlice<Element>> {
  return specializationRequired("slices.SortedFunc");
}

export function Values<Slice, Element>(source: Slice): Seq<Element> {
  return specializationRequired("slices.Values");
}

function specializationRequired(name: string): never {
  return GoPanic.raiseRuntime(`${name} requires a generated generic specialization`);
}
