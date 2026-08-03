import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { Awaitable } from "@gotots/runtime/scalars.js";

type Waiter = () => void;

export interface Locker extends GoInterfaceValue {
  Lock(): Awaitable<void>;
  Unlock(): Awaitable<void>;
}

export class Cond {
  readonly #waiters: Waiter[] = [];

  constructor(readonly L: Locker | undefined = undefined) {}

  static async Wait(receiver: Cond | undefined): Promise<void> {
    const condition = requireCond(receiver);
    const locker = requireLocker(condition.L);
    const waiting = new Promise<void>((resolve) => {
      condition.#waiters.push(resolve);
    });
    await locker.Unlock();
    await waiting;
    await locker.Lock();
  }

  static Signal(receiver: Cond | undefined): void {
    const condition = requireCond(receiver);
    condition.#waiters.shift()?.();
  }

  static Broadcast(receiver: Cond | undefined): void {
    const condition = requireCond(receiver);
    const waiters = condition.#waiters.splice(0);
    for (const resume of waiters) {
      resume();
    }
  }
}

export function NewCond(locker: Locker | undefined): Cond {
  return new Cond(locker);
}

function requireCond(receiver: Cond | undefined): Cond {
  if (receiver === undefined) {
    return GoPanic.raiseRuntime("Cond method called with nil receiver");
  }
  return receiver;
}

function requireLocker(locker: Locker | undefined): Locker {
  if (locker === undefined) {
    return GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return locker;
}
