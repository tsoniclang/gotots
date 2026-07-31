import { GoPanic } from "@gotots/runtime/panic.js";
import type { bool, int64 } from "@gotots/runtime/scalars.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import { Seq } from "../iter/sequence.js";
import type { OrderedValue } from "../cmp/ordered.js";
import { sliceValues } from "../../runtime/slice.js";
import { Sort, SortFunc } from "./sort.js";

type Comparison<T> = ((left: T, right: T) => int64) | undefined;

export function AppendSeq<T>(
  source: RuntimeSlice<T>,
  sequence: Seq<T>,
): RuntimeSlice<T> {
  const values: T[] = [];
  runSequence(sequence, (value): bool => {
    values.push(value);
    return true;
  });
  if (values.length === 0) {
    return source;
  }
  return RuntimeSlice.literal([...sliceValues(source), ...values]);
}

export function Collect<T>(sequence: Seq<T>): RuntimeSlice<T> {
  return AppendSeq(RuntimeSlice.nil<T>(), sequence);
}

export function Sorted<T extends OrderedValue>(
  sequence: Seq<T>,
): RuntimeSlice<T> {
  const result = Collect(sequence);
  Sort(result);
  return result;
}

export function SortedFunc<T>(
  sequence: Seq<T>,
  compare: Comparison<T>,
): RuntimeSlice<T> {
  const result = Collect(sequence);
  SortFunc(result, compare);
  return result;
}

export function Values<T>(source: RuntimeSlice<T>): Seq<T> {
  return new Seq<T>((yieldValue): void => {
    if (yieldValue === undefined) {
      GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
    }
    for (let index = 0; index < source.length; index += 1) {
      if (!yieldValue(source.get(index))) {
        return;
      }
    }
  });
}

function runSequence<T>(
  sequence: Seq<T>,
  yieldValue: (value: T) => bool,
): void {
  if (sequence.value === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  sequence.value(yieldValue);
}
