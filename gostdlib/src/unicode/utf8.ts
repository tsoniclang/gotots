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
  if (source.length === 0) {
    return false;
  }
  const first = source.get(0);
  if (first < RuneSelf || first < 0xc2 || first > 0xf4) {
    return true;
  }
  const width = first < 0xe0 ? 2 : first < 0xf0 ? 3 : 4;
  return source.length >= width;
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

function encodedBytes(rune: int32): uint8[] {
  return [...encodeRune(rune)].map((value) => value.charCodeAt(0));
}
