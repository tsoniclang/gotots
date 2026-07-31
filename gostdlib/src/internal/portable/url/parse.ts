import type { bool, gostring } from "@gotots/runtime/scalars.js";

import {
  decode,
  decodedPathMatches,
  escapeFragment,
  escapePath,
  EscapeMode,
} from "./escape.js";

export interface ParsedUserinfo {
  username: gostring;
  password: gostring;
  passwordPresent: bool;
}

export interface ParsedURL {
  Scheme: gostring;
  Opaque: gostring;
  User: ParsedUserinfo | undefined;
  Host: gostring;
  Path: gostring;
  Fragment: gostring;
  RawQuery: gostring;
  RawPath: gostring;
  RawFragment: gostring;
  ForceQuery: bool;
  OmitHost: bool;
}

function emptyURL(): ParsedURL {
  return {
    Scheme: "",
    Opaque: "",
    User: undefined,
    Host: "",
    Path: "",
    Fragment: "",
    RawQuery: "",
    RawPath: "",
    RawFragment: "",
    ForceQuery: false,
    OmitHost: false,
  };
}

function hasControlByte(value: string): boolean {
  for (const character of value) {
    const code = character.charCodeAt(0);
    if (code < 0x20 || code === 0x7f) {
      return true;
    }
  }
  return false;
}

function schemeAndRest(rawURL: string): readonly [string, string] {
  for (let index = 0; index < rawURL.length; index += 1) {
    const code = rawURL.charCodeAt(index);
    const alphabetic = (code >= 0x41 && code <= 0x5a) || (code >= 0x61 && code <= 0x7a);
    if (alphabetic) {
      continue;
    }
    const accepted = (code >= 0x30 && code <= 0x39) || code === 0x2b || code === 0x2d || code === 0x2e;
    if (accepted) {
      if (index === 0) {
        return ["", rawURL];
      }
      continue;
    }
    if (code === 0x3a) {
      if (index === 0) {
        throw new URIError("missing protocol scheme");
      }
      return [rawURL.slice(0, index), rawURL.slice(index + 1)];
    }
    return ["", rawURL];
  }
  return ["", rawURL];
}

function validOptionalPort(port: string): boolean {
  if (port === "") {
    return true;
  }
  if (!port.startsWith(":")) {
    return false;
  }
  for (const character of port.slice(1)) {
    if (character < "0" || character > "9") {
      return false;
    }
  }
  return true;
}

function validIPv4(source: string): boolean {
  const parts = source.split(".");
  return (
    parts.length === 4 &&
    parts.every((part): boolean => {
      if (!/^[0-9]+$/.test(part)) {
        return false;
      }
      const value = Number(part);
      return value >= 0 && value <= 255 && String(value) === part;
    })
  );
}

function ipv6GroupCount(source: string): number | undefined {
  if (source === "") {
    return 0;
  }
  const groups = source.split(":");
  let count = 0;
  for (let index = 0; index < groups.length; index += 1) {
    const group = groups[index] ?? "";
    if (group.includes(".")) {
      if (index !== groups.length - 1 || !validIPv4(group)) {
        return undefined;
      }
      count += 2;
    } else {
      if (!/^[0-9A-Fa-f]{1,4}$/.test(group)) {
        return undefined;
      }
      count += 1;
    }
  }
  return count;
}

function validIPv6(source: string): boolean {
  const compression = source.indexOf("::");
  if (compression < 0) {
    return ipv6GroupCount(source) === 8;
  }
  if (source.indexOf("::", compression + 2) >= 0) {
    return false;
  }
  const left = ipv6GroupCount(source.slice(0, compression));
  const right = ipv6GroupCount(source.slice(compression + 2));
  return left !== undefined && right !== undefined && left + right < 8;
}

function parseHost(scheme: string, source: string): string {
  const openBracket = source.lastIndexOf("[");
  if (openBracket > 0) {
    throw new URIError("invalid IP-literal");
  }
  if (openBracket === 0) {
    const closeBracket = source.lastIndexOf("]");
    if (closeBracket < 0) {
      throw new URIError("missing ']' in host");
    }
    const port = source.slice(closeBracket + 1);
    if (!validOptionalPort(port)) {
      throw new URIError(`invalid port ${JSON.stringify(port)} after host`);
    }
    const addressAndZone = source.slice(1, closeBracket);
    const zone = addressAndZone.indexOf("%25");
    const address = decode(zone < 0 ? addressAndZone : addressAndZone.slice(0, zone), EscapeMode.Host);
    const decodedZone = zone < 0 ? "" : decode(addressAndZone.slice(zone), EscapeMode.Zone);
    if (!validIPv6(address)) {
      throw new URIError("invalid IP-literal");
    }
    return `[${address}${decodedZone}]${decode(port, EscapeMode.Host)}`;
  }

  const firstColon = source.indexOf(":");
  if (firstColon >= 0) {
    const lastColon = source.lastIndexOf(":");
    const portColon = scheme === "http" || scheme === "https" ? firstColon : lastColon;
    if (!validOptionalPort(source.slice(portColon))) {
      throw new URIError(`invalid port ${JSON.stringify(source.slice(portColon))} after host`);
    }
  }
  return decode(source, EscapeMode.Host);
}

function validUserinfo(source: string): boolean {
  for (let index = 0; index < source.length; index += 1) {
    const character = source[index] ?? "";
    const code = character.charCodeAt(0);
    const alphaNumeric =
      (code >= 0x41 && code <= 0x5a) ||
      (code >= 0x61 && code <= 0x7a) ||
      (code >= 0x30 && code <= 0x39);
    if (
      alphaNumeric ||
      "-._~!$&'()*+,;=:@".includes(character) ||
      (character === "%" &&
        index + 2 < source.length &&
        /^[0-9A-Fa-f]{2}$/.test(source.slice(index + 1, index + 3)))
    ) {
      if (character === "%") {
        index += 2;
      }
      continue;
    }
    return false;
  }
  return true;
}

function parseAuthority(scheme: string, authority: string): readonly [ParsedUserinfo | undefined, string] {
  const separator = authority.lastIndexOf("@");
  const host = parseHost(scheme, separator < 0 ? authority : authority.slice(separator + 1));
  if (separator < 0) {
    return [undefined, host];
  }
  const encodedUserinfo = authority.slice(0, separator);
  if (!validUserinfo(encodedUserinfo)) {
    throw new URIError("net/url: invalid userinfo");
  }
  const colon = encodedUserinfo.indexOf(":");
  if (colon < 0) {
    return [{
      username: decode(encodedUserinfo, EscapeMode.UserPassword),
      password: "",
      passwordPresent: false,
    }, host];
  }
  return [
    {
      username: decode(encodedUserinfo.slice(0, colon), EscapeMode.UserPassword),
      password: decode(encodedUserinfo.slice(colon + 1), EscapeMode.UserPassword),
      passwordPresent: true,
    },
    host,
  ];
}

function setPath(result: ParsedURL, encodedPath: string): void {
  result.Path = decode(encodedPath, EscapeMode.Path);
  result.RawPath = escapePath(result.Path) === encodedPath ? "" : encodedPath;
}

function setFragment(result: ParsedURL, encodedFragment: string): void {
  result.Fragment = decode(encodedFragment, EscapeMode.Fragment);
  result.RawFragment = escapeFragment(result.Fragment) === encodedFragment ? "" : encodedFragment;
}

function parseWithoutFragment(rawURL: string): ParsedURL {
  if (hasControlByte(rawURL)) {
    throw new URIError("net/url: invalid control character in URL");
  }
  if (rawURL === "*") {
    const result = emptyURL();
    result.Path = "*";
    return result;
  }

  const [originalScheme, originalRest] = schemeAndRest(rawURL);
  const result = emptyURL();
  result.Scheme = originalScheme.toLowerCase();
  let rest = originalRest;
  if (rest.endsWith("?") && rest.indexOf("?") === rest.length - 1) {
    result.ForceQuery = true;
    rest = rest.slice(0, -1);
  } else {
    const query = rest.indexOf("?");
    if (query >= 0) {
      result.RawQuery = rest.slice(query + 1);
      rest = rest.slice(0, query);
    }
  }

  if (!rest.startsWith("/")) {
    if (result.Scheme !== "") {
      result.Opaque = rest;
      return result;
    }
    const firstSegment = rest.split("/", 1)[0] ?? "";
    if (firstSegment.includes(":")) {
      throw new URIError("first path segment in URL cannot contain colon");
    }
  }

  if ((result.Scheme !== "" || !rest.startsWith("///")) && rest.startsWith("//")) {
    const slash = rest.indexOf("/", 2);
    const authority = slash < 0 ? rest.slice(2) : rest.slice(2, slash);
    rest = slash < 0 ? "" : rest.slice(slash);
    [result.User, result.Host] = parseAuthority(result.Scheme, authority);
  } else if (result.Scheme !== "" && rest.startsWith("/")) {
    result.OmitHost = true;
  }

  setPath(result, rest);
  return result;
}

export function parseURL(rawURL: gostring): ParsedURL {
  const fragment = rawURL.indexOf("#");
  const withoutFragment = fragment < 0 ? rawURL : rawURL.slice(0, fragment);
  const result = parseWithoutFragment(withoutFragment);
  if (fragment >= 0) {
    setFragment(result, rawURL.slice(fragment + 1));
  }
  if (result.RawPath !== "" && !decodedPathMatches(result.RawPath, result.Path)) {
    result.RawPath = "";
  }
  return result;
}
