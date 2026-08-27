import type { int } from "@gotots/gostdlib/internal/scalars.js";
import { GoMapHash } from "@gotots/runtime/map.js";
import { GoPanic } from "@gotots/runtime/panic.js";

export class WaitGroup {
  #count: int = 0n;

  static $copy(source: WaitGroup): WaitGroup {
    const result = new WaitGroup();
    result.#count = source.#count;
    return result;
  }

  static $equal(left: WaitGroup, right: WaitGroup): boolean {
    return left.#count === right.#count;
  }

  static $hash(source: WaitGroup): number {
    return GoMapHash.bigint(source.#count);
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
  }

  static Done(receiver: WaitGroup | undefined): void {
    WaitGroup.Add(receiver, -1n);
  }

  static Go(
    receiver: WaitGroup | undefined,
    f: (() => void) | undefined,
  ): void {
    if (receiver === undefined) {
      GoPanic.raiseRuntime("WaitGroup.Go called with nil receiver");
    }
    WaitGroup.Add(receiver, 1n);
    try {
      if (f === undefined) {
        GoPanic.raiseRuntime("sync.WaitGroup.Go called with nil function");
      }
      f();
    } finally {
      WaitGroup.Done(receiver);
    }
  }

  static Wait(receiver: WaitGroup | undefined): void {
    if (receiver === undefined) {
      GoPanic.raiseRuntime("WaitGroup.Wait called with nil receiver");
    }
    if (receiver.#count !== 0n) {
      GoPanic.raiseRuntime(
        "sync: WaitGroup.Wait would block under serial execution",
      );
    }
  }
}
