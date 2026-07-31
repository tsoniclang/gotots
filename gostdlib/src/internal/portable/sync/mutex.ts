import { GoPanic } from "@gotots/runtime/panic.js";

type Waiter = () => void;

function nilReceiver(name: string): never {
  return GoPanic.raiseRuntime(`${name} called with nil receiver`);
}

export class Mutex {
  #locked = false;
  readonly #waiters: Waiter[] = [];

  static async Lock(receiver: Mutex | undefined): Promise<void> {
    if (receiver === undefined) {
      nilReceiver("Mutex.Lock");
    }
    if (!receiver.#locked) {
      receiver.#locked = true;
      return;
    }
    await new Promise<void>((resolve) => receiver.#waiters.push(resolve));
  }

  static Unlock(receiver: Mutex | undefined): void {
    if (receiver === undefined) {
      nilReceiver("Mutex.Unlock");
    }
    if (!receiver.#locked) {
      GoPanic.raiseRuntime("sync: unlock of unlocked mutex");
    }
    const waiter = receiver.#waiters.shift();
    if (waiter === undefined) {
      receiver.#locked = false;
    } else {
      waiter();
    }
  }
}

type LockWaiter =
  | { readonly kind: "reader"; readonly resume: Waiter }
  | { readonly kind: "writer"; readonly resume: Waiter };

export class RWMutex {
  #readers = 0;
  #writer = false;
  readonly #waiters: LockWaiter[] = [];

  static async Lock(receiver: RWMutex | undefined): Promise<void> {
    if (receiver === undefined) {
      nilReceiver("RWMutex.Lock");
    }
    if (!receiver.#writer && receiver.#readers === 0) {
      receiver.#writer = true;
      return;
    }
    await new Promise<void>((resolve) => {
      receiver.#waiters.push({ kind: "writer", resume: resolve });
    });
  }

  static Unlock(receiver: RWMutex | undefined): void {
    if (receiver === undefined) {
      nilReceiver("RWMutex.Unlock");
    }
    if (!receiver.#writer) {
      GoPanic.raiseRuntime("sync: unlock of unlocked RWMutex");
    }
    receiver.#writer = false;
    receiver.#wake();
  }

  static async RLock(receiver: RWMutex | undefined): Promise<void> {
    if (receiver === undefined) {
      nilReceiver("RWMutex.RLock");
    }
    const writerWaiting = receiver.#waiters.some((waiter) => waiter.kind === "writer");
    if (!receiver.#writer && !writerWaiting) {
      receiver.#readers += 1;
      return;
    }
    await new Promise<void>((resolve) => {
      receiver.#waiters.push({ kind: "reader", resume: resolve });
    });
  }

  static RUnlock(receiver: RWMutex | undefined): void {
    if (receiver === undefined) {
      nilReceiver("RWMutex.RUnlock");
    }
    if (receiver.#readers === 0) {
      GoPanic.raiseRuntime("sync: RUnlock of unlocked RWMutex");
    }
    receiver.#readers -= 1;
    if (receiver.#readers === 0) {
      receiver.#wake();
    }
  }

  #wake(): void {
    if (this.#writer || this.#readers !== 0) {
      return;
    }
    const first = this.#waiters.shift();
    if (first === undefined) {
      return;
    }
    if (first.kind === "writer") {
      this.#writer = true;
      first.resume();
      return;
    }
    this.#readers = 1;
    first.resume();
    while (this.#waiters[0]?.kind === "reader") {
      const reader = this.#waiters.shift();
      if (reader?.kind === "reader") {
        this.#readers += 1;
        reader.resume();
      }
    }
  }
}
