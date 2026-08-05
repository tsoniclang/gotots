import { GoPanic } from "@gotots/runtime/panic.js";
import type { bool, int } from "@gotots/gostdlib/internal/scalars.js";

export function Search(
  length: int,
  predicate: ((index: int) => bool) | undefined,
): int {
  if (length < 0n) {
    GoPanic.raiseRuntime("sort: negative length");
  }
  if (predicate === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  let low = 0n;
  let high = length;
  while (low < high) {
    const middle = low + (high - low) / 2n;
    if (predicate(middle)) {
      high = middle;
    } else {
      low = middle + 1n;
    }
  }
  return low;
}
