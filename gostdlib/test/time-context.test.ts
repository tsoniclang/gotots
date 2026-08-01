import assert from "node:assert/strict";
import test from "node:test";
import type { GoReceiveChannel } from "@gotots/runtime/channel.js";
import type { GoEmptyStruct } from "@gotots/runtime/struct.js";
import { ProviderError } from "../src/internal/runtime/error.js";
import {
  AfterFunc as ContextAfterFunc,
  Background,
  state,
  TODO,
  WithCancel,
  WithCancelCause,
  WithTimeout,
  WithValue,
} from "../src/context.js";
import {
  After,
  AfterFunc,
  Duration,
  Millisecond,
  NewTicker,
  NewTimer,
  Since,
  Second,
  Ticker,
  Now,
  Time,
  Timer,
  UnixMilli,
  Until,
} from "../src/time.js";

test("Duration and Time preserve arithmetic, ordering, formatting, and zero", () => {
  const duration = new Duration(1_250_000_000);
  assert.equal(duration.Nanoseconds(), 1_250_000_000);
  assert.equal(duration.Seconds(), 1.25);
  assert.equal(duration.String(), "1.25s");

  const epoch = UnixMilli(0);
  const later = epoch.Add(duration);
  assert.equal(epoch.Before(later), true);
  assert.equal(later.After(epoch), true);
  assert.equal(epoch.Equal(UnixMilli(0)), true);
  assert.equal(later.Sub(epoch).Nanoseconds(), duration.Nanoseconds());
  const localEpoch = new Date(0);
  const hour = localEpoch.getHours() % 12 || 12;
  const expectedTime = `${String(hour).padStart(2, "0")}:`
    + `${String(localEpoch.getMinutes()).padStart(2, "0")}:`
    + `${String(localEpoch.getSeconds()).padStart(2, "0")} `
    + `${localEpoch.getHours() < 12 ? "AM" : "PM"}`;
  assert.equal(UnixMilli(0).Format("03:04:05 PM"), expectedTime);
  assert.equal(new Time().IsZero(), true);
  assert.equal(UnixMilli(0).UnixMilli(), 0);
  assert.equal(UnixMilli(0).UnixNano(), 0);
  assert.match(UnixMilli(0).String(), /1970/u);
  assert.equal(Millisecond.Nanoseconds(), 1_000_000);
  assert.equal(Second.Nanoseconds(), 1_000_000_000);
  const now = Now();
  assert.equal(now.IsZero(), false);
  assert.ok(Since(now).Nanoseconds() >= 0);
  assert.ok(Until(now.Add(new Duration(1_000_000_000))).Nanoseconds() > 0);
});

test("Timer delivers once, supports reset, and reports stop state", async () => {
  const timer = NewTimer(new Duration(50_000_000));
  assert.equal(Timer.Stop(timer), true);
  assert.equal(Timer.Reset(timer, new Duration(1_000_000)), false);
  const [fired, ok] = await timer.C!.receive();
  assert.equal(ok, true);
  assert.equal(fired.IsZero(), false);
  assert.equal(Timer.Stop(timer), false);
});

test("Ticker produces values until stopped", async () => {
  const ticker = NewTicker(new Duration(1_000_000));
  const [, ok] = await ticker.C.receive();
  assert.equal(ok, true);
  Ticker.Stop(ticker);
});

test("After and AfterFunc schedule through the provider clock", async () => {
  const [, open] = await After(new Duration(1_000_000)).receive();
  assert.equal(open, true);

  let called = false;
  const timer = AfterFunc(new Duration(1_000_000), async () => {
    called = true;
  });
  await new Promise<void>((resolve) => setTimeout(resolve, 5));
  assert.equal(called, true);
  assert.equal(timer.C, undefined);
});

test("Context cancellation, causes, values, and deadlines propagate", async () => {
  const root = Background();
  const canonicalDoneOwner: {
    Done(): GoReceiveChannel<GoEmptyStruct> | undefined;
  } = root;
  assert.equal(canonicalDoneOwner.Done(), undefined);
  assert.notEqual(TODO(), root);
  const key = new ProviderError("key");
  const value = new ProviderError("value");
  const valued = WithValue(root, key, value);
  assert.equal(valued.Value(key), value);

  const [cancelled, cancel] = WithCancel(valued);
  let callbackCount = 0;
  const stopCallback = ContextAfterFunc(cancelled, async () => {
    callbackCount += 1;
  });
  await cancel();
  const [, open] = await cancelled.Done()!.receive();
  assert.equal(open, false);
  assert.equal(cancelled.Err(), state.Canceled);
  await Promise.resolve();
  assert.equal(callbackCount, 1);
  assert.equal(await stopCallback(), false);

  const cause = new ProviderError("cause");
  const [caused, cancelCause] = WithCancelCause(root);
  await cancelCause(cause);
  assert.equal(caused.Err(), cause);

  const [timed, stop] = WithTimeout(root, new Duration(1_000_000));
  const [, deadlineOpen] = await timed.Done()!.receive();
  assert.equal(deadlineOpen, false);
  assert.match(timed.Err()!.Error(), /deadline exceeded/u);
  await stop();
});
