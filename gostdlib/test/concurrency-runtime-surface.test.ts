import assert from "node:assert/strict";
import test from "node:test";
import * as context from "../src/context.js";
import * as runtime from "../src/runtime.js";
import * as debug from "../src/runtime/debug.js";
import * as metrics from "../src/runtime/metrics.js";
import * as pprof from "../src/runtime/pprof.js";
import * as sync from "../src/sync.js";
import * as atomic from "../src/sync/atomic.js";
import * as testing from "../src/testing.js";
import * as time from "../src/time.js";
import { ProviderChannel } from "../src/internal/portable/concurrency/channel.js";

type Same<Left, Right> =
  [Left] extends [Right]
    ? [Right] extends [Left]
      ? true
      : false
    : false;

const contextSurface: Same<
  keyof typeof context,
  | "AfterFunc"
  | "Background"
  | "Cause"
  | "state"
  | "TODO"
  | "WithCancel"
  | "WithCancelCause"
  | "WithTimeout"
  | "WithValue"
> = true;

const syncSurface: Same<
  keyof typeof sync,
  | "Map"
  | "Mutex"
  | "Once"
  | "OnceFunc"
  | "OnceValue"
  | "Pool"
  | "RWMutex"
  | "WaitGroup"
> = true;

const atomicSurface: Same<
  keyof typeof atomic,
  "Bool" | "Int32" | "Int64" | "Uint32" | "Uint64"
> = true;

const timeSurface: Same<
  keyof typeof time,
  | "After"
  | "AfterFunc"
  | "Duration"
  | "Hour"
  | "Microsecond"
  | "Millisecond"
  | "Minute"
  | "NewTicker"
  | "NewTimer"
  | "Nanosecond"
  | "Now"
  | "Parse"
  | "ParseDuration"
  | "ParseError"
  | "Since"
  | "Second"
  | "Ticker"
  | "Time"
  | "Timer"
  | "Unix"
  | "UnixMilli"
  | "Until"
> = true;

const runtimeSurface: Same<
  keyof typeof runtime,
  | "Caller"
  | "GC"
  | "GOARCH"
  | "GOMAXPROCS"
  | "GOOS"
  | "MemStats"
  | "ReadMemStats"
> = true;

const debugSurface: Same<
  keyof typeof debug,
  "SetMaxStack" | "Stack"
> = true;

const metricsSurface: Same<
  keyof typeof metrics,
  | "All"
  | "Description"
  | "KindFloat64"
  | "KindFloat64Histogram"
  | "KindUint64"
  | "Read"
  | "Sample"
  | "Value"
  | "ValueKind"
> = true;

const pprofSurface: Same<
  keyof typeof pprof,
  "Lookup" | "Profile" | "StartCPUProfile" | "StopCPUProfile"
> = true;

const testingSurface: Same<keyof typeof testing, "Testing"> = true;

interface CanonicalReceiveChannel<T> {
  $length(): number;
  $capacity(): number;
  receive(): Promise<[T, boolean]>;
  $selectReceive(
    accept: (value: T, ok: boolean) => void,
  ): {
    ready(): boolean;
    commit(): boolean | object;
    subscribe(
      claim: (failure: object | undefined) => boolean,
    ): () => void;
  };
}

const providerChannel =
  new ProviderChannel<number>(() => 0, (value) => value, 1);
const canonicalChannel: CanonicalReceiveChannel<number> = providerChannel;

test("assigned public modules expose exactly the selected clean surface", () => {
  assert.equal(contextSurface, true);
  assert.equal(syncSurface, true);
  assert.equal(atomicSurface, true);
  assert.equal(timeSurface, true);
  assert.equal(runtimeSurface, true);
  assert.equal(debugSurface, true);
  assert.equal(metricsSurface, true);
  assert.equal(pprofSurface, true);
  assert.equal(testingSurface, true);
});

test("provider channels satisfy the complete generated receive/select boundary", () => {
  assert.equal(canonicalChannel.$capacity(), 1);
  assert.equal(canonicalChannel.$length(), 0);

  assert.equal(providerChannel.offer(7), true);
  const received: Array<[number, boolean]> = [];
  const selected = providerChannel.$selectReceive(
    (value, ok) => received.push([value, ok]),
  );
  assert.equal(selected.ready(), true);
  assert.equal(selected.commit(), true);
  assert.deepEqual(received, [[7, true]]);

  const pending = new ProviderChannel<number>(() => 0, (value) => value, 0);
  const subscribed: Array<[number, boolean]> = [];
  const waiting = pending.$selectReceive(
    (value, ok) => subscribed.push([value, ok]),
  );
  const unsubscribe = waiting.subscribe(() => true);
  assert.equal(pending.offer(9), true);
  assert.deepEqual(subscribed, [[9, true]]);
  unsubscribe();
});
