import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoMapHash } from "@gotots/runtime/map.js";
import { GoPanic } from "@gotots/runtime/panic.js";

import { goInterfaceEqual } from "../../runtime/interface.js";

let nextConditionIdentity = 1;

export interface Locker extends GoInterfaceValue {
  Lock(): void;
  Unlock(): void;
}

export class Cond {
  readonly #identity = nextConditionIdentity++;
  #checkerIdentity = 0;

  constructor(readonly L: Locker | undefined = undefined) {}

  static $copy(source: Cond): Cond {
    const result = new Cond(source.L);
    result.#checkerIdentity = source.#checkerIdentity;
    return result;
  }

  static $equal(left: Cond, right: Cond): boolean {
    return goInterfaceEqual(left.L, right.L) &&
      left.#checkerIdentity === right.#checkerIdentity;
  }

  static $hash(source: Cond): number {
    let hash = source.L === undefined ? 0 : source.L.$go$hash();
    hash = GoMapHash.mix(hash, GoMapHash.number(source.#checkerIdentity));
    return hash;
  }

  static Wait(receiver: Cond | undefined): void {
    const condition = requireCond(receiver);
    condition.#check();
    requireLocker(condition.L);
    GoPanic.raiseRuntime(
      "sync: Cond.Wait would block under serial execution",
    );
  }

  static Signal(receiver: Cond | undefined): void {
    const condition = requireCond(receiver);
    condition.#check();
  }

  static Broadcast(receiver: Cond | undefined): void {
    const condition = requireCond(receiver);
    condition.#check();
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
