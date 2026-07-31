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

function encode(source: string, mode: EscapeMode): string {
  const bytes = new TextEncoder().encode(source);
  let result = "";
  for (const byte of bytes) {
    if (byte === 0x20 && mode === EscapeMode.QueryComponent) {
      result += "+";
    } else if (shouldEscape(byte, mode)) {
      result += `%${upperHex[byte >> 4]}${upperHex[byte & 0x0f]}`;
    } else {
      result += String.fromCharCode(byte);
    }
  }
  return result;
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

export function decode(source: string, mode: EscapeMode): string {
  const bytes: number[] = [];
  for (let index = 0; index < source.length; index += 1) {
    const character = source[index] ?? "";
    if (character === "%") {
      const high = source[index + 1];
      const low = source[index + 2];
      if (high === undefined || low === undefined || hexValue(high) < 0 || hexValue(low) < 0) {
        throw new URIError(`invalid URL escape ${JSON.stringify(source.slice(index, index + 3))}`);
      }
      const value = (hexValue(high) << 4) | hexValue(low);
      if (mode === EscapeMode.Host && value < 0x80 && value !== 0x25) {
        throw new URIError(`invalid URL escape ${JSON.stringify(source.slice(index, index + 3))}`);
      }
      bytes.push(value);
      index += 2;
    } else if (character === "+" && mode === EscapeMode.QueryComponent) {
      bytes.push(0x20);
    } else {
      const codePoint = source.codePointAt(index);
      if (codePoint === undefined) {
        break;
      }
      bytes.push(...new TextEncoder().encode(String.fromCodePoint(codePoint)));
      if (codePoint > 0xffff) {
        index += 1;
      }
    }
  }
  return new TextDecoder().decode(Uint8Array.from(bytes));
}

export function escapePath(source: string): string {
  return encode(source, EscapeMode.Path);
}

export function escapePathSegment(source: string): string {
  return encode(source, EscapeMode.PathSegment);
}

export function escapeQuery(source: string): string {
  return encode(source, EscapeMode.QueryComponent);
}

export function escapeFragment(source: string): string {
  return encode(source, EscapeMode.Fragment);
}

export function escapeUserPassword(source: string): string {
  return encode(source, EscapeMode.UserPassword);
}

export function decodedPathMatches(rawPath: string, path: string): boolean {
  try {
    return decode(rawPath, EscapeMode.Path) === path;
  } catch {
    return false;
  }
}
