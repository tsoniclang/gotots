import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring, int } from "@gotots/gostdlib/internal/scalars.js";

import { hostInteger } from "../../host-integer.js";

import { sliceValues } from "../../runtime/slice.js";
import { runeBoundaries } from "../utf8/codec.js";

export function Join(values: RuntimeSlice<gostring>, separator: gostring): gostring {
  return sliceValues(values).join(separator);
}

export function Split(text: gostring, separator: gostring): RuntimeSlice<gostring> {
  return SplitN(text, separator, -1n);
}

export function SplitN(
  text: gostring,
  separator: gostring,
  count: int,
): RuntimeSlice<gostring> {
  if (count === 0n) {
    return RuntimeSlice.nil<gostring>();
  }
  if (separator.length === 0) {
    const boundaries = runeBoundaries(text);
    const runeCount = boundaries.length - 1;
    const hostCount = hostInteger(count);
    const partCount = count < 0n || hostCount > runeCount ? runeCount : hostCount;
    const parts: gostring[] = [];
    for (let index = 0; index < partCount; index += 1) {
      const start = boundaries[index];
      const end = index + 1 === partCount
        ? text.length
        : boundaries[index + 1];
      parts.push(text.slice(start, end));
    }
    return RuntimeSlice.literal(parts);
  }
  if (count < 0n) {
    return RuntimeSlice.literal(text.split(separator));
  }
  const parts: gostring[] = [];
  let remainder = text;
  const hostCount = hostInteger(count);
  for (let index = 1; index < hostCount; index += 1) {
    const separatorIndex = remainder.indexOf(separator);
    if (separatorIndex < 0) {
      break;
    }
    parts.push(remainder.slice(0, separatorIndex));
    remainder = remainder.slice(separatorIndex + separator.length);
  }
  parts.push(remainder);
  return RuntimeSlice.literal(parts);
}
