import { GoPanic } from "@gotots/runtime/panic.js";
import type { GoMapValue } from "@gotots/runtime/map.js";
import type { bool } from "@gotots/runtime/scalars.js";

import { Seq } from "../iter/sequence.js";

type CooperativeSequence<T> = Seq<
  T,
  ((
    yieldValue: ((value: T) => Promise<bool>) | undefined,
  ) => Promise<void>) | undefined
>;

export function KeysCooperative<K, V>(
  source: GoMapValue<K, V>,
): CooperativeSequence<K> {
  return new Seq<K, CooperativeSequence<K>["value"]>(
    async (yieldValue): Promise<void> => {
      if (yieldValue === undefined) {
        GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
      }
      for (const key of source.keys()) {
        if (!await yieldValue(key)) {
          return;
        }
      }
    },
  );
}

export function ValuesCooperative<K, V>(
  source: GoMapValue<K, V>,
): CooperativeSequence<V> {
  return new Seq<V, CooperativeSequence<V>["value"]>(
    async (yieldValue): Promise<void> => {
      if (yieldValue === undefined) {
        GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
      }
      for (const key of source.keys()) {
        if (!await yieldValue(source.lookup(key))) {
          return;
        }
      }
    },
  );
}
