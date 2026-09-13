import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";

export class Pool {
  readonly #values: Array<GoInterfaceValue | undefined> = [];
  #used = false;

  constructor(
    public New: (() => GoInterfaceValue | undefined) | undefined = undefined,
  ) {}

  static $copy(source: Pool): Pool {
    const target = new Pool();
    Pool.$assign(target, source);
    return target;
  }

  static $assign(target: Pool, source: Pool): void {
    if (target === source) return;
    if (source.#used) {
      GoPanic.raiseRuntime("sync.Pool must not be copied after first use");
    }
    target.New = source.New;
    target.#values.splice(0);
    target.#used = false;
  }

  static Get(receiver: Pool | undefined): GoInterfaceValue | undefined {
    if (receiver === undefined) {
      GoPanic.raiseRuntime("Pool.Get called with nil receiver");
    }
    receiver.#used = true;
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
    receiver.#used = true;
    if (x !== undefined) {
      receiver.#values.push(x);
    }
  }
}
