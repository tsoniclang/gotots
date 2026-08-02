import { GoPanic } from "@gotots/runtime/panic.js";
import type { bool } from "@gotots/runtime/scalars.js";

import type { Seq } from "./internal/portable/iter/sequence.js";

type Equality<Left, Right> = ((left: Left, right: Right) => bool) | undefined;

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

export function EqualFunc<LeftMap, RightMap, Key, Left, Right>(
  left: LeftMap,
  right: RightMap,
  equal: Equality<Left, Right>,
): bool {
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
