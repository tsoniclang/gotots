import assert from "node:assert/strict";
import test from "node:test";
import {
  Bool,
  Int32,
  Int64,
  Uint32,
  Uint64,
} from "../src/sync/atomic.js";

test("atomic cells implement load, store, swap, add, and compare-and-swap", () => {
  const flag = new Bool();
  assert.equal(Bool.Load(flag), false);
  assert.equal(Bool.CompareAndSwap(flag, false, true), true);
  assert.equal(Bool.Load(flag), true);
  Bool.Store(flag, false);

  const signed32 = new Int32();
  Int32.Store(signed32, 7);
  assert.equal(Int32.Add(signed32, 5), 12);
  assert.equal(Int32.Swap(signed32, -2), 12);
  assert.equal(Int32.CompareAndSwap(signed32, -2, 9), true);

  const signed64 = new Int64();
  Int64.Store(signed64, 10n);
  assert.equal(Int64.Add(signed64, 2n), 12n);
  assert.equal(Int64.CompareAndSwap(signed64, 12n, 20n), true);

  const unsigned32 = new Uint32();
  Uint32.Store(unsigned32, 0xffff_ffff);
  assert.equal(Uint32.Add(unsigned32, 1), 0);

  const unsigned64 = new Uint64();
  assert.equal(Uint64.Add(unsigned64, 3n), 3n);
  assert.equal(Uint64.CompareAndSwap(unsigned64, 3n, 5n), true);
  assert.equal(Uint64.Load(unsigned64), 5n);
});
