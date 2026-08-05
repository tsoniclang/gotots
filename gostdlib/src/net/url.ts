import type { GoError } from "@gotots/runtime/interface-value.js";
import type { bool, gostring } from "@gotots/gostdlib/internal/scalars.js";

import { escapePathSegment, escapeQuery } from "../internal/portable/url/escape.js";
import {
  parseURL,
  type ParsedURL,
  type ParsedUserinfo,
} from "../internal/portable/url/parse.js";
import { ProviderError } from "../internal/runtime/error.js";

export abstract class Userinfo {
  protected constructor() {}
}

class ParsedUserinfoValue extends Userinfo {
  constructor(
    readonly username: gostring,
    readonly password: gostring,
    readonly passwordPresent: bool,
  ) {
    super();
  }
}

export class URL {
  Scheme: gostring = "";
  Opaque: gostring = "";
  User: Userinfo | undefined;
  Host: gostring = "";
  Path: gostring = "";
  Fragment: gostring = "";
  RawQuery: gostring = "";
  RawPath: gostring = "";
  RawFragment: gostring = "";
  ForceQuery: bool = false;
  OmitHost: bool = false;
}

function materializeUserinfo(source: ParsedUserinfo | undefined): Userinfo | undefined {
  return source === undefined
    ? undefined
    : new ParsedUserinfoValue(source.username, source.password, source.passwordPresent);
}

function materializeURL(source: ParsedURL): URL {
  const result = new URL();
  result.Scheme = source.Scheme;
  result.Opaque = source.Opaque;
  result.User = materializeUserinfo(source.User);
  result.Host = source.Host;
  result.Path = source.Path;
  result.Fragment = source.Fragment;
  result.RawQuery = source.RawQuery;
  result.RawPath = source.RawPath;
  result.RawFragment = source.RawFragment;
  result.ForceQuery = source.ForceQuery;
  result.OmitHost = source.OmitHost;
  return result;
}

export function Parse(rawURL: gostring): [URL | undefined, GoError | undefined] {
  try {
    return [materializeURL(parseURL(rawURL)), undefined];
  } catch (failure) {
    if (failure instanceof URIError) {
      return [undefined, new ProviderError(`parse ${JSON.stringify(rawURL)}: ${failure.message}`)];
    }
    throw failure;
  }
}

export function PathEscape(s: gostring): gostring {
  return escapePathSegment(s);
}

export function QueryEscape(s: gostring): gostring {
  return escapeQuery(s);
}
