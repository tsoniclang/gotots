import { GoString } from "@gotots/runtime/string-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type {
  bool,
  gostring,
  int,
  int32,
  uint8,
} from "@gotots/gostdlib/internal/scalars.js";

import { hostInteger, integerFromHost } from "../../host-integer.js";

import {
  decodeRuneAt,
  encodeRune,
  runeCount,
  validRune,
} from "../utf8/codec.js";
import { SimpleFold } from "../unicode/case.js";

export function Contains(text: gostring, substring: gostring): bool {
  return text.text().includes(substring.text());
}

export function ContainsFunc(
  text: gostring,
  predicate: ((rune: int32) => bool) | undefined,
): bool {
  return IndexFunc(text, predicate) >= 0n;
}

export function HasPrefix(text: gostring, prefix: gostring): bool {
  return text.text().startsWith(prefix.text());
}

export function HasSuffix(text: gostring, suffix: gostring): bool {
  return text.text().endsWith(suffix.text());
}

export function Index(text: gostring, substring: gostring): int {
  return integerFromHost(text.text().indexOf(substring.text()));
}

export function Clone(text: gostring): gostring {
  return GoString.fromText(text.text());
}

export function Compare(left: gostring, right: gostring): int {
  const leftText = left.text();
  const rightText = right.text();
  return leftText === rightText ? 0n : leftText < rightText ? -1n : 1n;
}

export function ContainsAny(text: gostring, characters: gostring): bool {
  return IndexAny(text, characters) >= 0n;
}

export function ContainsRune(text: gostring, rune: int32): bool {
  return IndexRune(text, rune) >= 0n;
}

export function IndexRune(text: gostring, rune: int32): int {
  if (!validRune(rune)) {
    return -1n;
  }
  if (rune === 0xfffd) {
    for (let index = 0; index < Number(text.sourceLength()); ) {
      const [decoded, width] = decodeRuneAt(text, index);
      if (decoded === rune) {
        return integerFromHost(index);
      }
      index += Math.max(1, hostInteger(width));
    }
    return -1n;
  }
  return integerFromHost(text.text().indexOf(encodeRune(rune)));
}

export function Count(text: gostring, substring: gostring): int {
  if (Number(substring.sourceLength()) === 0) {
    return integerFromHost(runeCount(text) + 1);
  }
  let count = 0;
  for (let start = 0; ; ) {
    const index = text.text().indexOf(substring.text(), start);
    if (index < 0) {
      return integerFromHost(count);
    }
    count += 1;
    start = index + Number(substring.sourceLength());
  }
}

export function Cut(text: gostring, separator: gostring): [gostring, gostring, bool] {
  const index = text.text().indexOf(separator.text());
  return index < 0
    ? [text, GoString.empty, false]
    : [text.slice(0, index), text.slice(index + Number(separator.sourceLength())), true];
}

export function CutPrefix(text: gostring, prefix: gostring): [gostring, bool] {
  return text.text().startsWith(prefix.text()) ? [text.slice(Number(prefix.sourceLength())), true] : [text, false];
}

export function CutSuffix(text: gostring, suffix: gostring): [gostring, bool] {
  return text.text().endsWith(suffix.text()) ? [text.slice(0, Number(text.sourceLength()) - Number(suffix.sourceLength())), true] : [text, false];
}

export function EqualFold(left: gostring, right: gostring): bool {
  let leftIndex = 0;
  let rightIndex = 0;
  while (leftIndex < Number(left.sourceLength()) && rightIndex < Number(right.sourceLength())) {
    const [leftRune, leftWidth] = decodeRuneAt(left, leftIndex);
    const [rightRune, rightWidth] = decodeRuneAt(right, rightIndex);
    if (!runesEqualFold(leftRune, rightRune)) {
      return false;
    }
    leftIndex += Math.max(1, hostInteger(leftWidth));
    rightIndex += Math.max(1, hostInteger(rightWidth));
  }
  return leftIndex === Number(left.sourceLength()) && rightIndex === Number(right.sourceLength());
}

export function IndexAny(text: gostring, characters: gostring): int {
  const set = runeSet(characters);
  for (let index = 0; index < Number(text.sourceLength()); ) {
    const [rune, width] = decodeRuneAt(text, index);
    if (set.has(rune)) {
      return integerFromHost(index);
    }
    index += Math.max(1, hostInteger(width));
  }
  return -1n;
}

export function IndexByte(text: gostring, value: uint8): int {
  return integerFromHost(text.text().indexOf(String.fromCharCode(value)));
}

export function IndexFunc(
  text: gostring,
  predicate: ((rune: int32) => bool) | undefined,
): int {
  return findByPredicate(text, predicate, true, false);
}

export function LastIndex(text: gostring, substring: gostring): int {
  return integerFromHost(text.text().lastIndexOf(substring.text()));
}

export function LastIndexByte(text: gostring, value: uint8): int {
  return integerFromHost(text.text().lastIndexOf(String.fromCharCode(value)));
}

export function LastIndexFunc(
  text: gostring,
  predicate: ((rune: int32) => bool) | undefined,
): int {
  return findByPredicate(text, predicate, true, true);
}

export function findByPredicate(
  text: gostring,
  predicate: ((rune: int32) => bool) | undefined,
  expected: boolean,
  last: boolean,
): int {
  if (predicate === undefined && Number(text.sourceLength()) > 0) {
    GoPanic.raiseRuntime("call of nil predicate function");
  }
  let found = -1;
  for (let index = 0; index < Number(text.sourceLength()); ) {
    const [rune, width] = decodeRuneAt(text, index);
    if (predicate?.(rune) === expected) {
      found = index;
      if (!last) {
        return integerFromHost(found);
      }
    }
    index += Math.max(1, hostInteger(width));
  }
  return integerFromHost(found);
}

function runeSet(text: gostring): Set<int32> {
  const result = new Set<int32>();
  for (let index = 0; index < Number(text.sourceLength()); ) {
    const [rune, width] = decodeRuneAt(text, index);
    result.add(rune);
    index += Math.max(1, hostInteger(width));
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
