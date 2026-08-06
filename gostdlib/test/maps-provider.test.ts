import assert from "node:assert/strict";
import test from "node:test";

import { GoMap, type GoMapValue } from "@gotots/runtime/map.js";

import {
  MapsCloneKernel as Clone,
} from "../src/internal/facets/generic-maps-kernel.js";

test("maps.Clone consumes typed construction and assignment capabilities", () => {
  const source: GoMapValue<string, number> = GoMap.make<string, number>(
    0,
    1,
    [["key", 1]],
  );
  const identity = <K, V>(value: GoMapValue<K, V>): GoMapValue<K, V> => value;
  const copy = <T>(value: T): T => value;
  const clone = Clone(
    identity,
    identity,
    copy,
    copy,
    (zero): GoMapValue<string, number> => GoMap.make(zero, 0, []),
    (): number => 0,
    source,
  );
  assert.notEqual(clone, source);
  assert.deepEqual(clone.lookupOk("key"), [1, true]);
  clone.store("key", 2);
  assert.deepEqual(source.lookupOk("key"), [1, true]);
});
