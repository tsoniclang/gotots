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
    yieldValue: ((value: T, recovery?: GoRecovery) => Promise<bool>) | undefined,
    recovery?: GoRecovery,
  ) => Promise<void>) | undefined
>;
type Convert<Source, Target> = (value: Source) => Target;
type CopyValue<T> = (value: T) => T;

export function MapsKeysCooperative<
  MapType,
  Key,
  Value,
>(
  sourceMap: Convert<MapType, GoMapValue<Key, Value>>,
  copyKey: CopyValue<Key>,
  source: MapType,
  _recovery?: GoRecovery,
): CooperativeSequence<Key> {
  return KeysCooperative(copyKey, sourceMap(source));
}

export function MapsKeysFullyCooperative<
  MapType,
  Key,
  Value,
>(
  sourceMap: Convert<MapType, GoMapValue<Key, Value>>,
  copyKey: CopyValue<Key>,
  source: MapType,
  _recovery?: GoRecovery,
): CooperativeSequence<Key> {
  return KeysCooperative(copyKey, sourceMap(source));
}

export function MapsValuesCooperative<
  MapType,
  Key,
  Value,
>(
  sourceMap: Convert<MapType, GoMapValue<Key, Value>>,
  copyValue: CopyValue<Value>,
  source: MapType,
  _recovery?: GoRecovery,
): CooperativeSequence<Value> {
  return ValuesCooperative(copyValue, sourceMap(source));
}

export function MapsValuesFullyCooperative<
  MapType,
  Key,
  Value,
>(
  sourceMap: Convert<MapType, GoMapValue<Key, Value>>,
  copyValue: CopyValue<Value>,
  source: MapType,
  _recovery?: GoRecovery,
): CooperativeSequence<Value> {
  return ValuesCooperative(copyValue, sourceMap(source));
}
