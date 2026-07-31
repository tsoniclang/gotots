import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { bool } from "@gotots/runtime/scalars.js";

interface Entry {
  readonly key: GoInterfaceValue | undefined;
  value: GoInterfaceValue | undefined;
}

function equal(
  left: GoInterfaceValue | undefined,
  right: GoInterfaceValue | undefined,
): boolean {
  return left === right
    || (left !== undefined && right !== undefined && left.$go$equal(right));
}

export class Map {
  readonly #entries: Entry[] = [];

  static Clear(receiver: Map | undefined): void {
    Map.#require(receiver).#entries.splice(0);
  }

  static Delete(receiver: Map | undefined, key: GoInterfaceValue | undefined): void {
    const entries = Map.#require(receiver).#entries;
    const index = entries.findIndex((entry) => equal(entry.key, key));
    if (index >= 0) {
      entries.splice(index, 1);
    }
  }

  static Load(
    receiver: Map | undefined,
    key: GoInterfaceValue | undefined,
  ): [GoInterfaceValue | undefined, bool] {
    const entry = Map.#require(receiver).#entries.find((candidate) => equal(candidate.key, key));
    return entry === undefined ? [undefined, false] : [entry.value, true];
  }

  static LoadOrStore(
    receiver: Map | undefined,
    key: GoInterfaceValue | undefined,
    value: GoInterfaceValue | undefined,
  ): [GoInterfaceValue | undefined, bool] {
    const map = Map.#require(receiver);
    const entry = map.#entries.find((candidate) => equal(candidate.key, key));
    if (entry !== undefined) {
      return [entry.value, true];
    }
    map.#entries.push({ key, value });
    return [value, false];
  }

  static async Range(
    receiver: Map | undefined,
    f: ((
      key: GoInterfaceValue | undefined,
      value: GoInterfaceValue | undefined,
    ) => Promise<bool>) | undefined,
  ): Promise<void> {
    if (f === undefined) {
      GoPanic.raiseRuntime("sync.Map.Range called with nil function");
    }
    for (const entry of [...Map.#require(receiver).#entries]) {
      if (!await f(entry.key, entry.value)) {
        return;
      }
    }
  }

  static Store(
    receiver: Map | undefined,
    key: GoInterfaceValue | undefined,
    value: GoInterfaceValue | undefined,
  ): void {
    const map = Map.#require(receiver);
    const entry = map.#entries.find((candidate) => equal(candidate.key, key));
    if (entry === undefined) {
      map.#entries.push({ key, value });
    } else {
      entry.value = value;
    }
  }

  static #require(receiver: Map | undefined): Map {
    if (receiver === undefined) {
      GoPanic.raiseRuntime("sync.Map method called with nil receiver");
    }
    return receiver;
  }
}
