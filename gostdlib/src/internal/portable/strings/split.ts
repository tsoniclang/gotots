import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring } from "@gotots/runtime/scalars.js";

import { sliceValues } from "../../runtime/slice.js";
import { runeBoundaries } from "../utf8/codec.js";

export function Join(values: RuntimeSlice<gostring>, separator: gostring): gostring {
  return sliceValues(values).join(separator);
}

export function Split(text: gostring, separator: gostring): RuntimeSlice<gostring> {
  if (separator.length === 0) {
    if (text.length === 0) {
      return RuntimeSlice.literal<gostring>([]);
    }
    const boundaries = runeBoundaries(text);
    const parts: gostring[] = [];
    for (let index = 0; index + 1 < boundaries.length; index += 1) {
      parts.push(text.slice(boundaries[index], boundaries[index + 1]));
    }
    return RuntimeSlice.literal(parts);
  }
  return RuntimeSlice.literal(text.split(separator));
}
