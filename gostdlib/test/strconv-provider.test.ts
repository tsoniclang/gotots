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
  assert.deepEqual(Atoi("-42"), [-42n, undefined]);
  assert.deepEqual(ParseInt("0x_7f", 0n, 8n), [127n, undefined]);
  assert.notEqual(ParseInt("_1", 0n, 8n)[1], undefined);
  assert.deepEqual(ParseInt("0_7", 0n, 8n), [7n, undefined]);
  assert.deepEqual(ParseUint("1111", 2n, 8n), [15n, undefined]);
  assert.deepEqual(ParseInt("128", 10n, 8n)[0], 127n);
  assert.equal(ParseInt("128", 10n, 8n)[1]?.Unwrap(), state.ErrRange);
  assert.notEqual(ParseUint("-1", 10n, 64n)[1], undefined);
});

test("strconv formatting uses clean lower-case radix output", () => {
  assert.equal(FormatInt(-255n, 16n), "-ff");
  assert.equal(FormatUint(255n, 16n), "ff");
  assert.equal(Itoa(1234n), "1234");
  assert.throws(() => FormatInt(1n, 1n));
});

test("strconv append formatting preserves the destination slice and Go forms", () => {
  const prefix = RuntimeSlice.literal([0x58]);
  assert.deepEqual(sliceValues(AppendBool(prefix, true)), [0x58, 0x74, 0x72, 0x75, 0x65]);
  assert.deepEqual(ascii(AppendInt(prefix, -255n, 16n)), "X-ff");
  assert.deepEqual(ascii(AppendUint(prefix, 255n, 16n)), "Xff");
  assert.deepEqual(ascii(AppendFloat(prefix, 1.25, code("g"), -1n, 64n)), "X1.25");
  assert.deepEqual(ascii(AppendFloat(prefix, 1.2, code("g"), -1n, 32n)), "X1.2");
  assert.deepEqual(ascii(AppendFloat(prefix, 1e6, code("g"), -1n, 64n)), "X1e+06");
  assert.deepEqual(ascii(AppendFloat(prefix, 1e-5, code("g"), -1n, 64n)), "X1e-05");
  assert.deepEqual(
    ascii(AppendFloat(prefix, 3.1415926535, code("E"), -1n, 32n)),
    "X3.1415927E+00",
  );
  assert.deepEqual(ascii(AppendFloat(prefix, 1.005, code("f"), 2n, 64n)), "X1.00");
  assert.deepEqual(ascii(AppendFloat(prefix, 1.5, code("x"), -1n, 64n)), "X0x1.8p+00");
});

test("strconv floating parsing covers decimal, hexadecimal, special, and range forms", () => {
  assert.deepEqual(ParseFloat("1_2.5e1", 64n), [125, undefined]);
  assert.deepEqual(ParseFloat("0x1.8p+1", 64n), [3, undefined]);
  assert.deepEqual(ParseFloat("1.5", 0n), [1.5, undefined]);
  assert.equal(ParseFloat("NaN", 64n)[1], undefined);
  assert.equal(ParseFloat("1e999", 64n)[0], Number.POSITIVE_INFINITY);
  const overflow = ParseFloat("1e999", 64n)[1];
  assert.equal(overflow?.Unwrap(), state.ErrRange);
  assert.equal(Unwrap(overflow), state.ErrRange);
  assert.equal(Unwrap(state.ErrRange), undefined);
  assert.notEqual(ParseFloat("1.2.3", 64n)[1], undefined);
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
