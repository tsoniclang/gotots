import type { GoError } from "@gotots/runtime/interface-value.js";
import type { gostring } from "@gotots/runtime/scalars.js";

import { WrappedProviderError } from "../errors/tree.js";
import { ProviderError } from "../../runtime/error.js";

export const ErrRange = new ProviderError("value out of range");
export const ErrSyntax = new ProviderError("invalid syntax");

const numberErrorType = Object.freeze({});

export class NumberError extends WrappedProviderError {
  private readonly message: gostring;

  constructor(
    functionName: gostring,
    value: gostring,
    private readonly cause: GoError,
  ) {
    super(numberErrorType);
    this.message = `strconv.${functionName}: parsing ${JSON.stringify(value)}: ${cause.Error()}`;
  }

  Error(): gostring {
    return this.message;
  }

  Unwrap(): GoError {
    return this.cause;
  }

  override $go$format(verb: string, _flags: string, _precision: number | undefined): string {
    if (verb === "T") {
      return "*strconv.NumError";
    }
    return verb === "q" ? JSON.stringify(this.message) : this.message;
  }
}

export function rangeError(functionName: gostring, value: gostring): NumberError {
  return new NumberError(functionName, value, ErrRange);
}

export function syntaxError(functionName: gostring, value: gostring): NumberError {
  return new NumberError(functionName, value, ErrSyntax);
}
