import {
  GoErrorMethodToken,
  type GoError,
  GoInterfaceValue,
  GoRuntimeErrorMethodToken,
} from "@gotots/runtime/interface-value.js";
import { GoString } from "@gotots/runtime/string-value.js";

import { fromHostString } from "../portable/utf8/codec.js";

export class ProviderError extends GoInterfaceValue {
  static readonly comparable = true;
  readonly $go$type: { readonly comparable: boolean } = ProviderError;
  readonly $go$methods: ReadonlySet<object>;
  readonly $go$formatString = false;

  readonly #message: GoString;

  constructor(message: GoString, runtime = false) {
    super();
    this.#message = message;
    this.$go$methods = runtime
      ? new Set<object>([GoErrorMethodToken, GoRuntimeErrorMethodToken])
      : new Set<object>([GoErrorMethodToken]);
  }

  static fromText(message: string, runtime = false): ProviderError {
    return new ProviderError(GoString.fromText(message), runtime);
  }

  $go$implements(contract: readonly object[]): boolean {
    return contract.every((token: object): boolean => this.$go$methods.has(token));
  }

  $go$equal(other: GoInterfaceValue): boolean {
    return this === other;
  }

  $go$hash(): number {
    return 0;
  }

  $go$format(verb: string, _flags: string, _precision: number | undefined): string {
    if (verb === "T") {
      return "*errors.errorString";
    }
    if (verb === "q") {
      return JSON.stringify(this.#message.text());
    }
    return this.#message.text();
  }

  Error(): GoString {
    return this.#message;
  }
}

export function providerError(failure: object): ProviderError {
  return new ProviderError(fromHostString(failure instanceof Error ? failure.message : String(failure)));
}

export function isGoError(value: GoInterfaceValue): value is GoError {
  return value.$go$implements([GoErrorMethodToken]);
}
