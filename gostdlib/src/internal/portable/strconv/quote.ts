import type { GoError } from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { gostring, int32, uint8 } from "@gotots/runtime/scalars.js";

import { decodeRuneAt, encodeRune, RuneError, validRune } from "../utf8/codec.js";
import { ErrSyntax } from "./number-error.js";

export function AppendQuote(
  target: RuntimeSlice<uint8>,
  value: gostring,
): RuntimeSlice<uint8> {
  return target.append(0, bytes(Quote(value)));
}

export function Quote(value: gostring): gostring {
  let result = '"';
  for (let index = 0; index < value.length; ) {
    const [rune, width] = decodeRuneAt(value, index);
    if (rune === RuneError && width === 1) {
      result += `\\x${hex(value.charCodeAt(index), 2)}`;
      index += 1;
      continue;
    }
    result += quoteRuneBody(rune, '"');
    index += width;
  }
  return `${result}"`;
}

export function QuoteRune(rune: int32): gostring {
  const selected = validRune(rune) ? rune : RuneError;
  return `'${quoteRuneBody(selected, "'")}'`;
}

export function Unquote(value: gostring): [gostring, GoError | undefined] {
  if (value.length < 2) {
    return ["", ErrSyntax];
  }
  const quote = value[0];
  if (quote === "`" && value.at(-1) === "`") {
    const body = value.slice(1, -1);
    return body.includes("`") ? ["", ErrSyntax] : [body.replaceAll("\r", ""), undefined];
  }
  if ((quote !== '"' && quote !== "'") || value.at(-1) !== quote) {
    return ["", ErrSyntax];
  }

  const end = value.length - 1;
  let index = 1;
  let result = "";
  let runes = 0;
  while (index < end) {
    const byte = value.charCodeAt(index);
    if (byte === 0x0a || byte === 0x0d) {
      return ["", ErrSyntax];
    }
    if (byte === 0x5c) {
      const escaped = unquoteEscape(value, index + 1, quote);
      if (escaped === undefined || escaped.next > end) {
        return ["", ErrSyntax];
      }
      result += escaped.value;
      index = escaped.next;
      runes += 1;
      continue;
    }
    if (value[index] === quote) {
      return ["", ErrSyntax];
    }
    const [rune, width] = decodeRuneAt(value, index);
    if (rune === RuneError && width === 1) {
      return ["", ErrSyntax];
    }
    result += value.slice(index, index + width);
    index += width;
    runes += 1;
  }
  if (quote === "'" && runes !== 1) {
    return ["", ErrSyntax];
  }
  return [result, undefined];
}

type UnquotedEscape = {
  readonly value: gostring;
  readonly next: number;
};

function unquoteEscape(
  source: gostring,
  index: number,
  quote: string,
): UnquotedEscape | undefined {
  const simple = source[index];
  const replacements: Readonly<Record<string, string>> = {
    a: "\x07",
    b: "\b",
    f: "\f",
    n: "\n",
    r: "\r",
    t: "\t",
    v: "\x0b",
    "\\": "\\",
    "'": "'",
    '"': '"',
  };
  const replacement = simple === undefined ? undefined : replacements[simple];
  if (replacement !== undefined) {
    if ((simple === "'" || simple === '"') && simple !== quote) {
      return undefined;
    }
    return { value: replacement, next: index + 1 };
  }
  if (simple === "x") {
    return numericEscape(source, index + 1, 2, 16, false);
  }
  if (simple === "u") {
    return numericEscape(source, index + 1, 4, 16, true);
  }
  if (simple === "U") {
    return numericEscape(source, index + 1, 8, 16, true);
  }
  if (simple !== undefined && simple >= "0" && simple <= "7") {
    return numericEscape(source, index, 3, 8, false);
  }
  return undefined;
}

function numericEscape(
  source: gostring,
  index: number,
  width: number,
  base: number,
  unicode: boolean,
): UnquotedEscape | undefined {
  const digits = source.slice(index, index + width);
  if (digits.length !== width || !validDigits(digits, base)) {
    return undefined;
  }
  const value = Number.parseInt(digits, base);
  if ((!unicode && value > 0xff) || (unicode && !validRune(value))) {
    return undefined;
  }
  return {
    value: unicode ? encodeRune(value) : String.fromCharCode(value),
    next: index + width,
  };
}

function quoteRuneBody(rune: int32, quote: string): gostring {
  switch (rune) {
  case 0x07: return "\\a";
  case 0x08: return "\\b";
  case 0x09: return "\\t";
  case 0x0a: return "\\n";
  case 0x0b: return "\\v";
  case 0x0c: return "\\f";
  case 0x0d: return "\\r";
  case 0x5c: return "\\\\";
  }
  if (rune === quote.charCodeAt(0)) {
    return `\\${quote}`;
  }
  if (isPrint(rune)) {
    return encodeRune(rune);
  }
  if (rune < 0x100) {
    return `\\x${hex(rune, 2)}`;
  }
  if (rune < 0x10000) {
    return `\\u${hex(rune, 4)}`;
  }
  return `\\U${hex(rune, 8)}`;
}

function isPrint(rune: int32): boolean {
  if (rune === 0x20 || (rune >= 0x21 && rune <= 0x7e)) {
    return true;
  }
  return /[\p{L}\p{M}\p{N}\p{P}\p{S}]/u.test(String.fromCodePoint(rune));
}

function validDigits(value: string, base: number): boolean {
  const pattern = base === 16 ? /^[0-9a-fA-F]+$/ : /^[0-7]+$/;
  return pattern.test(value);
}

function hex(value: number, width: number): string {
  return value.toString(16).padStart(width, "0");
}

function bytes(value: gostring): uint8[] {
  return [...value].map((character) => character.charCodeAt(0));
}
