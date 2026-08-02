import type { GoError } from "@gotots/runtime/interface-value.js";
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

type AssertError<E> = (
  failure: GoError | undefined,
) => [E, bool];

export function AsType<E>(
  assertError: AssertError<E>,
  failure: GoError | undefined,
): [E, bool] {
  const absent = assertError(undefined);
  return asType(assertError, failure, absent);
}

function asType<E>(
  assertError: AssertError<E>,
  failure: GoError | undefined,
  absent: [E, bool],
): [E, bool] {
  let current = failure;
  while (current !== undefined) {
    const selected = assertError(current);
    if (selected[1]) {
      return selected;
    }
    if (current instanceof WrappedProviderError) {
      current = current.Unwrap();
      continue;
    }
    if (current instanceof MessageWrappedErrors) {
      for (const cause of current.UnwrapAll()) {
        const nested = asType(assertError, cause, absent);
        if (nested[1]) {
          return nested;
        }
      }
    }
    return absent;
  }
  return absent;
}
