import assert from "node:assert/strict";
import { Buffer } from "node:buffer";
import test from "node:test";

import { RuntimeSlice } from "@gotots/runtime/slice.js";
import { sliceValues } from "../src/internal/runtime/slice.js";

import {
  Is,
  IsDigit,
  IsLetter,
  IsLower,
  IsNumber,
  IsPrint,
  IsSpace,
  IsUpper,
  In,
  Range16,
  RangeTable,
  ToLower,
  ToUpper,
  SimpleFold,
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
  AppendRune,
  DecodeRune as DecodeUTF8Rune,
  DecodeLastRuneInString,
  DecodeRuneInString,
  EncodeRune as EncodeUTF8Rune,
  FullRune,
  FullRuneInString,
  RuneError,
  RuneCount,
  RuneSelf,
  RuneStart,
  UTFMax,
  ValidString,
} from "../src/unicode/utf8.js";

test("unicode properties and case mappings match selected Go tables", () => {
  assert.equal(IsDigit(0x0665), true);
  assert.equal(IsLetter(0x4e2d), true);
  assert.equal(IsLetter(0x0035), false);
  assert.equal(IsNumber(0x2165), true);
  assert.equal(IsDigit(0x2165), false);
  assert.equal(IsPrint(0x1f642), true);
  assert.equal(IsPrint(0x0009), false);
  assert.equal(IsUpper(0x03a3), true);
  assert.equal(IsLower(0x03c2), true);
  assert.equal(IsSpace(0x3000), true);
  assert.equal(ToUpper(0x00b5), 0x039c);
  assert.equal(ToLower(0x0130), 0x0069);
  assert.equal(Is(state.Zs, 0x202f), true);
  assert.equal(Is(state.Nd, 0x0665), true);
  assert.equal(Is(state.Nd, 0x2165), false);
  assert.equal(Is(state.No, 0x00bd), true);
  assert.equal(Is(state.No, 0x0035), false);
  assert.equal(Is(state.Ideographic, 0x4e00), true);
  assert.equal(Is(state.Ideographic, 0x0041), false);

  const custom = new RangeTable(
    RuntimeSlice.literal([new Range16(10, 20, 2)]),
    RuntimeSlice.nil(),
    0n,
  );
  assert.equal(Is(custom, 16), true);
  assert.equal(Is(custom, 17), false);
  assert.equal(In(16, RuntimeSlice.literal([custom])), true);
  assert.equal(In(17, RuntimeSlice.literal([custom])), false);
  assert.equal(SimpleFold(0x004b), 0x006b);
  assert.equal(SimpleFold(0x006b), 0x212a);
  assert.throws(() => Is(undefined, 1));
});

test("unicode utf8 decoders preserve Go invalid-sequence widths", () => {
  assert.equal(RuneError, 0xfffd);
  assert.equal(RuneSelf, 0x80);
  assert.deepEqual(DecodeRuneInString(goText("é")), [0x00e9, 2n]);
  assert.deepEqual(DecodeRuneInString(String.fromCharCode(0xff, 0x41)), [0xfffd, 1n]);
  assert.deepEqual(DecodeLastRuneInString(goText("A𝄞")), [0x1d11e, 4n]);
  assert.deepEqual(DecodeLastRuneInString(""), [0xfffd, 0n]);
});

test("unicode utf8 byte operations preserve Go widths and append behavior", () => {
  const encoded = byteSlice(goText("é"));
  assert.equal(UTFMax, 4n);
  assert.deepEqual(DecodeUTF8Rune(encoded), [0x00e9, 2n]);
  assert.equal(FullRune(encoded), true);
  assert.equal(FullRune(byteSlice(String.fromCharCode(0xe2, 0x82))), false);
  assert.equal(FullRuneInString(goText("é")), true);
  assert.equal(FullRuneInString(String.fromCharCode(0xe2, 0x82)), false);
  assert.equal(FullRuneInString(String.fromCharCode(0xe0, 0x80)), true);
  assert.equal(FullRuneInString(String.fromCharCode(0xf0, 0x90, 0x80)), false);
  assert.equal(FullRuneInString(String.fromCharCode(0xf0, 0x80)), true);
  assert.equal(RuneCount(byteSlice(String.fromCharCode(0xff, 0x41))), 2n);
  assert.equal(RuneStart(0x80), false);
  assert.equal(RuneStart(0x41), true);
  assert.equal(ValidString(goText("A🙂")), true);
  assert.equal(ValidString(String.fromCharCode(0xff)), false);

  const target = RuntimeSlice.make<number>(4, null, 0);
  assert.equal(EncodeUTF8Rune(target, 0x1f642), 4n);
  assert.deepEqual(sliceValues(target), [0xf0, 0x9f, 0x99, 0x82]);
  assert.deepEqual(
    sliceValues(AppendRune(RuntimeSlice.literal([0x41]), 0x00e9)),
    [0x41, 0xc3, 0xa9],
  );
});

test("unicode utf16 handles valid and malformed surrogate pairs", () => {
  assert.equal(IsSurrogate(0xd800), true);
  assert.deepEqual(EncodeRune(0x1f642), [0xd83d, 0xde42]);
  assert.equal(DecodeRune(0xd83d, 0xde42), 0x1f642);
  assert.equal(DecodeRune(0xd83d, 0x41), 0xfffd);
  assert.equal(RuneLen(0xd800), -1n);
  assert.deepEqual(
    sliceValues(Decode(RuntimeSlice.literal([0x41, 0xd83d, 0xde42, 0xd800]))),
    [0x41, 0x1f642, 0xfffd],
  );
});

function goText(value: string): string {
  return Buffer.from(value, "utf8").toString("latin1");
}

function byteSlice(value: string): RuntimeSlice<number> {
  return RuntimeSlice.literal([...value].map((character) => character.charCodeAt(0)));
}
