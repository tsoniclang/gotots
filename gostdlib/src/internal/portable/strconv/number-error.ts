import type { GoError } from "@gotots/runtime/interface-value.js";
import { GoString } from "@gotots/runtime/string-value.js";
import type { gostring } from "@gotots/gostdlib/internal/scalars.js";

import { WrappedProviderError } from "../errors/tree.js";
import { ProviderError } from "../../runtime/error.js";

export const ErrRange = ProviderError.fromText("value out of range");
export const ErrSyntax = ProviderError.fromText("invalid syntax");

const numberErrorType = Object.freeze({ comparable: true });

export class NumberError extends WrappedProviderError {
  private readonly message: gostring;

  constructor(
    functionName: string,
    value: gostring,
    private readonly cause: GoError,
  ) {
    super(numberErrorType);
    this.message = GoString.fromText(`strconv.${functionName}: parsing ${JSON.stringify(value.text())}: ${cause.Error().text()}`);
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
    const message = this.message.text();
    return verb === "q" ? JSON.stringify(message) : message;
  }
}

export function rangeError(functionName: string, value: gostring): NumberError {
  return new NumberError(functionName, value, ErrRange);
}

export function syntaxError(functionName: string, value: gostring): NumberError {
  return new NumberError(functionName, value, ErrSyntax);
}
