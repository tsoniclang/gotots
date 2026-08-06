function escapedCharacter(source: string, index: number): readonly [string, number] | undefined {
  const escape = source[index + 1];
  if (escape === undefined) {
    return undefined;
  }
  const simple = new Map<string, string>([
    ["a", "\u0007"],
    ["b", "\b"],
    ["f", "\f"],
    ["n", "\n"],
    ["r", "\r"],
    ["t", "\t"],
    ["v", "\u000b"],
    ["\\", "\\"],
    ["\"", "\""],
  ]);
  const simpleValue = simple.get(escape);
  if (simpleValue !== undefined) {
    return [simpleValue, index + 2];
  }

  const width = escape === "x" ? 2 : escape === "u" ? 4 : escape === "U" ? 8 : 0;
  if (width > 0) {
    const digits = source.slice(index + 2, index + 2 + width);
    if (digits.length !== width || !/^[0-9A-Fa-f]+$/.test(digits)) {
      return undefined;
    }
    const codePoint = Number.parseInt(digits, 16);
    if (codePoint > 0x10ffff || (codePoint >= 0xd800 && codePoint <= 0xdfff)) {
      return undefined;
    }
    return [String.fromCodePoint(codePoint), index + 2 + width];
  }

  if (escape >= "0" && escape <= "7") {
    const digits = source.slice(index + 1, index + 4);
    if (digits.length !== 3 || !/^[0-7]{3}$/.test(digits)) {
      return undefined;
    }
    const value = Number.parseInt(digits, 8);
    if (value > 0xff) {
      return undefined;
    }
    return [String.fromCharCode(value), index + 4];
  }
  return undefined;
}

function unquote(source: string): string | undefined {
  if (source.length < 2 || source[0] !== "\"" || source[source.length - 1] !== "\"") {
    return undefined;
  }
  let result = "";
  for (let index = 1; index < source.length - 1; ) {
    const character = source[index] ?? "";
    if (character !== "\\") {
      result += character;
      index += 1;
      continue;
    }
    const escaped = escapedCharacter(source, index);
    if (escaped === undefined) {
      return undefined;
    }
    result += escaped[0];
    index = escaped[1];
  }
  return result;
}

export function lookupStructTag(tagValue: string, key: string): [string, boolean] {
  let tag = tagValue;
  while (tag !== "") {
    tag = tag.replace(/^ +/, "");
    if (tag === "") {
      break;
    }

    let colon = 0;
    while (
      colon < tag.length &&
      (tag.charCodeAt(colon) > 0x20) &&
      tag[colon] !== ":" &&
      tag[colon] !== "\"" &&
      tag.charCodeAt(colon) !== 0x7f
    ) {
      colon += 1;
    }
    if (colon === 0 || colon + 1 >= tag.length || tag[colon] !== ":" || tag[colon + 1] !== "\"") {
      break;
    }
    const name = tag.slice(0, colon);
    tag = tag.slice(colon + 1);

    let quoteEnd = 1;
    while (quoteEnd < tag.length && tag[quoteEnd] !== "\"") {
      if (tag[quoteEnd] === "\\") {
        quoteEnd += 1;
      }
      quoteEnd += 1;
    }
    if (quoteEnd >= tag.length) {
      break;
    }
    const quotedValue = tag.slice(0, quoteEnd + 1);
    tag = tag.slice(quoteEnd + 1);
    if (name === key) {
      const value = unquote(quotedValue);
      return value === undefined ? ["", false] : [value, true];
    }
  }
  return ["", false];
}

export function getStructTag(tagValue: string, key: string): string {
  return lookupStructTag(tagValue, key)[0];
}
