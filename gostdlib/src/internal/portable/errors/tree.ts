import {
  GoErrorMethodToken,
  type GoError,
} from "@gotots/runtime/interface-value.js";

import { ProviderInterfaceValue } from "../io/value.js";

export abstract class WrappedProviderError extends ProviderInterfaceValue implements GoError {
  override readonly $go$methods: ReadonlySet<object> = new Set<object>([
    GoErrorMethodToken,
  ]);

  protected constructor(typeIdentity: object) {
    super(typeIdentity);
  }

  abstract Error(): string;

  abstract Unwrap(): GoError | undefined;
}

const messageWrappedErrorType = Object.freeze({});

export class MessageWrappedError extends WrappedProviderError {
  constructor(
    private readonly message: string,
    private readonly cause: GoError,
  ) {
    super(messageWrappedErrorType);
  }

  Error(): string {
    return this.message;
  }

  Unwrap(): GoError {
    return this.cause;
  }
}
