import type { GoError } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { bool, gostring } from "@gotots/gostdlib/internal/scalars.js";

import {
  AsProviderErrorIsDirect,
  AsProviderErrorUnwrapDirect,
  AsProviderErrorUnwrapManyDirect,
} from "./internal/facets/provider-error.js";
import { unsupported } from "./internal/portable/errors/sentinel.js";
import { ProviderError } from "./internal/runtime/error.js";
import { sliceValues } from "./internal/runtime/slice.js";

export const state: {
  ErrUnsupported: GoError;
} = {
  ErrUnsupported: unsupported,
};

export function New(text: gostring): GoError {
  return new ProviderError(text);
}

export function Unwrap(failure: GoError | undefined): GoError | undefined {
  if (failure === undefined) {
    return undefined;
  }
  return AsProviderErrorUnwrapDirect(failure)?.Unwrap();
}

export function Is(failure: GoError | undefined, target: GoError | undefined): bool {
  if (failure === undefined || target === undefined) {
    return failure === target;
  }
  if (target.$go$type.comparable && failure.$go$equal(target)) {
    return true;
  }
  const custom = AsProviderErrorIsDirect(failure);
  if (custom !== undefined && custom.Is(target)) {
    return true;
  }
  const direct = AsProviderErrorUnwrapDirect(failure);
  if (direct !== undefined) {
    return Is(direct.Unwrap(), target);
  }
  const many = AsProviderErrorUnwrapManyDirect(failure);
  if (many !== undefined) {
    return sliceValues(many.Unwrap()).some((cause) => Is(cause, target));
  }
  return false;
}

export function AsType<E>(
  failure: GoError | undefined,
): [E, bool] {
  return GoPanic.raiseRuntime("errors.AsType requires a generated generic specialization");
}
