import { GoString } from "@gotots/runtime/string-value.js";
import assert from "node:assert/strict";
import test from "node:test";

import { Listen } from "../src/net.js";

test("net.Listen reports the unimplemented host boundary", (): void => {
  const [listener, failure] = Listen(GoString.fromText("unix"), GoString.fromText("/tmp/gotots.sock"));
  assert.equal(listener, undefined);
  assert.match(failure?.Error().text() ?? "", /net\.Listen unix/u);
});
