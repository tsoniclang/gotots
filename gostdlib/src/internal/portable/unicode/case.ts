import type { int32 } from "@gotots/gostdlib/internal/scalars.js";

import {
  caseOrbit,
  caseRanges,
  type CaseRangeRecord,
} from "./data.js";

const upperCase = 0;
const lowerCase = 1;
const upperLower = 0x110000;

export function ToUpper(rune: int32): int32 {
  if (rune >= 0x61 && rune <= 0x7a) {
    return rune - 0x20;
  }
  return mapCase(upperCase, rune);
}

export function ToLower(rune: int32): int32 {
  if (rune >= 0x41 && rune <= 0x5a) {
    return rune + 0x20;
  }
  return mapCase(lowerCase, rune);
}

export function SimpleFold(rune: int32): int32 {
  if (rune < 0 || rune > 0x10ffff) {
    return rune;
  }
  if (rune < 0x80) {
    if (rune >= 0x41 && rune <= 0x4a) {
      return rune + 0x20;
    }
    if (rune === 0x4b) {
      return 0x6b;
    }
    if (rune >= 0x4c && rune <= 0x52) {
      return rune + 0x20;
    }
    if (rune === 0x53) {
      return 0x73;
    }
    if (rune >= 0x54 && rune <= 0x5a) {
      return rune + 0x20;
    }
    if (rune >= 0x61 && rune <= 0x6a) {
      return rune - 0x20;
    }
    if (rune === 0x6b) {
      return 0x212a;
    }
    if (rune >= 0x6c && rune <= 0x72) {
      return rune - 0x20;
    }
    if (rune === 0x73) {
      return 0x17f;
    }
    if (rune >= 0x74 && rune <= 0x7a) {
      return rune - 0x20;
    }
    return rune;
  }

  const orbit = findOrbit(rune);
  if (orbit !== undefined) {
    return orbit;
  }

  const range = findCaseRange(rune);
  if (range === undefined) {
    return rune;
  }
  const lower = convertCase(lowerCase, rune, range);
  return lower !== rune ? lower : convertCase(upperCase, rune, range);
}

function mapCase(kind: number, rune: int32): int32 {
  const range = findCaseRange(rune);
  return range === undefined ? rune : convertCase(kind, rune, range);
}

function findCaseRange(rune: int32): CaseRangeRecord | undefined {
  let low = 0;
  let high = caseRanges.length;
  while (low < high) {
    const middle = Math.floor((low + high) / 2);
    const range = caseRanges[middle];
    if (range === undefined) {
      return undefined;
    }
    if (rune < range[0]) {
      high = middle;
    } else if (rune > range[1]) {
      low = middle + 1;
    } else {
      return range;
    }
  }
  return undefined;
}

function convertCase(kind: number, rune: int32, range: CaseRangeRecord): int32 {
  const delta = range[kind + 2];
  if (delta === undefined) {
    return rune;
  }
  if (delta === upperLower) {
    return range[0] + (((rune - range[0]) & ~1) | (kind & 1));
  }
  return rune + delta;
}

function findOrbit(rune: int32): int32 | undefined {
  let low = 0;
  let high = caseOrbit.length;
  while (low < high) {
    const middle = Math.floor((low + high) / 2);
    const pair = caseOrbit[middle];
    if (pair === undefined) {
      return undefined;
    }
    if (pair[0] < rune) {
      low = middle + 1;
    } else {
      high = middle;
    }
  }
  const pair = caseOrbit[low];
  return pair !== undefined && pair[0] === rune ? pair[1] : undefined;
}
