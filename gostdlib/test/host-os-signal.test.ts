import assert from "node:assert/strict";
import test from "node:test";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import { Background } from "../src/context.js";
import { NotifyContext } from "../src/os/signal.js";
import type { Signal as OsSignal } from "../src/os.js";
import { SIGINT } from "../src/syscall.js";

test("NotifyContext owns and releases its Node signal listener", () => {
  const listenersBefore = process.listenerCount("SIGINT");
  const [context, stop] = NotifyContext(
    Background(),
    RuntimeSlice.literal<OsSignal | undefined>([SIGINT]),
  );
  assert.ok(context !== undefined);
  assert.equal(process.listenerCount("SIGINT"), listenersBefore + 1);
  stop();
  assert.equal(process.listenerCount("SIGINT"), listenersBefore);
  assert.notEqual(context.Err(), undefined);
});
