import assert from "node:assert/strict";
import test from "node:test";

import { Listen } from "../src/net.js";

test("net.Listen reports the unimplemented host boundary", (): void => {
  const [listener, failure] = Listen("unix", "/tmp/gotots.sock");
  assert.equal(listener, undefined);
  assert.match(failure?.Error() ?? "", /net\.Listen unix/u);
});
