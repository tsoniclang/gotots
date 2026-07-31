import assert from "node:assert/strict";
import test from "node:test";

import { GoMap } from "@gotots/runtime/map.js";
import { GoPanic } from "@gotots/runtime/panic.js";

import { Clone } from "../src/maps.js";

test("maps.Clone exposes its missing construction capability explicitly", () => {
  const source = GoMap.make<string, number>(0, 1, [["key", 1]]);
  assert.throws(() => Clone(source), (failure): boolean => {
    assert.ok(failure instanceof GoPanic);
    assert.match(
      failure.value.$go$format("v", "", undefined),
      /maps\.Clone requires a generated map-construction capability/,
    );
    return true;
  });
});
