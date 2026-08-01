import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { bool, gostring, int64 } from "@gotots/runtime/scalars.js";

export interface Interface extends GoInterfaceValue {
  Len(): int64;
  Less(left: int64, right: int64): bool;
  Swap(left: int64, right: int64): void;
}

export function Sort(data: Interface | undefined): void {
  reorder(data);
}

export function Stable(data: Interface | undefined): void {
  reorder(data);
}

export function Strings(values: RuntimeSlice<gostring>): void {
  const sorted: gostring[] = [];
  for (let index = 0; index < values.length; index += 1) {
    sorted.push(values.get(index));
  }
  sorted.sort((left, right): number => left < right ? -1 : left > right ? 1 : 0);
  for (let index = 0; index < sorted.length; index += 1) {
    values.set(index, sorted[index] ?? "");
  }
}

function reorder(data: Interface | undefined): void {
  if (data === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  const length = data.Len();
  if (!Number.isInteger(length) || length < 0) {
    GoPanic.raiseRuntime("sort: invalid interface length");
  }
  const order = Array.from({ length }, (_value, index): number => index);
  order.sort((left, right): number => {
    if (data.Less(left, right)) {
      return -1;
    }
    if (data.Less(right, left)) {
      return 1;
    }
    return left - right;
  });
  applyPermutation(data, order);
}

function applyPermutation(data: Interface, order: readonly number[]): void {
  const current = Array.from(order, (_value, index): number => index);
  const position = Array.from(current);
  for (let target = 0; target < order.length; target += 1) {
    const wanted = order[target];
    if (wanted === undefined) {
      GoPanic.raiseRuntime("sort: invalid permutation");
    }
    const source = position[wanted];
    if (source === undefined) {
      GoPanic.raiseRuntime("sort: invalid permutation");
    }
    if (source === target) {
      continue;
    }
    data.Swap(target, source);
    const displaced = current[target];
    if (displaced === undefined) {
      GoPanic.raiseRuntime("sort: invalid permutation");
    }
    current[target] = wanted;
    current[source] = displaced;
    position[wanted] = target;
    position[displaced] = source;
  }
}
