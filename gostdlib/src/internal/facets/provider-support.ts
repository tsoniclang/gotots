import type {
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";

export interface InterfaceGuard<Value extends GoInterfaceValue> {
  (value: GoInterfaceValue | undefined): value is Value;
}

export interface InterfaceContract extends ReadonlyArray<object> {}

export interface FromProviderBridge<
  Provider extends GoInterfaceValue,
  Canonical extends GoInterfaceValue,
> {
  $from(value: Provider | undefined): Canonical | undefined;
}
