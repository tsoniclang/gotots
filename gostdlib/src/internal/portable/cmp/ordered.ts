import type { int64 } from "@gotots/runtime/scalars.js";

export type OrderedValue = number | bigint | string;

export function Compare<T extends OrderedValue>(left: T, right: T): int64 {
  const leftNaN = left !== left;
  const rightNaN = right !== right;
  if (leftNaN) {
    return rightNaN ? 0 : -1;
  }
  if (rightNaN) {
    return 1;
  }
  if (left < right) {
    return -1;
  }
  if (left > right) {
    return 1;
  }
  return 0;
}
