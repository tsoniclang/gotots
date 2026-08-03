import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { Awaitable, bool, gostring } from "@gotots/runtime/scalars.js";

import {
  MessageWrappedErrors,
  WrappedProviderError,
} from "../portable/errors/tree.js";

interface KernelError extends GoInterfaceValue {
  Error(): Awaitable<gostring>;
}

type AssertError<ErrorType> = (
  failure: KernelError | undefined,
) => [ErrorType, bool];

export function ErrorsAsTypeKernel<ErrorType>(
  assertError: AssertError<ErrorType>,
  failure: KernelError | undefined,
): [ErrorType, bool] {
  const absent = assertError(undefined);
  return asType(assertError, failure, absent);
}

function asType<ErrorType>(
  assertError: AssertError<ErrorType>,
  failure: KernelError | undefined,
  absent: [ErrorType, bool],
): [ErrorType, bool] {
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
