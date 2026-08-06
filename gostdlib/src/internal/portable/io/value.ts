import { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";

export abstract class ProviderInterfaceValue extends GoInterfaceValue {
  readonly $go$methods: ReadonlySet<object> = new Set<object>();
  readonly $go$formatString: boolean = false;

	protected constructor(
		readonly $go$type: { readonly comparable: boolean },
	) {
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

  $go$format(_verb: string, _flags: string, _precision: number | undefined): string {
    return GoPanic.raiseRuntime("provider value has no formatting contract");
  }
}
