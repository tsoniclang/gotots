import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoMapHash } from "@gotots/runtime/map.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { Awaitable } from "@gotots/gostdlib/internal/scalars.js";

import { goInterfaceEqual } from "../../runtime/interface.js";

type Waiter = () => void;
let nextConditionIdentity = 1;

export interface Locker extends GoInterfaceValue {
  Lock(): Awaitable<void>;
  Unlock(): Awaitable<void>;
}

export class Cond {
  readonly #identity = nextConditionIdentity++;
  #checkerIdentity = 0;
  readonly #waiters: Waiter[] = [];

  constructor(readonly L: Locker | undefined = undefined) {}

  static $copy(source: Cond): Cond {
    const result = new Cond(source.L);
    result.#checkerIdentity = source.#checkerIdentity;
    result.#waiters.push(...source.#waiters);
    return result;
  }

  static $equal(left: Cond, right: Cond): boolean {
    return goInterfaceEqual(left.L, right.L) &&
      left.#checkerIdentity === right.#checkerIdentity &&
      left.#waiters.length === right.#waiters.length;
  }

  static $hash(source: Cond): number {
    let hash = source.L === undefined ? 0 : source.L.$go$hash();
    hash = GoMapHash.mix(hash, GoMapHash.number(source.#checkerIdentity));
    hash = GoMapHash.mix(hash, GoMapHash.number(source.#waiters.length));
    return hash;
  }

  static async Wait(receiver: Cond | undefined): Promise<void> {
    const condition = requireCond(receiver);
    condition.#check();
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
    condition.#check();
    condition.#waiters.shift()?.();
  }

  static Broadcast(receiver: Cond | undefined): void {
    const condition = requireCond(receiver);
    condition.#check();
    const waiters = condition.#waiters.splice(0);
    for (const resume of waiters) {
      resume();
    }
  }

  #check(): void {
    if (this.#checkerIdentity === 0) {
      this.#checkerIdentity = this.#identity;
      return;
    }
    if (this.#checkerIdentity !== this.#identity) {
      GoPanic.raiseRuntime("sync.Cond is copied");
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
