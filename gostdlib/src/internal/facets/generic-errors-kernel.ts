import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { bool } from "@gotots/gostdlib/internal/scalars.js";

import {
  MessageWrappedErrors,
  WrappedProviderError,
} from "../portable/errors/tree.js";
import { sliceValues } from "../runtime/slice.js";

type AssertError<ErrorType> = (
  failure: GoInterfaceValue | undefined,
) => [ErrorType, bool];

export function ErrorsAsTypeKernel<ErrorType>(
  assertError: AssertError<ErrorType>,
  failure: GoInterfaceValue | undefined,
): [ErrorType, bool] {
  const absent = assertError(undefined);
  return asType(assertError, failure, absent);
}

function asType<ErrorType>(
  assertError: AssertError<ErrorType>,
  failure: GoInterfaceValue | undefined,
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
      for (const cause of sliceValues(current.Unwrap())) {
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
