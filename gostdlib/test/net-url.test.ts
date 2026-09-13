import { GoString } from "@gotots/runtime/string-value.js";
import assert from "node:assert/strict";
import test from "node:test";
import { fromHostString } from "../src/internal/portable/utf8/codec.js";

import {
  Parse,
  PathEscape,
  QueryEscape,
  URL,
} from "../src/net/url.js";

test("net/url escaping follows Go component rules", () => {
  assert.equal((PathEscape(GoString.fromText("a/b;c,d?e")))?.text(), "a%2Fb%3Bc%2Cd%3Fe");
  assert.equal(PathEscape(fromHostString("café")).text(), "caf%C3%A9");
  assert.equal(PathEscape(fromHostString("😀")).text(), "%F0%9F%98%80");
  assert.equal((QueryEscape(GoString.fromText("a b+c&d")))?.text(), "a+b%2Bc%26d");
});

test("net/url Parse preserves decoded and encoded fields", () => {
  const [parsed, error] = Parse(GoString.fromText("https://user:pass@example.com/a%2fb?q=1#x%2fy"));
  assert.equal(error, undefined);
  assert.ok(parsed instanceof URL);
  assert.equal((parsed.Scheme)?.text(), "https");
  assert.equal((parsed.Host)?.text(), "example.com");
  assert.equal((parsed.Path)?.text(), "/a/b");
  assert.equal((parsed.RawPath)?.text(), "/a%2fb");
  assert.equal((parsed.RawQuery)?.text(), "q=1");
  assert.equal((parsed.Fragment)?.text(), "x/y");
  assert.equal((parsed.RawFragment)?.text(), "x%2fy");
  assert.notEqual(parsed.User, undefined);
});

test("net/url Parse preserves opaque and force-query forms", () => {
  const [opaque, opaqueError] = Parse(GoString.fromText("mailto:user@example.com"));
  assert.equal(opaqueError, undefined);
  assert.equal((opaque?.Scheme)?.text(), "mailto");
  assert.equal((opaque?.Opaque)?.text(), "user@example.com");

  const [forced, forcedError] = Parse(GoString.fromText("/search?"));
  assert.equal(forcedError, undefined);
  assert.equal((forced?.Path)?.text(), "/search");
  assert.equal(forced?.ForceQuery, true);
});

test("net/url Parse validates bracketed IPv6 hosts", () => {
  const [parsed, error] = Parse(GoString.fromText("tcp://[2001:db8::1]:443/path"));
  assert.equal(error, undefined);
  assert.equal((parsed?.Host)?.text(), "[2001:db8::1]:443");

  const [invalid, invalidError] = Parse(GoString.fromText("tcp://[not-ipv6]:443/path"));
  assert.equal(invalid, undefined);
  assert.equal(invalidError?.Error().text().includes("invalid IP-literal"), true);
});

test("net/url Parse reports malformed escapes as Go errors", () => {
  const [parsed, error] = Parse(GoString.fromText("https://example.com/%zz"));
  assert.equal(parsed, undefined);
  assert.equal(error?.Error().text().includes("invalid URL escape"), true);
});
