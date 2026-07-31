import type { GoError } from "@gotots/runtime/interface-value.js";
import type { gostring } from "@gotots/runtime/scalars.js";

import { ProviderError } from "../../runtime/error.js";

export const ErrRange = new ProviderError("value out of range");
const errSyntax = new ProviderError("invalid syntax");

export class NumberError extends ProviderError {
  constructor(
    functionName: gostring,
    value: gostring,
    private readonly cause: GoError,
  ) {
    super(`strconv.${functionName}: parsing ${JSON.stringify(value)}: ${cause.Error()}`);
  }

  Unwrap(): GoError {
    return this.cause;
  }
}

export function rangeError(functionName: gostring, value: gostring): NumberError {
  return new NumberError(functionName, value, ErrRange);
}

export function syntaxError(functionName: gostring, value: gostring): NumberError {
  return new NumberError(functionName, value, errSyntax);
}
