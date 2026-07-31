import assert from "node:assert/strict";
import test from "node:test";

import {
  Contains,
  HasPrefix,
  HasSuffix,
  Index,
} from "../src/strings.js";

test("strings search operations preserve Go results", () => {
  assert.equal(Contains("gopher", "ph"), true);
  assert.equal(Contains("gopher", "xy"), false);
  assert.equal(HasPrefix("gopher", "go"), true);
  assert.equal(HasSuffix("gopher", "her"), true);
  assert.equal(Index("gopher", "ph"), 2);
  assert.equal(Index("gopher", "xy"), -1);
});
