import assert from "node:assert/strict";
import test from "node:test";
import { ENOTDIR } from "../src/syscall.js";

test("syscall ENOTDIR preserves the selected Linux errno identity", () => {
  assert.equal(ENOTDIR.value, 20n);
  assert.equal(ENOTDIR.Error(), "not a directory");
  assert.equal(ENOTDIR, ENOTDIR);
});
