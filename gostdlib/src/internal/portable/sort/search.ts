import { GoPanic } from "@gotots/runtime/panic.js";
import type { bool, int64 } from "@gotots/runtime/scalars.js";

export function Search(
  length: int64,
  predicate: ((index: int64) => bool) | undefined,
): int64 {
  if (length < 0) {
    GoPanic.raiseRuntime("sort: negative length");
  }
  if (predicate === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  let low = 0;
  let high = length;
  while (low < high) {
    const middle = low + Math.trunc((high - low) / 2);
    if (predicate(middle)) {
      high = middle;
    } else {
      low = middle + 1;
    }
  }
  return low;
}
