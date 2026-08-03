import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring, int32, int64, uint8 } from "@gotots/runtime/scalars.js";

import {
  decodeLastRune,
  decodeRuneAt,
  encodeRune,
  RuneError as runeError,
  RuneSelf as runeSelf,
} from "../internal/portable/utf8/codec.js";

export const RuneError: int32 = runeError;
export const RuneSelf: int32 = runeSelf;
export const UTFMax: int64 = 4;

export function AppendRune(target: RuntimeSlice<uint8>, rune: int32): RuntimeSlice<uint8> {
  return target.append(0, encodedBytes(rune));
}

export function DecodeRune(source: RuntimeSlice<uint8>): [int32, int64] {
  return decodeRuneAt(byteString(source), 0);
}

export function DecodeRuneInString(value: gostring): [int32, int64] {
  return decodeRuneAt(value, 0);
}

export function DecodeLastRuneInString(value: gostring): [int32, int64] {
  return decodeLastRune(value);
}

export function EncodeRune(target: RuntimeSlice<uint8>, rune: int32): int64 {
  const bytes = encodedBytes(rune);
  for (let index = 0; index < bytes.length; index += 1) {
    target.set(index, bytes[index]!);
  }
  return bytes.length;
}

export function FullRune(source: RuntimeSlice<uint8>): boolean {
  return fullRunePrefix(
    source.length,
    source.length > 0 ? source.get(0) : 0,
    source.length > 1 ? source.get(1) : 0,
    source.length > 2 ? source.get(2) : 0,
  );
}

export function FullRuneInString(source: gostring): boolean {
  return fullRunePrefix(
    source.length,
    source.length > 0 ? source.charCodeAt(0) : 0,
    source.length > 1 ? source.charCodeAt(1) : 0,
    source.length > 2 ? source.charCodeAt(2) : 0,
  );
}

export function RuneCount(source: RuntimeSlice<uint8>): int64 {
  const value = byteString(source);
  let count = 0;
  for (let index = 0; index < value.length; count += 1) {
    const [, width] = decodeRuneAt(value, index);
    index += Math.max(1, width);
  }
  return count;
}

export function RuneStart(value: uint8): boolean {
  return (value & 0xc0) !== 0x80;
}

export function ValidString(value: gostring): boolean {
  for (let index = 0; index < value.length; ) {
    const [rune, width] = decodeRuneAt(value, index);
    if (rune === RuneError && width === 1) {
      return false;
    }
    index += width;
  }
  return true;
}

function byteString(source: RuntimeSlice<uint8>): gostring {
  let value = "";
  for (let index = 0; index < source.length; index += 1) {
    value += String.fromCharCode(source.get(index));
  }
  return value;
}

function fullRunePrefix(
  length: number,
  first: number,
  second: number,
  third: number,
): boolean {
  if (length === 0) {
    return false;
  }
  if (first < 0xc2 || first > 0xf4) {
    return true;
  }
  const width = first < 0xe0 ? 2 : first < 0xf0 ? 3 : 4;
  if (length >= width) {
    return true;
  }
  if (length > 1 && !validSecondByte(first, second)) {
    return true;
  }
  return length > 2 && (third < 0x80 || third > 0xbf);
}

function validSecondByte(first: number, second: number): boolean {
  return second >= (first === 0xe0 ? 0xa0 : first === 0xf0 ? 0x90 : 0x80)
    && second <= (first === 0xed ? 0x9f : first === 0xf4 ? 0x8f : 0xbf);
}

function encodedBytes(rune: int32): uint8[] {
  return [...encodeRune(rune)].map((value) => value.charCodeAt(0));
}
