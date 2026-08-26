import { GoMapHash } from "@gotots/runtime/map.js";
import { GoPanic } from "@gotots/runtime/panic.js";

function nilReceiver(name: string): never {
  return GoPanic.raiseRuntime(`${name} called with nil receiver`);
}

export class Mutex {
  #locked = false;

  static $copy(source: Mutex): Mutex {
    const result = new Mutex();
    result.#locked = source.#locked;
    return result;
  }

  static $assign(target: Mutex, source: Mutex): void {
    target.#locked = source.#locked;
  }

  static $equal(left: Mutex, right: Mutex): boolean {
    return left.#locked === right.#locked;
  }

  static $hash(source: Mutex): number {
    return GoMapHash.boolean(source.#locked);
  }

  static Lock(receiver: Mutex | undefined): void {
    if (receiver === undefined) {
      nilReceiver("Mutex.Lock");
    }
    if (receiver.#locked) {
      GoPanic.raiseRuntime(
        "sync: Mutex.Lock would block under serial execution",
      );
    }
    receiver.#locked = true;
  }

  static Unlock(receiver: Mutex | undefined): void {
    if (receiver === undefined) {
      nilReceiver("Mutex.Unlock");
    }
    if (!receiver.#locked) {
      GoPanic.raiseRuntime("sync: unlock of unlocked mutex");
    }
    receiver.#locked = false;
  }
}

export class RWMutex {
  #readers = 0;
  #writer = false;

  static $copy(source: RWMutex): RWMutex {
    const result = new RWMutex();
    result.#readers = source.#readers;
    result.#writer = source.#writer;
    return result;
  }

  static $equal(left: RWMutex, right: RWMutex): boolean {
    if (
      left.#readers !== right.#readers ||
      left.#writer !== right.#writer
    ) {
      return false;
    }
    return true;
  }

  static $hash(source: RWMutex): number {
    let hash = GoMapHash.number(source.#readers);
    hash = GoMapHash.mix(hash, GoMapHash.boolean(source.#writer));
    return hash;
  }

  static Lock(receiver: RWMutex | undefined): void {
    if (receiver === undefined) {
      nilReceiver("RWMutex.Lock");
    }
    if (receiver.#writer || receiver.#readers !== 0) {
      GoPanic.raiseRuntime(
        "sync: RWMutex.Lock would block under serial execution",
      );
    }
    receiver.#writer = true;
  }

  static Unlock(receiver: RWMutex | undefined): void {
    if (receiver === undefined) {
      nilReceiver("RWMutex.Unlock");
    }
    if (!receiver.#writer) {
      GoPanic.raiseRuntime("sync: unlock of unlocked RWMutex");
    }
    receiver.#writer = false;
  }

  static RLock(receiver: RWMutex | undefined): void {
    if (receiver === undefined) {
      nilReceiver("RWMutex.RLock");
    }
    if (receiver.#writer) {
      GoPanic.raiseRuntime(
        "sync: RWMutex.RLock would block under serial execution",
      );
    }
    receiver.#readers += 1;
  }

  static RUnlock(receiver: RWMutex | undefined): void {
    if (receiver === undefined) {
      nilReceiver("RWMutex.RUnlock");
    }
    if (receiver.#readers === 0) {
      GoPanic.raiseRuntime("sync: RUnlock of unlocked RWMutex");
    }
    receiver.#readers -= 1;
  }
}
