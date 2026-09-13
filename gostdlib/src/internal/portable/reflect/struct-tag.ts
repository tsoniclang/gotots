import { GoString } from "@gotots/runtime/string-value.js";
import type { gostring } from "../../scalars.js";
import { Unquote } from "../strconv/quote.js";

export function lookupStructTag(tagValue: gostring, key: gostring): [gostring, boolean] {
  const text = tagValue.text();
  const selectedKey = key.text();
  let offset = 0;
  while (offset < text.length) {
    while (text[offset] === " ") offset++;
    const start = offset;
    while (offset < text.length && text.charCodeAt(offset) > 0x20 &&
      text[offset] !== ":" && text[offset] !== '"' && text.charCodeAt(offset) !== 0x7f) {
      offset++;
    }
    if (offset === start || text[offset] !== ":" || text[offset + 1] !== '"') break;
    const name = text.slice(start, offset);
    const quoteStart = offset + 1;
    offset = quoteStart + 1;
    while (offset < text.length && text[offset] !== '"') {
      if (text[offset] === "\\") offset++;
      offset++;
    }
    if (offset >= text.length) break;
    offset++;
    if (name === selectedKey) {
      const [value, failure] = Unquote(tagValue.slice(quoteStart, offset));
      return failure === undefined ? [value, true] : [GoString.empty, false];
    }
  }
  return [GoString.empty, false];
}

export function getStructTag(tagValue: gostring, key: gostring): gostring {
  return lookupStructTag(tagValue, key)[0];
}
