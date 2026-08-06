import { GoPanic } from "@gotots/runtime/panic.js";
import type { Awaitable, bool, gostring, int, int32 } from "@gotots/gostdlib/internal/scalars.js";

import { hostInteger, integerFromHost } from "../../host-integer.js";

import { decodeRuneAt, encodeRune } from "../utf8/codec.js";

type Predicate = ((rune: int32) => Awaitable<bool>) | undefined;

export async function ContainsFunc(
  text: gostring,
  predicate: Predicate,
): Promise<bool> {
  return await IndexFunc(text, predicate) >= 0n;
}

export async function IndexFunc(
  text: gostring,
  predicate: Predicate,
): Promise<int> {
  return findByPredicate(text, predicate, true, false);
}

export async function LastIndexFunc(
  text: gostring,
  predicate: Predicate,
): Promise<int> {
  return findByPredicate(text, predicate, true, true);
}

export async function Map(
  mapping: ((rune: int32) => Awaitable<int32>) | undefined,
  text: gostring,
): Promise<gostring> {
  if (mapping === undefined && text.length > 0) {
    GoPanic.raiseRuntime("call of nil mapping function");
  }
  let result = "";
  for (let index = 0; index < text.length; ) {
    const [rune, width] = decodeRuneAt(text, index);
    const mapped = mapping === undefined ? rune : await mapping(rune);
    if (mapped >= 0) {
      result += encodeRune(mapped);
    }
    index += Math.max(1, hostInteger(width));
  }
  return result;
}

export async function TrimFunc(
  text: gostring,
  predicate: Predicate,
): Promise<gostring> {
  return TrimRightFunc(await TrimLeftFunc(text, predicate), predicate);
}

export async function TrimLeftFunc(
  text: gostring,
  predicate: Predicate,
): Promise<gostring> {
  const index = await findByPredicate(text, predicate, false, false);
  return index < 0n ? "" : text.slice(hostInteger(index));
}

export async function TrimRightFunc(
  text: gostring,
  predicate: Predicate,
): Promise<gostring> {
  const index = await findByPredicate(text, predicate, false, true);
  if (index < 0n) {
    return "";
  }
  const hostIndex = hostInteger(index);
  const [, width] = decodeRuneAt(text, hostIndex);
  return text.slice(0, hostIndex + Math.max(1, hostInteger(width)));
}

async function findByPredicate(
  text: gostring,
  predicate: Predicate,
  expected: boolean,
  last: boolean,
): Promise<int> {
  if (predicate === undefined && text.length > 0) {
    GoPanic.raiseRuntime("call of nil predicate function");
  }
  let found = -1;
  for (let index = 0; index < text.length; ) {
    const [rune, width] = decodeRuneAt(text, index);
    if (predicate !== undefined && await predicate(rune) === expected) {
      found = index;
      if (!last) {
        return integerFromHost(found);
      }
    }
    index += Math.max(1, hostInteger(width));
  }
  return integerFromHost(found);
}
