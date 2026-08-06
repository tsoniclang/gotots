import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { bool, gostring, int, uint8 } from "@gotots/gostdlib/internal/scalars.js";

import { hostInteger, integerFromHost } from "../../host-integer.js";

import { IsSpace } from "../unicode/properties.js";
import { decodeRuneAt } from "../utf8/codec.js";

export function Clone(source: RuntimeSlice<uint8>): RuntimeSlice<uint8> {
  if (source.isNil()) {
    return RuntimeSlice.nil<uint8>();
  }
  const target = RuntimeSlice.make<uint8>(source.length, null, 0);
  RuntimeSlice.copy(target, source);
  return target;
}

export function Compare(left: RuntimeSlice<uint8>, right: RuntimeSlice<uint8>): int {
  const count = Math.min(left.length, right.length);
  for (let index = 0; index < count; index += 1) {
    const compared = left.get(index) - right.get(index);
    if (compared !== 0) {
      return compared < 0 ? -1n : 1n;
    }
  }
  return left.length === right.length ? 0n : left.length < right.length ? -1n : 1n;
}

export function Cut(
  source: RuntimeSlice<uint8>,
  separator: RuntimeSlice<uint8>,
): [RuntimeSlice<uint8>, RuntimeSlice<uint8>, bool] {
  const index = indexOf(source, separator);
  return index < 0
    ? [source, RuntimeSlice.nil<uint8>(), false]
    : [
        source.slice(0, index, null),
        source.slice(index + separator.length, source.length, null),
        true,
      ];
}

export function Equal(left: RuntimeSlice<uint8>, right: RuntimeSlice<uint8>): bool {
  if (left.length !== right.length) {
    return false;
  }
  for (let index = 0; index < left.length; index += 1) {
    if (left.get(index) !== right.get(index)) {
      return false;
    }
  }
  return true;
}

export function IndexAny(source: RuntimeSlice<uint8>, characters: gostring): int {
  const selected = runeSet(characters);
  const value = toByteString(source);
  for (let index = 0; index < value.length; ) {
    const [rune, width] = decodeRuneAt(value, index);
    if (selected.has(rune)) {
      return integerFromHost(index);
    }
    index += Math.max(1, hostInteger(width));
  }
  return -1n;
}

export function IndexByte(source: RuntimeSlice<uint8>, value: uint8): int {
  for (let index = 0; index < source.length; index += 1) {
    if (source.get(index) === value) {
      return integerFromHost(index);
    }
  }
  return -1n;
}

export function Join(
  values: RuntimeSlice<RuntimeSlice<uint8>>,
  separator: RuntimeSlice<uint8>,
): RuntimeSlice<uint8> {
  if (values.length === 0) {
    return RuntimeSlice.literal<uint8>([]);
  }
  let result = RuntimeSlice.literal<uint8>([]);
  for (let outer = 0; outer < values.length; outer += 1) {
    if (outer !== 0) {
      result = appendSlice(result, separator);
    }
    result = appendSlice(result, values.get(outer));
  }
  return result;
}

export function LastIndexByte(source: RuntimeSlice<uint8>, value: uint8): int {
  for (let index = source.length - 1; index >= 0; index -= 1) {
    if (source.get(index) === value) {
      return integerFromHost(index);
    }
  }
  return -1n;
}

export function Trim(source: RuntimeSlice<uint8>, cutset: gostring): RuntimeSlice<uint8> {
  return trim(source, cutset, true, true);
}

export function TrimLeft(source: RuntimeSlice<uint8>, cutset: gostring): RuntimeSlice<uint8> {
  return trim(source, cutset, true, false);
}

export function TrimRight(source: RuntimeSlice<uint8>, cutset: gostring): RuntimeSlice<uint8> {
  return trim(source, cutset, false, true);
}

export function TrimSpace(source: RuntimeSlice<uint8>): RuntimeSlice<uint8> {
  const byteString = toByteString(source);
  let start = 0;
  while (start < byteString.length) {
    const [rune, width] = decodeRuneAt(byteString, start);
    if (!IsSpace(rune)) {
      break;
    }
    start += Math.max(1, hostInteger(width));
  }

  let end = start;
  let lastNonSpace = start;
  while (end < byteString.length) {
    const [rune, width] = decodeRuneAt(byteString, end);
    end += Math.max(1, hostInteger(width));
    if (!IsSpace(rune)) {
      lastNonSpace = end;
    }
  }
  return source.slice(start, lastNonSpace, null);
}

function indexOf(source: RuntimeSlice<uint8>, separator: RuntimeSlice<uint8>): number {
  if (separator.length === 0) {
    return 0;
  }
  const limit = source.length - separator.length;
  for (let start = 0; start <= limit; start += 1) {
    let matches = true;
    for (let index = 0; index < separator.length; index += 1) {
      if (source.get(start + index) !== separator.get(index)) {
        matches = false;
        break;
      }
    }
    if (matches) {
      return start;
    }
  }
  return -1;
}

function toByteString(source: RuntimeSlice<uint8>): string {
  let result = "";
  for (let index = 0; index < source.length; index += 1) {
    result += String.fromCharCode(source.get(index));
  }
  return result;
}

function appendSlice(
  target: RuntimeSlice<uint8>,
  source: RuntimeSlice<uint8>,
): RuntimeSlice<uint8> {
  const values: uint8[] = [];
  for (let index = 0; index < source.length; index += 1) {
    values.push(source.get(index));
  }
  return target.append(0, values);
}

function trim(
  source: RuntimeSlice<uint8>,
  cutset: gostring,
  left: boolean,
  right: boolean,
): RuntimeSlice<uint8> {
  const selected = runeSet(cutset);
  const value = toByteString(source);
  let start = 0;
  let end = value.length;
  if (left) {
    while (start < end) {
      const [rune, width] = decodeRuneAt(value, start);
      if (!selected.has(rune)) {
        break;
      }
      start += Math.max(1, hostInteger(width));
    }
  }
  if (right) {
    let index = start;
    let lastRetained = start;
    while (index < end) {
      const [rune, width] = decodeRuneAt(value, index);
      index += Math.max(1, hostInteger(width));
      if (!selected.has(rune)) {
        lastRetained = index;
      }
    }
    end = lastRetained;
  }
  return source.slice(start, end, null);
}

function runeSet(value: gostring): Set<number> {
  const result = new Set<number>();
  for (let index = 0; index < value.length; ) {
    const [rune, width] = decodeRuneAt(value, index);
    result.add(rune);
    index += Math.max(1, hostInteger(width));
  }
  return result;
}
