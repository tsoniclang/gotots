import type { GoError } from "@gotots/runtime/interface-value.js";
import type { bool, gostring } from "@gotots/runtime/scalars.js";

import { WrappedProviderError } from "./internal/portable/errors/tree.js";
import { ProviderError } from "./internal/runtime/error.js";

export function New(text: gostring): GoError {
  return new ProviderError(text);
}

export function Is(failure: GoError | undefined, target: GoError | undefined): bool {
  if (failure === undefined || target === undefined) {
    return failure === target;
  }
  if (failure.$go$equal(target)) {
    return true;
  }
  if (failure instanceof WrappedProviderError) {
    return Is(failure.Unwrap(), target);
  }
  return false;
}
