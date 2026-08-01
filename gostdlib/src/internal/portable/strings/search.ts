import { GoPanic } from "@gotots/runtime/panic.js";
import type {
  bool,
  gostring,
  int32,
  int64,
  uint8,
} from "@gotots/runtime/scalars.js";

import {
  decodeRuneAt,
  encodeRune,
  runeCount,
  validRune,
} from "../utf8/codec.js";
import { SimpleFold } from "../unicode/case.js";

export function Contains(text: gostring, substring: gostring): bool {
  return text.includes(substring);
}

export function ContainsFunc(
  text: gostring,
  predicate: ((rune: int32) => bool) | undefined,
): bool {
  return IndexFunc(text, predicate) >= 0;
}

export function HasPrefix(text: gostring, prefix: gostring): bool {
  return text.startsWith(prefix);
}

export function HasSuffix(text: gostring, suffix: gostring): bool {
  return text.endsWith(suffix);
}

export function Index(text: gostring, substring: gostring): int64 {
  return text.indexOf(substring);
}

export function Clone(text: gostring): gostring {
  return text;
}

export function Compare(left: gostring, right: gostring): int64 {
  return left === right ? 0 : left < right ? -1 : 1;
}

export function ContainsAny(text: gostring, characters: gostring): bool {
  return IndexAny(text, characters) >= 0;
}

export function ContainsRune(text: gostring, rune: int32): bool {
  return IndexRune(text, rune) >= 0;
}

export function IndexRune(text: gostring, rune: int32): int64 {
  if (!validRune(rune)) {
    return -1;
  }
  if (rune === 0xfffd) {
    for (let index = 0; index < text.length; ) {
      const [decoded, width] = decodeRuneAt(text, index);
      if (decoded === rune) {
        return index;
      }
      index += Math.max(1, width);
    }
    return -1;
  }
  return text.indexOf(encodeRune(rune));
}

export function Count(text: gostring, substring: gostring): int64 {
  if (substring.length === 0) {
    return runeCount(text) + 1;
  }
  let count = 0;
  for (let start = 0; ; ) {
    const index = text.indexOf(substring, start);
    if (index < 0) {
      return count;
    }
    count += 1;
    start = index + substring.length;
  }
}

export function Cut(text: gostring, separator: gostring): [gostring, gostring, bool] {
  const index = text.indexOf(separator);
  return index < 0
    ? [text, "", false]
    : [text.slice(0, index), text.slice(index + separator.length), true];
}

export function CutPrefix(text: gostring, prefix: gostring): [gostring, bool] {
  return text.startsWith(prefix) ? [text.slice(prefix.length), true] : [text, false];
}

export function CutSuffix(text: gostring, suffix: gostring): [gostring, bool] {
  return text.endsWith(suffix) ? [text.slice(0, text.length - suffix.length), true] : [text, false];
}

export function EqualFold(left: gostring, right: gostring): bool {
  let leftIndex = 0;
  let rightIndex = 0;
  while (leftIndex < left.length && rightIndex < right.length) {
    const [leftRune, leftWidth] = decodeRuneAt(left, leftIndex);
    const [rightRune, rightWidth] = decodeRuneAt(right, rightIndex);
    if (!runesEqualFold(leftRune, rightRune)) {
      return false;
    }
    leftIndex += Math.max(1, leftWidth);
    rightIndex += Math.max(1, rightWidth);
  }
  return leftIndex === left.length && rightIndex === right.length;
}

export function IndexAny(text: gostring, characters: gostring): int64 {
  const set = runeSet(characters);
  for (let index = 0; index < text.length; ) {
    const [rune, width] = decodeRuneAt(text, index);
    if (set.has(rune)) {
      return index;
    }
    index += Math.max(1, width);
  }
  return -1;
}

export function IndexByte(text: gostring, value: uint8): int64 {
  return text.indexOf(String.fromCharCode(value));
}

export function IndexFunc(
  text: gostring,
  predicate: ((rune: int32) => bool) | undefined,
): int64 {
  return findByPredicate(text, predicate, true, false);
}

export function LastIndex(text: gostring, substring: gostring): int64 {
  return text.lastIndexOf(substring);
}

export function LastIndexByte(text: gostring, value: uint8): int64 {
  return text.lastIndexOf(String.fromCharCode(value));
}

export function LastIndexFunc(
  text: gostring,
  predicate: ((rune: int32) => bool) | undefined,
): int64 {
  return findByPredicate(text, predicate, true, true);
}

export function findByPredicate(
  text: gostring,
  predicate: ((rune: int32) => bool) | undefined,
  expected: boolean,
  last: boolean,
): int64 {
  if (predicate === undefined && text.length > 0) {
    GoPanic.raiseRuntime("call of nil predicate function");
  }
  let found = -1;
  for (let index = 0; index < text.length; ) {
    const [rune, width] = decodeRuneAt(text, index);
    if (predicate?.(rune) === expected) {
      found = index;
      if (!last) {
        return found;
      }
    }
    index += Math.max(1, width);
  }
  return found;
}

function runeSet(text: gostring): Set<int32> {
  const result = new Set<int32>();
  for (let index = 0; index < text.length; ) {
    const [rune, width] = decodeRuneAt(text, index);
    result.add(rune);
    index += Math.max(1, width);
  }
  return result;
}

function runesEqualFold(left: int32, right: int32): boolean {
  if (left === right) {
    return true;
  }
  let smaller = Math.min(left, right);
  const larger = Math.max(left, right);
  const start = smaller;
  do {
    smaller = SimpleFold(smaller);
    if (smaller === larger) {
      return true;
    }
  } while (smaller !== start && smaller < larger);
  return false;
}
