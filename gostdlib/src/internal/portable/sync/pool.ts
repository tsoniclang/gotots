import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";

export class Pool {
  readonly #values: Array<GoInterfaceValue | undefined> = [];

  constructor(
    public New: (() => GoInterfaceValue | undefined) | undefined = undefined,
  ) {}

  static Get(receiver: Pool | undefined): GoInterfaceValue | undefined {
    if (receiver === undefined) {
      GoPanic.raiseRuntime("Pool.Get called with nil receiver");
    }
    if (receiver.#values.length !== 0) {
      return receiver.#values.pop();
    }
    return receiver.New?.();
  }

  static Put(
    receiver: Pool | undefined,
    x: GoInterfaceValue | undefined,
  ): void {
    if (receiver === undefined) {
      GoPanic.raiseRuntime("Pool.Put called with nil receiver");
    }
    if (x !== undefined) {
      receiver.#values.push(x);
    }
  }
}
