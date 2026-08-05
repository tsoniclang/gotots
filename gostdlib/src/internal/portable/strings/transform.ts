import { GoPanic } from "@gotots/runtime/panic.js";
import type { bool, gostring, int, int32 } from "@gotots/gostdlib/internal/scalars.js";

import { hostInteger } from "../../host-integer.js";

import { ToLower as lowerRune, ToUpper as upperRune } from "../unicode/case.js";
import { IsSpace } from "../unicode/properties.js";
import {
  decodeRuneAt,
  encodeRune,
  runeBoundaries,
} from "../utf8/codec.js";
import { findByPredicate } from "./search.js";

export function Map(
  mapping: ((rune: int32) => int32) | undefined,
  text: gostring,
): gostring {
  if (mapping === undefined && text.length > 0) {
    GoPanic.raiseRuntime("call of nil mapping function");
  }
  let result = "";
  for (let index = 0; index < text.length; ) {
    const [rune, width] = decodeRuneAt(text, index);
    const mapped = mapping?.(rune) ?? rune;
    if (mapped >= 0) {
      result += encodeRune(mapped);
    }
    index += Math.max(1, hostInteger(width));
  }
  return result;
}

export function Repeat(text: gostring, count: int): gostring {
  if (count < 0n) {
    GoPanic.raiseRuntime("strings: negative Repeat count");
  }
  const hostCount = hostInteger(count);
  if (text.length !== 0 && hostCount > Math.floor(0x1fffffff / text.length)) {
    GoPanic.raiseRuntime("strings: Repeat output length overflow");
  }
  return text.repeat(hostCount);
}

export function Replace(
  text: gostring,
  oldText: gostring,
  newText: gostring,
  count: int,
): gostring {
  if (count === 0n) {
    return text;
  }
  const limit = count < 0n ? Number.POSITIVE_INFINITY : hostInteger(count);
  if (oldText.length === 0) {
    const boundaries = runeBoundaries(text);
    let result = "";
    let replacements = 0;
    for (let index = 0; index < boundaries.length; index += 1) {
      const boundary = boundaries[index] ?? text.length;
      if (replacements < limit) {
        result += newText;
        replacements += 1;
      }
      const next = boundaries[index + 1];
      if (next !== undefined) {
        result += text.slice(boundary, next);
      }
    }
    return result;
  }

  let result = "";
  let start = 0;
  let replacements = 0;
  while (replacements < limit) {
    const index = text.indexOf(oldText, start);
    if (index < 0) {
      break;
    }
    result += text.slice(start, index) + newText;
    start = index + oldText.length;
    replacements += 1;
  }
  return replacements === 0 ? text : result + text.slice(start);
}

export function ReplaceAll(text: gostring, oldText: gostring, newText: gostring): gostring {
  return Replace(text, oldText, newText, -1n);
}

export function ToLower(text: gostring): gostring {
  return Map(lowerRune, text);
}

export function ToUpper(text: gostring): gostring {
  return Map(upperRune, text);
}

export function ToValidUTF8(text: gostring, replacement: gostring): gostring {
  let result = "";
  let invalid = false;
  for (let index = 0; index < text.length; ) {
    const [rune, width] = decodeRuneAt(text, index);
    if (rune === 0xfffd && width === 1n && text.charCodeAt(index) >= 0x80) {
      if (!invalid) {
        result += replacement;
        invalid = true;
      }
      index += 1;
      continue;
    }
    invalid = false;
    result += text.slice(index, index + Math.max(1, hostInteger(width)));
    index += Math.max(1, hostInteger(width));
  }
  return result;
}

export function Trim(text: gostring, cutset: gostring): gostring {
  const set = cutsetPredicate(cutset);
  return TrimFunc(text, set);
}

export function TrimLeft(text: gostring, cutset: gostring): gostring {
  return TrimLeftFunc(text, cutsetPredicate(cutset));
}

export function TrimRight(text: gostring, cutset: gostring): gostring {
  return TrimRightFunc(text, cutsetPredicate(cutset));
}

export function TrimFunc(
  text: gostring,
  predicate: ((rune: int32) => bool) | undefined,
): gostring {
  return TrimRightFunc(TrimLeftFunc(text, predicate), predicate);
}

export function TrimLeftFunc(
  text: gostring,
  predicate: ((rune: int32) => bool) | undefined,
): gostring {
  const index = findByPredicate(text, predicate, false, false);
  return index < 0n ? "" : text.slice(hostInteger(index));
}

export function TrimRightFunc(
  text: gostring,
  predicate: ((rune: int32) => bool) | undefined,
): gostring {
  const index = findByPredicate(text, predicate, false, true);
  if (index < 0n) {
    return "";
  }
  const hostIndex = hostInteger(index);
  const [, width] = decodeRuneAt(text, hostIndex);
  return text.slice(0, hostIndex + Math.max(1, hostInteger(width)));
}

export function TrimSpace(text: gostring): gostring {
  return TrimFunc(text, IsSpace);
}

export function TrimPrefix(text: gostring, prefix: gostring): gostring {
  return text.startsWith(prefix) ? text.slice(prefix.length) : text;
}

export function TrimSuffix(text: gostring, suffix: gostring): gostring {
  return text.endsWith(suffix) ? text.slice(0, text.length - suffix.length) : text;
}

function cutsetPredicate(cutset: gostring): (rune: int32) => bool {
  const runes = new Set<int32>();
  for (let index = 0; index < cutset.length; ) {
    const [rune, width] = decodeRuneAt(cutset, index);
    runes.add(rune);
    index += Math.max(1, hostInteger(width));
  }
  return (rune: int32): bool => runes.has(rune);
}
