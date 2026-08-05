import type { gostring } from "@gotots/gostdlib/internal/scalars.js";
import { hostInteger } from "../../host-integer.js";

import {
  decodeRuneAt,
  toHostString,
} from "../utf8/codec.js";

export type CompiledPattern = {
  readonly source: string;
  readonly flags: string;
  readonly expression: gostring;
  readonly names: ReadonlyMap<string, number>;
};

export function translatePattern(expression: gostring): CompiledPattern {
  assertValidPatternBytes(expression);
  let source = toHostString(expression);
  const selectedFlags = new Set<string>(["d", "g", "u"]);
  let ungreedy = false;
  for (;;) {
    const flags = /^\(\?([imsU]+)\)/.exec(source);
    if (flags === null) {
      break;
    }
    for (const flag of flags[1] ?? "") {
      if (flag === "U") {
        ungreedy = true;
      } else {
        selectedFlags.add(flag);
      }
    }
    source = source.slice(flags[0].length);
  }
  rejectUnsupportedSyntax(source);
  const names = captureNames(source);
  source = rewriteQuotedLiterals(source);
  source = source.replaceAll(/\\x\{([0-9a-fA-F]+)\}/g, "\\u{$1}");
  source = source.replaceAll("\\A", "^");
  source = source.replaceAll("\\z", "(?![\\s\\S])");
  source = source.replaceAll("\\a", "\\x07");
  source = source.replaceAll("\\e", "\\x1b");
  source = source.replaceAll("(?P<", "(?<");
  source = rewriteAsciiClasses(source);
  source = rewriteLooseBraces(source);
  if (!selectedFlags.has("m")) {
    source = rewriteEndAnchors(source);
  }
  if (ungreedy) {
    source = invertGreediness(source);
  }
  return {
    source,
    flags: [...selectedFlags].sort().join(""),
    expression,
    names,
  };
}

function assertValidPatternBytes(expression: gostring): void {
  for (let index = 0; index < expression.length; ) {
    const [rune, width] = decodeRuneAt(expression, index);
    if (rune === 0xfffd && width === 1n && expression.charCodeAt(index) >= 0x80) {
      throw new Error("invalid UTF-8");
    }
    index += Math.max(1, hostInteger(width));
  }
}

function rejectUnsupportedSyntax(source: string): void {
  if (/\(\?(?:=|!|<=|<!)/.test(source)) {
    throw new Error("invalid or unsupported Perl syntax");
  }
  if (/\\[1-9]/.test(source) || /\\k</.test(source)) {
    throw new Error("invalid escape sequence");
  }
  if (/\(\?[imsU-]+:/.test(source)) {
    throw new Error("scoped regexp flags require the portable RE2 engine");
  }
}

function rewriteQuotedLiterals(source: string): string {
  return source.replaceAll(/\\Q([\s\S]*?)\\E/g, (_whole: string, literal: string): string =>
    literal.replaceAll(/[\\^$.*+?()[\]{}|]/g, "\\$&"),
  );
}

function rewriteAsciiClasses(source: string): string {
  const normalized = source
    .replaceAll("[:digit:]", "0-9")
    .replaceAll("[:space:]", "\\t\\n\\v\\f\\r ")
    .replaceAll("[:word:]", "0-9A-Za-z_");
  let result = "";
  for (let index = 0; index < normalized.length; index += 1) {
    const character = normalized[index] ?? "";
    if (character === "\\") {
      const escaped = normalized[index + 1] ?? "";
      result += outsideAsciiClass(escaped) ?? `${character}${escaped}`;
      index += escaped.length === 0 ? 0 : 1;
      continue;
    }
    if (character !== "[") {
      result += character;
      continue;
    }
    const end = characterClassEnd(normalized, index);
    result += rewriteCharacterClass(normalized.slice(index + 1, end));
    index = end;
  }
  return result;
}

function outsideAsciiClass(escaped: string): string | undefined {
  switch (escaped) {
    case "d":
      return "[0-9]";
    case "D":
      return "[^0-9]";
    case "s":
      return "[\\t\\n\\f\\r ]";
    case "S":
      return "[^\\t\\n\\f\\r ]";
    case "w":
      return "[0-9A-Za-z_]";
    case "W":
      return "[^0-9A-Za-z_]";
    default:
      return undefined;
  }
}

function characterClassEnd(source: string, start: number): number {
  let escaped = false;
  for (let index = start + 1; index < source.length; index += 1) {
    const character = source[index] ?? "";
    if (escaped) {
      escaped = false;
    } else if (character === "\\") {
      escaped = true;
    } else if (character === "]" && index > start + 1) {
      return index;
    }
  }
  throw new Error("unterminated character class");
}

function rewriteCharacterClass(body: string): string {
  const negated = body.startsWith("^");
  const content = negated ? body.slice(1) : body;
  if (content === "\\s\\S" || content === "\\S\\s") {
    return "[\\s\\S]";
  }
  const complement = singleComplementClass(content);
  if (complement !== undefined) {
    return negated ? complement.positive : complement.negative;
  }

  let result = "";
  for (let index = 0; index < content.length; index += 1) {
    const character = content[index] ?? "";
    if (character !== "\\") {
      result += character;
      continue;
    }
    const escaped = content[index + 1] ?? "";
    switch (escaped) {
      case "d":
        result += "0-9";
        break;
      case "s":
        result += "\\t\\n\\f\\r ";
        break;
      case "w":
        result += "0-9A-Za-z_";
        break;
      case "D":
      case "S":
      case "W":
        throw new Error("complement character class cannot be combined");
      default:
        result += `${character}${escaped}`;
        break;
    }
    index += escaped.length === 0 ? 0 : 1;
  }
  return `[${negated ? "^" : ""}${result}]`;
}

function singleComplementClass(
  content: string,
): { readonly positive: string; readonly negative: string } | undefined {
  switch (content) {
    case "\\D":
      return { positive: "[0-9]", negative: "[^0-9]" };
    case "\\S":
      return { positive: "[\\t\\n\\f\\r ]", negative: "[^\\t\\n\\f\\r ]" };
    case "\\W":
      return { positive: "[0-9A-Za-z_]", negative: "[^0-9A-Za-z_]" };
    default:
      return undefined;
  }
}

function rewriteLooseBraces(source: string): string {
  let result = "";
  let escaped = false;
  let inClass = false;
  for (let index = 0; index < source.length; index += 1) {
    const character = source[index] ?? "";
    if (escaped) {
      result += character;
      escaped = false;
    } else if (character === "\\") {
      result += character;
      escaped = true;
    } else if (character === "[") {
      result += character;
      inClass = true;
    } else if (character === "]") {
      result += character;
      inClass = false;
    } else if (character === "{" && !inClass) {
      const quantifier = /^\{[0-9]+(?:,[0-9]*)?\}/.exec(source.slice(index));
      const unicodeEscape = result.endsWith("\\u")
        ? /^\{[0-9a-fA-F]+\}/.exec(source.slice(index))
        : null;
      if (quantifier === null && unicodeEscape === null) {
        result += "\\{";
      } else {
        const braces = quantifier?.[0] ?? unicodeEscape?.[0] ?? "";
        result += braces;
        index += braces.length - 1;
      }
    } else if (character === "}" && !inClass) {
      result += "\\}";
    } else {
      result += character;
    }
  }
  return result;
}

function rewriteEndAnchors(source: string): string {
  let result = "";
  let escaped = false;
  let inClass = false;
  for (const character of source) {
    if (escaped) {
      result += character;
      escaped = false;
    } else if (character === "\\") {
      result += character;
      escaped = true;
    } else if (character === "[") {
      result += character;
      inClass = true;
    } else if (character === "]") {
      result += character;
      inClass = false;
    } else if (character === "$" && !inClass) {
      result += "(?![\\s\\S])";
    } else {
      result += character;
    }
  }
  return result;
}

function invertGreediness(source: string): string {
  let result = "";
  let escaped = false;
  let inClass = false;
  for (let index = 0; index < source.length; index += 1) {
    const character = source[index] ?? "";
    if (escaped) {
      result += character;
      escaped = false;
    } else if (character === "\\") {
      result += character;
      escaped = true;
    } else if (character === "[") {
      result += character;
      inClass = true;
    } else if (character === "]") {
      result += character;
      inClass = false;
    } else if (!inClass && (character === "*" || character === "+" || character === "?")) {
      if (source[index + 1] === "?") {
        result += character;
        index += 1;
      } else {
        result += `${character}?`;
      }
    } else {
      result += character;
    }
  }
  return result;
}

function captureNames(source: string): ReadonlyMap<string, number> {
  const names = new Map<string, number>();
  let captures = 0;
  let escaped = false;
  let inClass = false;
  for (let index = 0; index < source.length; index += 1) {
    const character = source[index] ?? "";
    if (escaped) {
      escaped = false;
    } else if (character === "\\") {
      escaped = true;
    } else if (character === "[") {
      inClass = true;
    } else if (character === "]") {
      inClass = false;
    } else if (character === "(" && !inClass && source[index + 1] !== "?") {
      captures += 1;
    } else if (character === "(" && !inClass && source.startsWith("(?P<", index)) {
      captures += 1;
      const end = source.indexOf(">", index + 4);
      if (end > index) {
        names.set(source.slice(index + 4, end), captures);
      }
    } else if (character === "(" && !inClass && source.startsWith("(?<", index)) {
      captures += 1;
      const end = source.indexOf(">", index + 3);
      if (end > index) {
        names.set(source.slice(index + 3, end), captures);
      }
    }
  }
  return names;
}
