import {
  GoErrorMethodToken,
  type GoError,
} from "@gotots/runtime/interface-value.js";

import { ProviderInterfaceValue } from "../io/value.js";

export abstract class WrappedProviderError extends ProviderInterfaceValue implements GoError {
  override readonly $go$methods: ReadonlySet<object> = new Set<object>([
    GoErrorMethodToken,
  ]);

	protected constructor(typeIdentity: { readonly comparable: boolean }) {
    super(typeIdentity);
  }

  abstract Error(): string;

  abstract Unwrap(): GoError | undefined;

  override $go$format(verb: string, _flags: string, _precision: number | undefined): string {
    if (verb === "T") {
      return "*fmt.wrapError";
    }
    const message = this.Error();
    return verb === "q" ? JSON.stringify(message) : message;
  }
}

const messageWrappedErrorType = Object.freeze({ comparable: true });

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

const messageWrappedErrorsType = Object.freeze({ comparable: true });

export class MessageWrappedErrors extends ProviderInterfaceValue implements GoError {
  override readonly $go$methods: ReadonlySet<object> = new Set<object>([
    GoErrorMethodToken,
  ]);

  constructor(
    private readonly message: string,
    private readonly causes: readonly GoError[],
  ) {
    super(messageWrappedErrorsType);
  }

  Error(): string {
    return this.message;
  }

  UnwrapAll(): readonly GoError[] {
    return this.causes.slice();
  }

  override $go$format(verb: string, _flags: string, _precision: number | undefined): string {
    if (verb === "T") {
      return "*fmt.wrapErrors";
    }
    return verb === "q" ? JSON.stringify(this.message) : this.message;
  }
}
