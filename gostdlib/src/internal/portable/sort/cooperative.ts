import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { Awaitable, bool, int64 } from "@gotots/gostdlib/internal/scalars.js";

import { hostInteger, integerFromHost } from "../../host-integer.js";

export interface Interface extends GoInterfaceValue {
  Len(): Awaitable<int64>;
  Less(left: int64, right: int64): Awaitable<bool>;
  Swap(left: int64, right: int64): Awaitable<void>;
}

export async function Sort(data: Interface | undefined): Promise<void> {
  await reorder(data);
}

export async function Stable(data: Interface | undefined): Promise<void> {
  await reorder(data);
}

export async function Search(
  length: int64,
  predicate: ((index: int64) => Awaitable<bool>) | undefined,
): Promise<int64> {
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
    if (await predicate(middle)) {
      high = middle;
    } else {
      low = middle + 1n;
    }
  }
  return low;
}

async function reorder(data: Interface | undefined): Promise<void> {
  if (data === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  const length = await data.Len();
  if (length < 0n) {
    GoPanic.raiseRuntime("sort: invalid interface length");
  }
  const order = Array.from(
    { length: hostInteger(length) },
    (_value, index): number => index,
  );
  await sortOrder(data, order);
  await applyPermutation(data, order);
}

async function sortOrder(data: Interface, order: number[]): Promise<void> {
  const merged = Array.from(order);
  for (let width = 1; width < order.length; width *= 2) {
    for (let start = 0; start < order.length; start += width * 2) {
      const middle = Math.min(start + width, order.length);
      const end = Math.min(start + width * 2, order.length);
      let left = start;
      let right = middle;
      for (let target = start; target < end; target += 1) {
        let takeRight = right < end && left >= middle;
        if (right < end && left < middle) {
          takeRight = await data.Less(
            integerFromHost(requiredIndex(order, right)),
            integerFromHost(requiredIndex(order, left)),
          );
        }
        if (takeRight) {
          merged[target] = requiredIndex(order, right);
          right += 1;
        } else {
          merged[target] = requiredIndex(order, left);
          left += 1;
        }
      }
      for (let target = start; target < end; target += 1) {
        order[target] = requiredIndex(merged, target);
      }
    }
  }
}

async function applyPermutation(
  data: Interface,
  order: readonly number[],
): Promise<void> {
  const current = Array.from(order, (_value, index): number => index);
  const position = Array.from(current);
  for (let target = 0; target < order.length; target += 1) {
    const wanted = requiredIndex(order, target);
    const source = requiredIndex(position, wanted);
    if (source === target) {
      continue;
    }
    await data.Swap(integerFromHost(target), integerFromHost(source));
    const displaced = requiredIndex(current, target);
    current[target] = wanted;
    current[source] = displaced;
    position[wanted] = target;
    position[displaced] = source;
  }
}

function requiredIndex(values: readonly number[], index: number): number {
  const selected = values[index];
  if (selected === undefined) {
    return GoPanic.raiseRuntime("sort: invalid permutation");
  }
  return selected;
}
