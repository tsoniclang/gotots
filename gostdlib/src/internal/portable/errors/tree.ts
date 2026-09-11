import {
  GoErrorMethodToken,
  type GoError,
} from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import { GoString } from "@gotots/runtime/string-value.js";

import { ProviderInterfaceValue } from "../io/value.js";

export abstract class WrappedProviderError extends ProviderInterfaceValue implements GoError {
  override readonly $go$methods: ReadonlySet<object> = new Set<object>([
    GoErrorMethodToken,
  ]);

  protected constructor(typeIdentity: { readonly comparable: boolean }) {
    super(typeIdentity);
  }

  abstract Error(): GoString;

  abstract Unwrap(): GoError | undefined;

  override $go$format(verb: string, _flags: string, _precision: number | undefined): string {
    if (verb === "T") {
      return "*fmt.wrapError";
    }
    const message = this.Error().text();
    return verb === "q" ? JSON.stringify(message) : message;
  }
}

const messageWrappedErrorType = Object.freeze({ comparable: true });

export class MessageWrappedError extends WrappedProviderError {
  readonly #message: GoString;

  constructor(
    message: string,
    private readonly cause: GoError,
  ) {
    super(messageWrappedErrorType);
    this.#message = GoString.fromText(message);
  }

  Error(): GoString {
    return this.#message;
  }

  Unwrap(): GoError {
    return this.cause;
  }
}

const messageWrappedErrorsType = Object.freeze({ comparable: true });

export class MessageWrappedErrors extends ProviderInterfaceValue implements GoError {
  readonly #message: GoString;
  override readonly $go$methods: ReadonlySet<object> = new Set<object>([
    GoErrorMethodToken,
  ]);

  constructor(
    message: string,
    private readonly causes: readonly GoError[],
  ) {
    super(messageWrappedErrorsType);
    this.#message = GoString.fromText(message);
  }

  Error(): GoString {
    return this.#message;
  }

  Unwrap(): RuntimeSlice<GoError | undefined> {
    return RuntimeSlice.literal(this.causes.slice());
  }

  override $go$format(verb: string, _flags: string, _precision: number | undefined): string {
    if (verb === "T") {
      return "*fmt.wrapErrors";
    }
    const message = this.#message.text();
    return verb === "q" ? JSON.stringify(message) : message;
  }
}
