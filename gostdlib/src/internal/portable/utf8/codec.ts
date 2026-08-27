import type { bool, gostring, int, int32 } from "@gotots/gostdlib/internal/scalars.js";

import { hostInteger, integerFromHost } from "../../host-integer.js";

export const RuneError: int32 = 0xfffd;
export const RuneSelf = 0x80;
export const MaxRune = 0x10ffff;

export function decodeRuneAt(value: gostring, index: number): [int32, int] {
  if (index >= value.length) {
    return [RuneError, 0n];
  }

  const first = value.charCodeAt(index);
  if (first < RuneSelf) {
    return [first, 1n];
  }
  if (first < 0xc2 || first > 0xf4) {
    return [RuneError, 1n];
  }

  const width = first < 0xe0 ? 2 : first < 0xf0 ? 3 : 4;
  if (index + width > value.length) {
    return [RuneError, 1n];
  }

  const second = value.charCodeAt(index + 1);
  if (
    second < 0x80 ||
    second > 0xbf ||
    (first === 0xe0 && second < 0xa0) ||
    (first === 0xed && second > 0x9f) ||
    (first === 0xf0 && second < 0x90) ||
    (first === 0xf4 && second > 0x8f)
  ) {
    return [RuneError, 1n];
  }

  let rune = (first & (width === 2 ? 0x1f : width === 3 ? 0x0f : 0x07)) << 6;
  rune |= second & 0x3f;
  if (width === 2) {
    return [rune, integerFromHost(width)];
  }

  const third = value.charCodeAt(index + 2);
  if (third < 0x80 || third > 0xbf) {
    return [RuneError, 1n];
  }
  rune = (rune << 6) | (third & 0x3f);
  if (width === 3) {
    return [rune, integerFromHost(width)];
  }

  const fourth = value.charCodeAt(index + 3);
  if (fourth < 0x80 || fourth > 0xbf) {
    return [RuneError, 1n];
  }
  return [(rune << 6) | (fourth & 0x3f), integerFromHost(width)];
}

export function decodeLastRune(value: gostring): [int32, int] {
  if (value.length === 0) {
    return [RuneError, 0n];
  }

  const end = value.length;
  const last = value.charCodeAt(end - 1);
  if (last < RuneSelf) {
    return [last, 1n];
  }

  const startLimit = Math.max(0, end - 4);
  let start = end - 1;
  while (start > startLimit && isContinuation(value.charCodeAt(start))) {
    start -= 1;
  }
  const [rune, width] = decodeRuneAt(value, start);
  if (start + hostInteger(width) !== end) {
    return [RuneError, 1n];
  }
  return [rune, width];
}

export function encodeRune(rune: int32): gostring {
  const scalar = validRune(rune) ? rune : RuneError;
  if (scalar < RuneSelf) {
    return String.fromCharCode(scalar);
  }
  if (scalar < 0x800) {
    return String.fromCharCode(0xc0 | (scalar >> 6), 0x80 | (scalar & 0x3f));
  }
  if (scalar < 0x10000) {
    return String.fromCharCode(
      0xe0 | (scalar >> 12),
      0x80 | ((scalar >> 6) & 0x3f),
      0x80 | (scalar & 0x3f),
    );
  }
  return String.fromCharCode(
    0xf0 | (scalar >> 18),
    0x80 | ((scalar >> 12) & 0x3f),
    0x80 | ((scalar >> 6) & 0x3f),
    0x80 | (scalar & 0x3f),
  );
}

export function runeCount(value: gostring): number {
  let count = 0;
  for (let index = 0; index < value.length; count += 1) {
    const [, width] = decodeRuneAt(value, index);
    index += Math.max(1, hostInteger(width));
  }
  return count;
}

export function runeBoundaries(value: gostring): number[] {
  const boundaries = [0];
  for (let index = 0; index < value.length; ) {
    const [, width] = decodeRuneAt(value, index);
    index += Math.max(1, hostInteger(width));
    boundaries.push(index);
  }
  return boundaries;
}

export function toHostString(value: gostring): string {
  let result = "";
  for (let index = 0; index < value.length; ) {
    const [rune, width] = decodeRuneAt(value, index);
    result += String.fromCodePoint(rune);
    index += Math.max(1, hostInteger(width));
  }
  return result;
}

export function fromHostString(value: string): gostring {
  let result = "";
  for (const scalar of value) {
    result += encodeRune(scalar.codePointAt(0) ?? RuneError);
  }
  return result;
}

export function toHostBytes(value: gostring): Uint8Array {
  const result = new Uint8Array(value.length);
  for (let index = 0; index < value.length; index += 1) {
    const byte = value.charCodeAt(index);
    if (byte > 0xff) {
      throw new RangeError("non-canonical Go string byte");
    }
    result[index] = byte;
  }
  return result;
}

export function fromHostBytes(value: Uint8Array): gostring {
  let result = "";
  for (const byte of value) {
    result += String.fromCharCode(byte);
  }
  return result;
}

export function validRune(rune: int32): bool {
  return Number.isInteger(rune) && rune >= 0 && rune <= MaxRune && (rune < 0xd800 || rune > 0xdfff);
}

function isContinuation(byte: number): boolean {
  return byte >= 0x80 && byte <= 0xbf;
}
