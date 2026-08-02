import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int64 } from "@gotots/runtime/scalars.js";

export function Compare<T>(left: T, right: T): int64 {
  return specializationRequired("cmp.Compare");
}

export function Or<T>(values: RuntimeSlice<T>): T {
  return specializationRequired("cmp.Or");
}

function specializationRequired(name: string): never {
  return GoPanic.raiseRuntime(`${name} requires a generated generic specialization`);
}
