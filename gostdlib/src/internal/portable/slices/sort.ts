import type { int64 } from "@gotots/runtime/scalars.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import { Compare as compareOrdered } from "../cmp/ordered.js";
import type { OrderedValue } from "../cmp/ordered.js";
import { sliceValues } from "../../runtime/slice.js";
import { callComparison } from "./read.js";

type Comparison<T> = ((left: T, right: T) => int64) | undefined;

export function Sort<T extends OrderedValue>(source: RuntimeSlice<T>): void {
  writeSorted(source, sliceValues(source).sort(compareOrdered));
}

export function SortFunc<T>(
  source: RuntimeSlice<T>,
  compare: Comparison<T>,
): void {
  writeSorted(
    source,
    sliceValues(source).sort(
      (left, right): number => callComparison(compare, left, right),
    ),
  );
}

export function SortStableFunc<T>(
  source: RuntimeSlice<T>,
  compare: Comparison<T>,
): void {
  const values = sliceValues(source).map(
    (value, index): { readonly value: T; readonly index: number } => ({
      value,
      index,
    }),
  );
  values.sort((left, right): number => {
    const result = callComparison(compare, left.value, right.value);
    return result === 0 ? left.index - right.index : result;
  });
  writeSorted(
    source,
    values.map((entry): T => entry.value),
  );
}

function writeSorted<T>(target: RuntimeSlice<T>, values: readonly T[]): void {
  let index = 0;
  for (const value of values) {
    target.set(index, value);
    index += 1;
  }
}
