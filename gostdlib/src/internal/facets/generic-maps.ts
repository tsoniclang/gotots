import type { GoMapValue } from "@gotots/runtime/map.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { bool } from "@gotots/runtime/scalars.js";

import { Seq } from "../../iter.js";
import {
  KeysCooperative,
  ValuesCooperative,
} from "../portable/maps/cooperative.js";

type CooperativeSequence<T> = Seq<
  T,
  ((
    yieldValue: ((value: T) => Promise<bool>) | undefined,
  ) => Promise<void>) | undefined
>;

export function MapsKeysCooperative<
  MapType extends GoMapValue<Key, Value>,
  Key,
  Value,
>(
  source: MapType,
  _recovery?: GoRecovery,
): CooperativeSequence<Key> {
  return KeysCooperative(source);
}

export function MapsKeysFullyCooperative<
  MapType extends GoMapValue<Key, Value>,
  Key,
  Value,
>(
  source: MapType,
  _recovery?: GoRecovery,
): CooperativeSequence<Key> {
  return KeysCooperative(source);
}

export function MapsValuesCooperative<
  MapType extends GoMapValue<Key, Value>,
  Key,
  Value,
>(
  source: MapType,
  _recovery?: GoRecovery,
): CooperativeSequence<Value> {
  return ValuesCooperative(source);
}

export function MapsValuesFullyCooperative<
  MapType extends GoMapValue<Key, Value>,
  Key,
  Value,
>(
  source: MapType,
  _recovery?: GoRecovery,
): CooperativeSequence<Value> {
  return ValuesCooperative(source);
}
