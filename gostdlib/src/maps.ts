import { GoPanic } from "@gotots/runtime/panic.js";
import type { Awaitable, bool } from "@gotots/gostdlib/internal/scalars.js";

import type { Seq } from "./internal/portable/iter/sequence.js";

export function Clone<MapType, Key, Value>(source: MapType): MapType {
  return specializationRequired("maps.Clone");
}

export function Copy<TargetMap, SourceMap, Key, Value>(
  target: TargetMap,
  source: SourceMap,
): void {
  return specializationRequired("maps.Copy");
}

export function Equal<LeftMap, RightMap, Key, Value>(
  left: LeftMap,
  right: RightMap,
): bool {
  return specializationRequired("maps.Equal");
}

export async function EqualFunc<LeftMap, RightMap, Key, Left, Right>(
  left: LeftMap,
  right: RightMap,
  equal: ((left: Left, right: Right) => Awaitable<bool>) | undefined,
): Promise<bool> {
  return specializationRequired("maps.EqualFunc");
}

export function Keys<MapType, Key, Value>(source: MapType): Seq<Key> {
  return specializationRequired("maps.Keys");
}

export function Values<MapType, Key, Value>(source: MapType): Seq<Value> {
  return specializationRequired("maps.Values");
}

function specializationRequired(name: string): never {
  return GoPanic.raiseRuntime(`${name} requires a generated generic specialization`);
}
