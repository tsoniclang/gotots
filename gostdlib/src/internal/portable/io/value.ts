import { GoInterfaceValue } from "@gotots/runtime/interface-value.js";

export abstract class ProviderInterfaceValue extends GoInterfaceValue {
  readonly $go$methods: ReadonlySet<object> = new Set<object>();

  protected constructor(readonly $go$type: object) {
    super();
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
}
