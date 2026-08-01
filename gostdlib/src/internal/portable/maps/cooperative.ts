import { GoPanic } from "@gotots/runtime/panic.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { GoMapValue } from "@gotots/runtime/map.js";
import type { bool } from "@gotots/runtime/scalars.js";

import { Seq } from "../iter/sequence.js";

type CooperativeSequence<T> = Seq<
  T,
  ((
    yieldValue: ((value: T, recovery?: GoRecovery) => Promise<bool>) | undefined,
    recovery?: GoRecovery,
  ) => Promise<void>) | undefined
>;
type CopyValue<T> = (value: T) => T;

export function KeysCooperative<K, V>(
  copyKey: CopyValue<K>,
  source: GoMapValue<K, V>,
): CooperativeSequence<K> {
  return new Seq<K, CooperativeSequence<K>["value"]>(
    async (yieldValue): Promise<void> => {
      if (yieldValue === undefined) {
        GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
      }
      for (const key of source.keys()) {
        if (!await yieldValue(copyKey(key))) {
          return;
        }
      }
    },
  );
}

export function ValuesCooperative<K, V>(
  copyValue: CopyValue<V>,
  source: GoMapValue<K, V>,
): CooperativeSequence<V> {
  return new Seq<V, CooperativeSequence<V>["value"]>(
    async (yieldValue): Promise<void> => {
      if (yieldValue === undefined) {
        GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
      }
      for (const key of source.keys()) {
        if (!await yieldValue(copyValue(source.lookup(key)))) {
          return;
        }
      }
    },
  );
}
