import assert from "node:assert/strict";
import test from "node:test";

import {
  AppendBool,
  AppendFloat,
  AppendInt,
  AppendQuote,
  AppendUint,
  Atoi,
  FormatInt,
  FormatUint,
  Itoa,
  ParseFloat,
  ParseInt,
  ParseUint,
  Quote,
  QuoteRune,
  state,
  Unquote,
} from "../src/strconv.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import { Unwrap } from "../src/errors.js";
import { sliceValues } from "../src/internal/runtime/slice.js";

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

test("strconv append formatting preserves the destination slice and Go forms", () => {
  const prefix = RuntimeSlice.literal([0x58]);
  assert.deepEqual(sliceValues(AppendBool(prefix, true)), [0x58, 0x74, 0x72, 0x75, 0x65]);
  assert.deepEqual(ascii(AppendInt(prefix, -255, 16)), "X-ff");
  assert.deepEqual(ascii(AppendUint(prefix, 255, 16)), "Xff");
  assert.deepEqual(ascii(AppendFloat(prefix, 1.25, code("g"), -1, 64)), "X1.25");
  assert.deepEqual(ascii(AppendFloat(prefix, 1.2, code("g"), -1, 32)), "X1.2");
  assert.deepEqual(ascii(AppendFloat(prefix, 1e6, code("g"), -1, 64)), "X1e+06");
  assert.deepEqual(ascii(AppendFloat(prefix, 1e-5, code("g"), -1, 64)), "X1e-05");
  assert.deepEqual(
    ascii(AppendFloat(prefix, 3.1415926535, code("E"), -1, 32)),
    "X3.1415927E+00",
  );
  assert.deepEqual(ascii(AppendFloat(prefix, 1.005, code("f"), 2, 64)), "X1.00");
  assert.deepEqual(ascii(AppendFloat(prefix, 1.5, code("x"), -1, 64)), "X0x1.8p+00");
});

test("strconv floating parsing covers decimal, hexadecimal, special, and range forms", () => {
  assert.deepEqual(ParseFloat("1_2.5e1", 64), [125, undefined]);
  assert.deepEqual(ParseFloat("0x1.8p+1", 64), [3, undefined]);
  assert.deepEqual(ParseFloat("1.5", 0), [1.5, undefined]);
  assert.equal(ParseFloat("NaN", 64)[1], undefined);
  assert.equal(ParseFloat("1e999", 64)[0], Number.POSITIVE_INFINITY);
  const overflow = ParseFloat("1e999", 64)[1];
  assert.equal(overflow?.Unwrap(), state.ErrRange);
  assert.equal(Unwrap(overflow), state.ErrRange);
  assert.equal(Unwrap(state.ErrRange), undefined);
  assert.notEqual(ParseFloat("1.2.3", 64)[1], undefined);
});

test("strconv quoting preserves Go byte strings and escape rules", () => {
  assert.equal(Quote("a\n\t\"b"), '"a\\n\\t\\\"b"');
  assert.equal(QuoteRune(0x27), "'\\''");
  assert.equal(QuoteRune(0x00e9), "'Ã©'");
  assert.deepEqual(Unquote('"a\\x00\\u00e9"'), ["a\0Ã©", undefined]);
  assert.deepEqual(Unquote("`a\r\nb`"), ["a\nb", undefined]);
  assert.equal(Unquote('"\\x0"')[1], state.ErrSyntax);
  assert.deepEqual(
    sliceValues(AppendQuote(RuntimeSlice.literal([0x58]), "A\n")),
    [0x58, 0x22, 0x41, 0x5c, 0x6e, 0x22],
  );
});

function ascii(source: RuntimeSlice<number>): string {
  return String.fromCharCode(...sliceValues(source));
}

function code(character: string): number {
  return character.charCodeAt(0);
}
