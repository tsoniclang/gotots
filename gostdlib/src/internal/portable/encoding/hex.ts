import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring, uint8 } from "@gotots/runtime/scalars.js";

const digits = "0123456789abcdef";

export function EncodeToString(source: RuntimeSlice<uint8>): gostring {
  let result = "";
  for (let index = 0; index < source.length; index += 1) {
    const byte = source.get(index);
    result += digits.charAt(Math.floor(byte / 16)) + digits.charAt(byte % 16);
  }
  return result;
}
