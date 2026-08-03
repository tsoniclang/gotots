import type { Awaitable, int64 } from "@gotots/runtime/scalars.js";
import { GoPanic } from "@gotots/runtime/panic.js";

export class WaitGroup {
  #count = 0;
  readonly #waiters: Array<() => void> = [];

  static Add(receiver: WaitGroup | undefined, delta: int64): void {
    if (receiver === undefined) {
      GoPanic.raiseRuntime("WaitGroup.Add called with nil receiver");
    }
    const next = receiver.#count + delta;
    if (next < 0) {
      GoPanic.raiseRuntime("sync: negative WaitGroup counter");
    }
    receiver.#count = next;
    if (next === 0) {
      for (const resume of receiver.#waiters.splice(0)) {
        resume();
      }
    }
  }

  static Done(receiver: WaitGroup | undefined): void {
    WaitGroup.Add(receiver, -1);
  }

  static Go(
    receiver: WaitGroup | undefined,
    f: (() => Awaitable<void>) | undefined,
  ): void {
    WaitGroup.Add(receiver, 1);
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
    if (receiver.#count === 0) {
      return Promise.resolve();
    }
    return new Promise<void>((resolve) => receiver.#waiters.push(resolve));
  }
}
