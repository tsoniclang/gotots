import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int } from "@gotots/gostdlib/internal/scalars.js";

export function Compare<T>(left: T, right: T): int {
  return specializationRequired("cmp.Compare");
}

export function Or<T>(values: RuntimeSlice<T>): T {
  return specializationRequired("cmp.Or");
}

function specializationRequired(name: string): never {
  return GoPanic.raiseRuntime(`${name} requires a generated generic specialization`);
}
