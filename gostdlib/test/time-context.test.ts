import assert from "node:assert/strict";
import test from "node:test";
import type { GoReceiveChannel } from "@gotots/runtime/channel.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { GoEmptyStruct } from "@gotots/runtime/struct.js";
import { ProviderError } from "../src/internal/runtime/error.js";
import { ProviderInterfaceValue } from "../src/internal/portable/io/value.js";
import {
  AfterFunc as ContextAfterFunc,
  Background,
  Cause,
  state,
  TODO,
  WithCancel,
  WithCancelCause,
  WithTimeout,
  WithValue,
} from "../src/context.js";
import type { Context } from "../src/context.js";
import {
  After,
  AfterFunc,
  Duration,
  Hour,
  Microsecond,
  Millisecond,
  NewTicker,
  NewTimer,
  Nanosecond,
  Parse,
  ParseDuration,
  Since,
  Sleep,
  Second,
  Ticker,
  Now,
  Time,
  Timer,
  UnixMilli,
  Until,
} from "../src/time.js";

const testContextType = Object.freeze({ comparable: true });

class NilDoneFailedContext extends ProviderInterfaceValue implements Context {
  constructor(private readonly failure: ProviderError) {
    super(testContextType);
  }

  Deadline(): [Time, boolean] {
    return [new Time(), false];
  }

  Done(): undefined {
    return undefined;
  }

  Err(): ProviderError {
    return this.failure;
  }

  Value(): undefined {
    return undefined;
  }
}

test("ParseDuration preserves Go units, fractions, and diagnostics", (): void => {
  const valid: ReadonlyArray<readonly [string, bigint]> = [
    ["0", 0n],
    ["+0", 0n],
    ["300ms", 300_000_000n],
    ["-1.5h", -5_400_000_000_000n],
    ["2h45m", 9_900_000_000_000n],
    [".5s", 500_000_000n],
    ["1us", 1_000n],
    ["1µs", 1_000n],
    ["1μs", 1_000n],
  ];
  for (const [source, want] of valid) {
    const [duration, failure] = ParseDuration(source);
    assert.equal(failure, undefined, source);
    assert.equal(duration.Nanoseconds(), want, source);
  }

  const invalid: ReadonlyArray<readonly [string, string]> = [
    ["", 'time: invalid duration ""'],
    ["1", 'time: missing unit in duration "1"'],
    [".s", 'time: invalid duration ".s"'],
    ["1x", 'time: unknown unit "x" in duration "1x"'],
    ["2562047h47m16.854775808s", 'time: invalid duration "2562047h47m16.854775808s"'],
  ];
  for (const [source, want] of invalid) {
    const [duration, failure] = ParseDuration(source);
    assert.equal(duration.Nanoseconds(), 0n, source);
    assert.equal(failure?.Error(), want, source);
  }
});

test("Duration and Time preserve arithmetic, ordering, formatting, and zero", () => {
  const duration = new Duration(1_250_000_000n);
  assert.equal(duration.Nanoseconds(), 1_250_000_000n);
  assert.equal(duration.Seconds(), 1.25);
  assert.equal(duration.String(), "1.25s");

  const epoch = UnixMilli(0n);
  const later = epoch.Add(duration);
  assert.equal(epoch.Before(later), true);
  assert.equal(later.After(epoch), true);
  assert.equal(epoch.Equal(UnixMilli(0n)), true);
  assert.equal(later.Sub(epoch).Nanoseconds(), duration.Nanoseconds());
  const localEpoch = new Date(0);
  const hour = localEpoch.getHours() % 12 || 12;
  const expectedTime = `${String(hour).padStart(2, "0")}:`
    + `${String(localEpoch.getMinutes()).padStart(2, "0")}:`
    + `${String(localEpoch.getSeconds()).padStart(2, "0")} `
    + `${localEpoch.getHours() < 12 ? "AM" : "PM"}`;
  assert.equal(UnixMilli(0n).Format("03:04:05 PM"), expectedTime);
  assert.equal(new Time().IsZero(), true);
  assert.equal(UnixMilli(0n).UnixMilli(), 0n);
  assert.equal(UnixMilli(0n).UnixNano(), 0n);
  assert.match(UnixMilli(0n).String(), /1970/u);
  const prefix = RuntimeSlice.literal([0x61, 0x74, 0x3d]);
  const appended = UnixMilli(0n).AppendFormat(prefix, "2006");
  const bytes = Array.from(
    { length: appended.length },
    (_, index): number => appended.get(index),
  );
  assert.equal(new TextDecoder().decode(Uint8Array.from(bytes)), "at=1970");
  assert.equal(Millisecond.Nanoseconds(), 1_000_000n);
  assert.equal(Nanosecond.Nanoseconds(), 1n);
  assert.equal(Microsecond.Nanoseconds(), 1_000n);
  assert.equal(Second.Nanoseconds(), 1_000_000_000n);
  assert.equal(Hour.Nanoseconds(), 3_600_000_000_000n);
  const now = Now();
  assert.equal(now.IsZero(), false);
  assert.ok(Since(now).Nanoseconds() >= 0n);
  assert.ok(Until(now.Add(new Duration(1_000_000_000n))).Nanoseconds() > 0n);
});

test("Time.UnmarshalText preserves the parsed instant and fixed offset", (): void => {
  const parsed = new Time();
  assert.equal(
    parsed.UnmarshalText(RuntimeSlice.literal(Array.from(
      new TextEncoder().encode("2024-01-02T03:04:05.123456789+02:30"),
    ))),
    undefined,
  );
  assert.equal(
    parsed.Format("2006-01-02T15:04:05.000000000Z07:00"),
    "2024-01-02T03:04:05.123456789+02:30",
  );

  const sameInstant = new Time();
  assert.equal(
    sameInstant.UnmarshalText(RuntimeSlice.literal(Array.from(
      new TextEncoder().encode("2024-01-02T00:34:05.123456789Z"),
    ))),
    undefined,
  );
  assert.equal(parsed.Equal(sameInstant), true);

  const invalid = UnixMilli(0n);
  const failure = invalid.UnmarshalText(RuntimeSlice.literal(Array.from(
    new TextEncoder().encode("not-a-time"),
  )));
  assert.match(failure?.Error() ?? "", /^parsing time /u);
  assert.equal(invalid.IsZero(), true);
});

test("time.Parse consumes Go reference layouts", (): void => {
  const cases: ReadonlyArray<readonly [string, string, string]> = [
    [
      "2006-01-02 15:04:05",
      "2024-02-29 23:07:08",
      "2024-02-29T23:07:08.000000000Z",
    ],
    [
      "Jan _2 3:04pm MST",
      "Feb  3 9:07pm UTC",
      "0000-02-03T21:07:00.000000000Z",
    ],
    [
      "2006-002 15:04:05.999999999Z07:00",
      "2024-060 01:02:03.456789123+02:30",
      "2024-02-29T01:02:03.456789123+02:30",
    ],
  ];
  for (const [layout, source, want] of cases) {
    const [parsed, failure] = Parse(layout, source);
    assert.equal(failure, undefined, `${layout} / ${source}`);
    assert.equal(
      parsed.Format("2006-01-02T15:04:05.000000000Z07:00"),
      want,
    );
  }

  const [invalid, failure] = Parse("2006-01-02", "2024-02-30");
  assert.equal(invalid.IsZero(), true);
  assert.match(failure?.Error() ?? "", /^parsing time /u);
});

test("Timer delivers once, supports reset, and reports stop state", async () => {
  const timer = NewTimer(new Duration(50_000_000n));
  assert.equal(Timer.Stop(timer), true);
  assert.equal(Timer.Reset(timer, new Duration(1_000_000n)), false);
  const [fired, ok] = await timer.C!.receive();
  assert.equal(ok, true);
  assert.equal(fired.IsZero(), false);
  assert.equal(Timer.Stop(timer), false);
});

test("Ticker produces values until stopped", async () => {
  const ticker = NewTicker(new Duration(1_000_000n));
  const [, ok] = await ticker.C.receive();
  assert.equal(ok, true);
  Ticker.Stop(ticker);
});

test("After and AfterFunc schedule through the provider clock", async () => {
  const [, open] = await After(new Duration(1_000_000n)).receive();
  assert.equal(open, true);

  let called = false;
  const timer = AfterFunc(new Duration(1_000_000n), async () => {
    called = true;
  });
  await new Promise<void>((resolve) => setTimeout(resolve, 5));
  assert.equal(called, true);
  assert.equal(timer.C, undefined);
});

test("Sleep delays positive durations and accepts non-positive durations", async () => {
  const order: string[] = [];
  const sleeping = Sleep(new Duration(1_000_000n)).then(() => order.push("awake"));
  order.push("scheduled");
  await sleeping;
  assert.deepEqual(order, ["scheduled", "awake"]);

  await Sleep(new Duration(0n));
  await Sleep(new Duration(-1n));
});

test("Context cancellation, causes, values, and deadlines propagate", async () => {
  const root = Background();
  const canonicalDoneOwner: {
    Done(): GoReceiveChannel<GoEmptyStruct> | undefined;
  } = root;
  assert.equal(canonicalDoneOwner.Done(), undefined);
  assert.notEqual(TODO(), root);
  const ignoredFailure = new ProviderError("ignored without Done");
  const [neverCanceled, cancelNever] = WithCancel(
    new NilDoneFailedContext(ignoredFailure),
  );
  assert.equal(neverCanceled.Err(), undefined);
  await cancelNever();
  assert.equal(neverCanceled.Err(), state.Canceled);

  const [closedParent, closeParent] = WithCancel(root);
  await closeParent();
  const [closedChild] = WithCancel(closedParent);
  assert.equal(closedChild.Err(), state.Canceled);

  const [futureParent, closeFutureParent] = WithCancel(root);
  const [futureChild] = WithCancel(futureParent);
  await closeFutureParent();
  assert.equal(futureChild.Err(), state.Canceled);
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
  assert.equal(caused.Err(), state.Canceled);
  assert.equal(Cause(caused), cause);

  const [timed, stop] = WithTimeout(root, new Duration(1_000_000n));
  const [, deadlineOpen] = await timed.Done()!.receive();
  assert.equal(deadlineOpen, false);
  assert.match(timed.Err()!.Error(), /deadline exceeded/u);
  assert.equal(Cause(timed), state.DeadlineExceeded);
  await stop();
});
