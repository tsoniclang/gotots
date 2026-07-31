import assert from "node:assert/strict";
import test from "node:test";

import { RuntimeSlice } from "@gotots/runtime/slice.js";

import { New as NewError } from "../src/errors.js";
import { PathError } from "../src/io/fs.js";
import { Float, Int } from "../src/math/big.js";
import { URL } from "../src/net/url.js";
import { MemStats } from "../src/runtime.js";
import { Duration, NewTicker, Ticker, Timer } from "../src/time.js";
import { Range16, Range32 } from "../src/unicode.js";
import {
  IoFsPathErrorOperations,
} from "../src/internal/facets/named-io-fs.js";
import {
  MathBigFloatOperations,
  MathBigIntOperations,
} from "../src/internal/facets/named-math-big.js";
import {
  NetUrlURLOperations,
} from "../src/internal/facets/named-net-url.js";
import {
  RuntimeMemStatsOperations,
} from "../src/internal/facets/named-runtime.js";
import {
  TimeTickerOperations,
  TimeTimerOperations,
} from "../src/internal/facets/named-time.js";
import {
  UnicodeRangeTableOperations,
} from "../src/internal/facets/named-unicode.js";

test("provider named-struct storage capabilities preserve exact values", (): void => {
  const failure = NewError("denied");
  const pathError = IoFsPathErrorOperations.$make("open", "/tmp/x", failure);
  assert.ok(pathError instanceof PathError);
  assert.deepEqual(
    [pathError.Op, pathError.Path, pathError.Err],
    ["open", "/tmp/x", failure],
  );
  assert.equal(IoFsPathErrorOperations.$fromStorage(
    IoFsPathErrorOperations.$storageOf(pathError),
  ), pathError);

  const integer = new Int();
  assert.equal(MathBigIntOperations.$fromStorage(
    MathBigIntOperations.$storageOf(integer),
  ), integer);
  const floating = new Float();
  assert.equal(MathBigFloatOperations.$fromStorage(
    MathBigFloatOperations.$storageOf(floating),
  ), floating);

  const url = new URL();
  assert.equal(NetUrlURLOperations.$fromStorage(
    NetUrlURLOperations.$storageOf(url),
  ), url);

  const memory = new MemStats();
  assert.equal(RuntimeMemStatsOperations.$fromStorage(
    RuntimeMemStatsOperations.$storageOf(memory),
  ), memory);

  const timer = new Timer();
  assert.equal(TimeTimerOperations.$fromStorage(
    TimeTimerOperations.$storageOf(timer),
  ), timer);
  const ticker = NewTicker(new Duration(1_000_000_000));
  assert.equal(TimeTickerOperations.$fromStorage(
    TimeTickerOperations.$storageOf(ticker),
  ), ticker);
  Ticker.Stop(ticker);

  const ranges16 = RuntimeSlice.literal<Range16>([]);
  const ranges32 = RuntimeSlice.literal<Range32>([]);
  const table = UnicodeRangeTableOperations.$make(ranges16, ranges32, 0);
  assert.equal(UnicodeRangeTableOperations.$fromStorage(
    UnicodeRangeTableOperations.$storageOf(table),
  ), table);
});
