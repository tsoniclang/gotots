import assert from "node:assert/strict";
import test from "node:test";

import type { CancelCauseFunc } from "../src/context.js";
import type { WalkDirFunc } from "../src/io/fs.js";

interface CanonicalFailure {
  Error(): Promise<string>;
}

interface CanonicalEntry {
  Name(): Promise<string>;
}

test("provider callable value facets accept canonical generated signatures", async () => {
  const cancel: NonNullable<CancelCauseFunc<
    ((cause: CanonicalFailure | undefined) => Promise<void>) | undefined
  >> = async () => {};
  const walk: NonNullable<WalkDirFunc<
    ((
      path: string,
      entry: CanonicalEntry | undefined,
      failure: CanonicalFailure | undefined,
    ) => Promise<CanonicalFailure | undefined>) | undefined
  >> = async (_path, _entry, failure) => failure;

  await cancel(undefined);
  assert.equal(await walk(".", undefined, undefined), undefined);
});
