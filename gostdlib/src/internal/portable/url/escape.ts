import { GoString } from "@gotots/runtime/string-value.js";

const upperHex = "0123456789ABCDEF";

export const enum EscapeMode {
  Path,
  PathSegment,
  UserPassword,
  QueryComponent,
  Fragment,
  Host,
  Zone,
}

function isAlphaNumeric(byte: number): boolean {
  return (
    (byte >= 0x61 && byte <= 0x7a) ||
    (byte >= 0x41 && byte <= 0x5a) ||
    (byte >= 0x30 && byte <= 0x39)
  );
}

function shouldEscape(byte: number, mode: EscapeMode): boolean {
  if (isAlphaNumeric(byte) || byte === 0x2d || byte === 0x5f || byte === 0x2e || byte === 0x7e) {
    return false;
  }

  if (mode === EscapeMode.Host || mode === EscapeMode.Zone) {
    return !(
      byte === 0x21 ||
      byte === 0x24 ||
      byte === 0x26 ||
      byte === 0x27 ||
      byte === 0x28 ||
      byte === 0x29 ||
      byte === 0x2a ||
      byte === 0x2b ||
      byte === 0x2c ||
      byte === 0x3b ||
      byte === 0x3a ||
      byte === 0x3d ||
      byte === 0x5b ||
      byte === 0x5d ||
      byte === 0x3c ||
      byte === 0x3e ||
      byte === 0x22
    );
  }

  switch (byte) {
    case 0x24:
    case 0x26:
    case 0x2b:
    case 0x2c:
    case 0x2f:
    case 0x3a:
    case 0x3b:
    case 0x3d:
    case 0x3f:
    case 0x40:
      switch (mode) {
        case EscapeMode.Path:
          return byte === 0x3f;
        case EscapeMode.PathSegment:
          return byte === 0x2f || byte === 0x3b || byte === 0x2c || byte === 0x3f;
        case EscapeMode.UserPassword:
          return byte === 0x40 || byte === 0x2f || byte === 0x3f || byte === 0x3a || byte === 0x3b || byte === 0x2c;
        case EscapeMode.QueryComponent:
          return true;
        case EscapeMode.Fragment:
          return false;
        default:
          return true;
      }
    default:
      return true;
  }
}

function encode(source: GoString, mode: EscapeMode): GoString {
  const text = source.text();
  let result = "";
  let changed = false;
  for (let index = 0; index < text.length; index++) {
    const byte = text.charCodeAt(index);
    if (byte === 0x20 && mode === EscapeMode.QueryComponent) {
      changed = true;
      result += "+";
    } else if (shouldEscape(byte, mode)) {
      changed = true;
      result += `%${upperHex[byte >> 4]}${upperHex[byte & 0x0f]}`;
    } else {
      result += String.fromCharCode(byte);
    }
  }
  return changed ? GoString.fromText(result) : source;
}

function hexValue(character: string): number {
  const code = character.charCodeAt(0);
  if (code >= 0x30 && code <= 0x39) {
    return code - 0x30;
  }
  if (code >= 0x41 && code <= 0x46) {
    return code - 0x41 + 10;
  }
  if (code >= 0x61 && code <= 0x66) {
    return code - 0x61 + 10;
  }
  return -1;
}

export function decode(source: GoString, mode: EscapeMode): GoString {
  const text = source.text();
  let decoded = "";
  let changed = false;
  for (let index = 0; index < text.length; index++) {
    const character = text[index];
    if (character === "%") {
      const high = text[index + 1];
      const low = text[index + 2];
      if (high === undefined || low === undefined || hexValue(high) < 0 || hexValue(low) < 0) {
        throw new URIError(`invalid URL escape ${JSON.stringify(text.slice(index, index + 3))}`);
      }
      const value = (hexValue(high) << 4) | hexValue(low);
      if (mode === EscapeMode.Host && value < 0x80 && value !== 0x25) {
        throw new URIError(`invalid URL escape ${JSON.stringify(text.slice(index, index + 3))}`);
      }
      decoded += String.fromCharCode(value);
      changed = true;
      index += 2;
    } else if (character === "+" && mode === EscapeMode.QueryComponent) {
      decoded += " ";
      changed = true;
    } else {
      decoded += text.charAt(index);
    }
  }
  return changed ? GoString.fromText(decoded) : source;
}

export function escapePath(source: GoString): GoString {
  return encode(source, EscapeMode.Path);
}

export function escapePathSegment(source: GoString): GoString {
  return encode(source, EscapeMode.PathSegment);
}

export function escapeQuery(source: GoString): GoString {
  return encode(source, EscapeMode.QueryComponent);
}

export function escapeFragment(source: GoString): GoString {
  return encode(source, EscapeMode.Fragment);
}

export function escapeUserPassword(source: GoString): GoString {
  return encode(source, EscapeMode.UserPassword);
}

export function decodedPathMatches(rawPath: GoString, path: GoString): boolean {
  try {
    return decode(rawPath, EscapeMode.Path).text() === path.text();
  } catch {
    return false;
  }
}
