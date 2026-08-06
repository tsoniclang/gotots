import assert from "node:assert/strict";
import test from "node:test";

import {
  Parse,
  PathEscape,
  QueryEscape,
  URL,
} from "../src/net/url.js";

test("net/url escaping follows Go component rules", () => {
  assert.equal(PathEscape("a/b;c,d?e"), "a%2Fb%3Bc%2Cd%3Fe");
  assert.equal(PathEscape("café"), "caf%C3%A9");
  assert.equal(PathEscape("😀"), "%F0%9F%98%80");
  assert.equal(QueryEscape("a b+c&d"), "a+b%2Bc%26d");
});

test("net/url Parse preserves decoded and encoded fields", () => {
  const [parsed, error] = Parse("https://user:pass@example.com/a%2fb?q=1#x%2fy");
  assert.equal(error, undefined);
  assert.ok(parsed instanceof URL);
  assert.equal(parsed.Scheme, "https");
  assert.equal(parsed.Host, "example.com");
  assert.equal(parsed.Path, "/a/b");
  assert.equal(parsed.RawPath, "/a%2fb");
  assert.equal(parsed.RawQuery, "q=1");
  assert.equal(parsed.Fragment, "x/y");
  assert.equal(parsed.RawFragment, "x%2fy");
  assert.notEqual(parsed.User, undefined);
});

test("net/url Parse preserves opaque and force-query forms", () => {
  const [opaque, opaqueError] = Parse("mailto:user@example.com");
  assert.equal(opaqueError, undefined);
  assert.equal(opaque?.Scheme, "mailto");
  assert.equal(opaque?.Opaque, "user@example.com");

  const [forced, forcedError] = Parse("/search?");
  assert.equal(forcedError, undefined);
  assert.equal(forced?.Path, "/search");
  assert.equal(forced?.ForceQuery, true);
});

test("net/url Parse validates bracketed IPv6 hosts", () => {
  const [parsed, error] = Parse("tcp://[2001:db8::1]:443/path");
  assert.equal(error, undefined);
  assert.equal(parsed?.Host, "[2001:db8::1]:443");

  const [invalid, invalidError] = Parse("tcp://[not-ipv6]:443/path");
  assert.equal(invalid, undefined);
  assert.equal(invalidError?.Error().includes("invalid IP-literal"), true);
});

test("net/url Parse reports malformed escapes as Go errors", () => {
  const [parsed, error] = Parse("https://example.com/%zz");
  assert.equal(parsed, undefined);
  assert.equal(error?.Error().includes("invalid URL escape"), true);
});
