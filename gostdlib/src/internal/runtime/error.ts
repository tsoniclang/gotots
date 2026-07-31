import {
  GoErrorMethodToken,
  type GoError,
  GoInterfaceValue,
  GoRuntimeErrorMethodToken,
} from "@gotots/runtime/interface-value.js";

export class ProviderError extends GoInterfaceValue {
  readonly $go$type: object = ProviderError;
  readonly $go$methods: ReadonlySet<object>;
  readonly $go$formatString = false;

  constructor(private readonly message: string, runtime = false) {
    super();
    this.$go$methods = runtime
      ? new Set<object>([GoErrorMethodToken, GoRuntimeErrorMethodToken])
      : new Set<object>([GoErrorMethodToken]);
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
      return JSON.stringify(this.message);
    }
    return this.message;
  }

  Error(): string {
    return this.message;
  }
}

export function providerError(failure: object): ProviderError {
  return new ProviderError(failure instanceof Error ? failure.message : String(failure));
}

export function isGoError(value: GoInterfaceValue): value is GoError {
  return value.$go$implements([GoErrorMethodToken]);
}
