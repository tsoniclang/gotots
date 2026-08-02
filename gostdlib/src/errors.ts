import type { GoError } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { bool, gostring } from "@gotots/runtime/scalars.js";

import {
  MessageWrappedErrors,
  WrappedProviderError,
} from "./internal/portable/errors/tree.js";
import { unsupported } from "./internal/portable/errors/sentinel.js";
import { ProviderError } from "./internal/runtime/error.js";

export const state: {
  ErrUnsupported: GoError;
} = {
  ErrUnsupported: unsupported,
};

export function New(text: gostring): GoError {
  return new ProviderError(text);
}

export function Unwrap(failure: GoError | undefined): GoError | undefined {
  return failure instanceof WrappedProviderError
    ? failure.Unwrap()
    : undefined;
}

export function Is(failure: GoError | undefined, target: GoError | undefined): bool {
  if (failure === undefined || target === undefined) {
    return failure === target;
  }
  if (target.$go$type.comparable && failure.$go$equal(target)) {
    return true;
  }
  if (failure instanceof WrappedProviderError) {
    return Is(failure.Unwrap(), target);
  }
  if (failure instanceof MessageWrappedErrors) {
    return failure.UnwrapAll().some((cause) => Is(cause, target));
  }
  return false;
}

export function AsType<E>(
  failure: GoError | undefined,
): [E, bool] {
  return GoPanic.raiseRuntime("errors.AsType requires a generated generic specialization");
}
