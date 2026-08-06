import type { Awaitable, int } from "@gotots/gostdlib/internal/scalars.js";
import { GoMapHash } from "@gotots/runtime/map.js";
import { GoPanic } from "@gotots/runtime/panic.js";

export class WaitGroup {
  #count: int = 0n;
  readonly #waiters: Array<() => void> = [];

  static $copy(source: WaitGroup): WaitGroup {
    const result = new WaitGroup();
    result.#count = source.#count;
    result.#waiters.push(...source.#waiters);
    return result;
  }

  static $equal(left: WaitGroup, right: WaitGroup): boolean {
    return left.#count === right.#count &&
      left.#waiters.length === right.#waiters.length;
  }

  static $hash(source: WaitGroup): number {
    let hash = GoMapHash.bigint(source.#count);
    hash = GoMapHash.mix(hash, GoMapHash.number(source.#waiters.length));
    return hash;
  }

  static Add(receiver: WaitGroup | undefined, delta: int): void {
    if (receiver === undefined) {
      GoPanic.raiseRuntime("WaitGroup.Add called with nil receiver");
    }
    const next = receiver.#count + delta;
    if (next < 0n) {
      GoPanic.raiseRuntime("sync: negative WaitGroup counter");
    }
    receiver.#count = next;
    if (next === 0n) {
      for (const resume of receiver.#waiters.splice(0)) {
        resume();
      }
    }
  }

  static Done(receiver: WaitGroup | undefined): void {
    WaitGroup.Add(receiver, -1n);
  }

  static Go(
    receiver: WaitGroup | undefined,
    f: (() => Awaitable<void>) | undefined,
  ): void {
    WaitGroup.Add(receiver, 1n);
    void (async (): Promise<void> => {
      try {
        if (f === undefined) {
          GoPanic.raiseRuntime("sync.WaitGroup.Go called with nil function");
        }
        await f();
      } finally {
        WaitGroup.Done(receiver);
      }
    })();
  }

  static Wait(receiver: WaitGroup | undefined): Promise<void> {
    if (receiver === undefined) {
      GoPanic.raiseRuntime("WaitGroup.Wait called with nil receiver");
    }
    if (receiver.#count === 0n) {
      return Promise.resolve();
    }
    return new Promise<void>((resolve) => receiver.#waiters.push(resolve));
  }
}
