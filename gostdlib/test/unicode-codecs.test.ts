import assert from "node:assert/strict";
import { Buffer } from "node:buffer";
import test from "node:test";

import { RuntimeSlice } from "@gotots/runtime/slice.js";
import { sliceValues } from "../src/internal/runtime/slice.js";

import {
  Is,
  IsDigit,
  IsLower,
  IsSpace,
  IsUpper,
  Range16,
  RangeTable,
  ToLower,
  ToUpper,
  state,
} from "../src/unicode.js";
import {
  Decode,
  DecodeRune,
  EncodeRune,
  IsSurrogate,
  RuneLen,
} from "../src/unicode/utf16.js";
import {
  DecodeLastRuneInString,
  DecodeRuneInString,
  RuneError,
  RuneSelf,
} from "../src/unicode/utf8.js";

test("unicode properties and case mappings match selected Go tables", () => {
  assert.equal(IsDigit(0x0665), true);
  assert.equal(IsUpper(0x03a3), true);
  assert.equal(IsLower(0x03c2), true);
  assert.equal(IsSpace(0x3000), true);
  assert.equal(ToUpper(0x00b5), 0x039c);
  assert.equal(ToLower(0x0130), 0x0069);
  assert.equal(Is(state.Zs, 0x202f), true);

  const custom = new RangeTable(
    RuntimeSlice.literal([new Range16(10, 20, 2)]),
    RuntimeSlice.nil(),
    0,
  );
  assert.equal(Is(custom, 16), true);
  assert.equal(Is(custom, 17), false);
  assert.throws(() => Is(undefined, 1));
});

test("unicode utf8 decoders preserve Go invalid-sequence widths", () => {
  assert.equal(RuneError, 0xfffd);
  assert.equal(RuneSelf, 0x80);
  assert.deepEqual(DecodeRuneInString(goText("é")), [0x00e9, 2]);
  assert.deepEqual(DecodeRuneInString(String.fromCharCode(0xff, 0x41)), [0xfffd, 1]);
  assert.deepEqual(DecodeLastRuneInString(goText("A𝄞")), [0x1d11e, 4]);
  assert.deepEqual(DecodeLastRuneInString(""), [0xfffd, 0]);
});

test("unicode utf16 handles valid and malformed surrogate pairs", () => {
  assert.equal(IsSurrogate(0xd800), true);
  assert.deepEqual(EncodeRune(0x1f642), [0xd83d, 0xde42]);
  assert.equal(DecodeRune(0xd83d, 0xde42), 0x1f642);
  assert.equal(DecodeRune(0xd83d, 0x41), 0xfffd);
  assert.equal(RuneLen(0xd800), -1);
  assert.deepEqual(
    sliceValues(Decode(RuntimeSlice.literal([0x41, 0xd83d, 0xde42, 0xd800]))),
    [0x41, 0x1f642, 0xfffd],
  );
});

function goText(value: string): string {
  return Buffer.from(value, "utf8").toString("latin1");
}
