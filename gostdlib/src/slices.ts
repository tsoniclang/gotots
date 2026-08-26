import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { bool, int } from "@gotots/gostdlib/internal/scalars.js";

import type { Seq } from "./internal/portable/iter/sequence.js";

export function AppendSeq<Slice, Element>(
  source: Slice,
  sequence: Seq<Element>,
): Slice {
  return specializationRequired("slices.AppendSeq");
}

export function BinarySearch<Slice, Element>(
  source: Slice,
  target: Element,
): [int, bool] {
  return specializationRequired("slices.BinarySearch");
}

export function BinarySearchFunc<Slice, Element, Target>(
  source: Slice,
  target: Target,
  compare: ((left: Element, right: Target) => int) | undefined,
): [int, bool] {
  return specializationRequired("slices.BinarySearchFunc");
}

export function Clip<Slice, Element>(source: Slice): Slice {
  return specializationRequired("slices.Clip");
}

export function Clone<Slice, Element>(source: Slice): Slice {
  return specializationRequired("slices.Clone");
}

export function Collect<Element>(
  sequence: Seq<Element>,
): RuntimeSlice<Element> {
  return specializationRequired("slices.Collect");
}

export function Compact<Slice, Element>(source: Slice): Slice {
  return specializationRequired("slices.Compact");
}

export function CompactFunc<Slice, Element>(
  source: Slice,
  equal: ((left: Element, right: Element) => bool) | undefined,
): Slice {
  return specializationRequired("slices.CompactFunc");
}

export function Compare<Slice, Element>(left: Slice, right: Slice): int {
  return specializationRequired("slices.Compare");
}

export function CompareFunc<LeftSlice, RightSlice, Left, Right>(
  left: LeftSlice,
  right: RightSlice,
  compare: ((left: Left, right: Right) => int) | undefined,
): int {
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

export function ContainsFunc<Slice, Element>(
  source: Slice,
  predicate: ((value: Element) => bool) | undefined,
): bool {
  return specializationRequired("slices.ContainsFunc");
}

export function Delete<Slice, Element>(
  source: Slice,
  start: int,
  end: int,
): Slice {
  return specializationRequired("slices.Delete");
}

export function DeleteFunc<Slice, Element>(
  source: Slice,
  predicate: ((value: Element) => bool) | undefined,
): Slice {
  return specializationRequired("slices.DeleteFunc");
}

export function Equal<Slice, Element>(left: Slice, right: Slice): bool {
  return specializationRequired("slices.Equal");
}

export function EqualFunc<LeftSlice, RightSlice, Left, Right>(
  left: LeftSlice,
  right: RightSlice,
  equal: ((left: Left, right: Right) => bool) | undefined,
): bool {
  return specializationRequired("slices.EqualFunc");
}

export function Grow<Slice, Element>(source: Slice, amount: int): Slice {
  return specializationRequired("slices.Grow");
}

export function Index<Slice, Element>(
  source: Slice,
  target: Element,
): int {
  return specializationRequired("slices.Index");
}

export function IndexFunc<Slice, Element>(
  source: Slice,
  predicate: ((value: Element) => bool) | undefined,
): int {
  return specializationRequired("slices.IndexFunc");
}

export function Insert<Slice, Element>(
  source: Slice,
  index: int,
  values: RuntimeSlice<Element>,
): Slice {
  return specializationRequired("slices.Insert");
}

export function Repeat<Slice, Element>(source: Slice, count: int): Slice {
  return specializationRequired("slices.Repeat");
}

export function Replace<Slice, Element>(
  source: Slice,
  start: int,
  end: int,
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

export function SortFunc<Slice, Element>(
  source: Slice,
  compare: ((left: Element, right: Element) => int) | undefined,
): void {
  return specializationRequired("slices.SortFunc");
}

export function SortStableFunc<Slice, Element>(
  source: Slice,
  compare: ((left: Element, right: Element) => int) | undefined,
): void {
  return specializationRequired("slices.SortStableFunc");
}

export function Sorted<Element>(
  sequence: Seq<Element>,
): RuntimeSlice<Element> {
  return specializationRequired("slices.Sorted");
}

export function SortedFunc<Element>(
  sequence: Seq<Element>,
  compare: ((left: Element, right: Element) => int) | undefined,
): RuntimeSlice<Element> {
  return specializationRequired("slices.SortedFunc");
}

export function Values<Slice, Element>(source: Slice): Seq<Element> {
  return specializationRequired("slices.Values");
}

function specializationRequired(name: string): never {
  return GoPanic.raiseRuntime(`${name} requires a generated generic specialization`);
}
