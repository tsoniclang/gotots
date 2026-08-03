import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring } from "@gotots/runtime/scalars.js";

import { sliceValues } from "../../runtime/slice.js";
import { runeBoundaries } from "../utf8/codec.js";

export function Join(values: RuntimeSlice<gostring>, separator: gostring): gostring {
  return sliceValues(values).join(separator);
}

export function Split(text: gostring, separator: gostring): RuntimeSlice<gostring> {
  return SplitN(text, separator, -1);
}

export function SplitN(
  text: gostring,
  separator: gostring,
  count: number,
): RuntimeSlice<gostring> {
  if (count === 0) {
    return RuntimeSlice.nil<gostring>();
  }
  if (separator.length === 0) {
    const boundaries = runeBoundaries(text);
    const runeCount = boundaries.length - 1;
    const partCount = count < 0 || count > runeCount ? runeCount : count;
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
  if (count < 0) {
    return RuntimeSlice.literal(text.split(separator));
  }
  const parts: gostring[] = [];
  let remainder = text;
  for (let index = 1; index < count; index += 1) {
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
