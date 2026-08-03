import assert from "node:assert/strict";
import test from "node:test";

import type { CancelCauseFunc } from "../src/context.js";
import type { WalkDirFunc } from "../src/io/fs.js";

interface CanonicalFailure {
  Error(): string | Promise<string>;
}

interface CanonicalEntry {
  Name(): string | Promise<string>;
}

test("provider named callables keep source arity", async () => {
  const cancel: NonNullable<CancelCauseFunc> = async () => {};
  const walk: NonNullable<WalkDirFunc> = async (
    _path,
    _entry,
    failure,
  ) => failure;

  await cancel(undefined);
  assert.equal(await walk(".", undefined, undefined), undefined);
});
