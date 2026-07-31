import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { bool, uint8 } from "@gotots/runtime/scalars.js";

import { IsSpace } from "../unicode/properties.js";
import { decodeRuneAt } from "../utf8/codec.js";

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

export function TrimSpace(source: RuntimeSlice<uint8>): RuntimeSlice<uint8> {
  const byteString = toByteString(source);
  let start = 0;
  while (start < byteString.length) {
    const [rune, width] = decodeRuneAt(byteString, start);
    if (!IsSpace(rune)) {
      break;
    }
    start += Math.max(1, width);
  }

  let end = start;
  let lastNonSpace = start;
  while (end < byteString.length) {
    const [rune, width] = decodeRuneAt(byteString, end);
    end += Math.max(1, width);
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
