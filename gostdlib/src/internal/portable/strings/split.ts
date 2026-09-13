import { RuntimeSlice } from "@gotots/runtime/slice.js";
import { GoString } from "@gotots/runtime/string-value.js";
import type { gostring, int } from "@gotots/gostdlib/internal/scalars.js";

import { hostInteger } from "../../host-integer.js";

import { sliceValues } from "../../runtime/slice.js";
import { runeBoundaries } from "../utf8/codec.js";

export function Join(values: RuntimeSlice<gostring>, separator: gostring): gostring {
  const count = values.sourceLength();
  if (count === 0 || count === 0n) return GoString.empty;
  if (count === 1 || count === 1n) return values.get(0);
  return GoString.fromText(sliceValues(values).map((value) => value.text()).join(separator.text()));
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
  const separatorText = separator.text();
  if (separatorText.length === 0) {
    const boundaries = runeBoundaries(text);
    const runeCount = boundaries.length - 1;
    const hostCount = hostInteger(count);
    const partCount = count < 0n || hostCount > runeCount ? runeCount : hostCount;
    const parts: gostring[] = [];
    for (let index = 0; index < partCount; index += 1) {
      const start = boundaries[index];
      const end = index + 1 === partCount
        ? Number(text.sourceLength())
        : boundaries[index + 1];
      if (start === undefined || end === undefined) throw new RangeError("Missing rune boundary.");
      parts.push(text.slice(start, end));
    }
    return RuntimeSlice.literal(parts);
  }
  const parts: gostring[] = [];
  let remainder = text;
  const hostCount = count < 0n ? Number.POSITIVE_INFINITY : hostInteger(count);
  for (let index = 1; index < hostCount; index += 1) {
    const separatorIndex = remainder.text().indexOf(separatorText);
    if (separatorIndex < 0) {
      break;
    }
    parts.push(remainder.slice(0, separatorIndex));
    remainder = remainder.slice(separatorIndex + separatorText.length);
  }
  parts.push(remainder);
  return RuntimeSlice.literal(parts);
}
