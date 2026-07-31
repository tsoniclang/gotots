import type { int32 } from "@gotots/runtime/scalars.js";

import {
  lowerRanges16,
  lowerRanges32,
  upperRanges16,
  upperRanges32,
} from "./category-data.js";
import {
  digitRanges16,
  digitRanges32,
  type RangeRecord,
} from "./data.js";

export function IsDigit(rune: int32): boolean {
  if (rune <= 0xff) {
    return rune >= 0x30 && rune <= 0x39;
  }
  return inRecords(rune, digitRanges16) || inRecords(rune, digitRanges32);
}

export function IsLower(rune: int32): boolean {
  if (rune <= 0xff) {
    return (
      (rune >= 0x61 && rune <= 0x7a) ||
      rune === 0xb5 ||
      (rune >= 0xdf && rune <= 0xf6) ||
      (rune >= 0xf8 && rune <= 0xff)
    );
  }
  return inRecords(rune, lowerRanges16) || inRecords(rune, lowerRanges32);
}

export function IsUpper(rune: int32): boolean {
  if (rune <= 0xff) {
    return (
      (rune >= 0x41 && rune <= 0x5a) ||
      (rune >= 0xc0 && rune <= 0xd6) ||
      (rune >= 0xd8 && rune <= 0xde)
    );
  }
  return inRecords(rune, upperRanges16) || inRecords(rune, upperRanges32);
}

export function IsSpace(rune: int32): boolean {
  if (rune <= 0xff) {
    return (
      rune === 0x09 ||
      rune === 0x0a ||
      rune === 0x0b ||
      rune === 0x0c ||
      rune === 0x0d ||
      rune === 0x20 ||
      rune === 0x85 ||
      rune === 0xa0
    );
  }
  return (
    rune === 0x1680 ||
    (rune >= 0x2000 && rune <= 0x200a) ||
    rune === 0x2028 ||
    rune === 0x2029 ||
    rune === 0x202f ||
    rune === 0x205f ||
    rune === 0x3000
  );
}

function inRecords(rune: int32, records: readonly RangeRecord[]): boolean {
  let low = 0;
  let high = records.length;
  while (low < high) {
    const middle = Math.floor((low + high) / 2);
    const range = records[middle];
    if (range === undefined) {
      return false;
    }
    if (rune < range[0]) {
      high = middle;
    } else if (rune > range[1]) {
      low = middle + 1;
    } else {
      return range[2] === 1 || (rune - range[0]) % range[2] === 0;
    }
  }
  return false;
}
