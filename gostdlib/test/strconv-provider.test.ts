import assert from "node:assert/strict";
import test from "node:test";

import {
  Atoi,
  FormatInt,
  FormatUint,
  Itoa,
  ParseFloat,
  ParseInt,
  ParseUint,
  state,
} from "../src/strconv.js";

test("strconv integer parsing honors bases, prefixes, underscores, and bounds", () => {
  assert.deepEqual(Atoi("-42"), [-42, undefined]);
  assert.deepEqual(ParseInt("0x_7f", 0, 8), [127, undefined]);
  assert.notEqual(ParseInt("_1", 0, 8)[1], undefined);
  assert.deepEqual(ParseInt("0_7", 0, 8), [7, undefined]);
  assert.deepEqual(ParseUint("1111", 2, 8), [15, undefined]);
  assert.deepEqual(ParseInt("128", 10, 8)[0], 127);
  assert.equal(ParseInt("128", 10, 8)[1]?.Unwrap(), state.ErrRange);
  assert.notEqual(ParseUint("-1", 10, 64)[1], undefined);
});

test("strconv formatting uses clean lower-case radix output", () => {
  assert.equal(FormatInt(-255, 16), "-ff");
  assert.equal(FormatUint(255, 16), "ff");
  assert.equal(Itoa(1234), "1234");
  assert.throws(() => FormatInt(1, 1));
});

test("strconv floating parsing covers decimal, hexadecimal, special, and range forms", () => {
  assert.deepEqual(ParseFloat("1_2.5e1", 64), [125, undefined]);
  assert.deepEqual(ParseFloat("0x1.8p+1", 64), [3, undefined]);
  assert.deepEqual(ParseFloat("1.5", 0), [1.5, undefined]);
  assert.equal(ParseFloat("NaN", 64)[1], undefined);
  assert.equal(ParseFloat("1e999", 64)[0], Number.POSITIVE_INFINITY);
  assert.equal(ParseFloat("1e999", 64)[1]?.Unwrap(), state.ErrRange);
  assert.notEqual(ParseFloat("1.2.3", 64)[1], undefined);
});
