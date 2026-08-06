import type { int64 } from "@gotots/gostdlib/internal/scalars.js";

export type OrderedValue = number | bigint | string;
export type OrderedEquality<T> = (left: T, right: T) => boolean;
export type OrderedLess<T> = (left: T, right: T) => boolean;

export function Compare<T>(
  less: OrderedLess<T>,
  equal: OrderedEquality<T>,
  left: T,
  right: T,
): int64 {
  const leftNaN = !equal(left, left);
  const rightNaN = !equal(right, right);
  if (leftNaN) {
    return rightNaN ? 0n : -1n;
  }
  if (rightNaN) {
    return 1n;
  }
  if (less(left, right)) {
    return -1n;
  }
  if (less(right, left)) {
    return 1n;
  }
  return 0n;
}
