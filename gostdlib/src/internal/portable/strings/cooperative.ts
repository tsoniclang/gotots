import { GoPanic } from "@gotots/runtime/panic.js";
import type { Awaitable, bool, gostring, int32, int64 } from "@gotots/runtime/scalars.js";

import { decodeRuneAt, encodeRune } from "../utf8/codec.js";

type Predicate = ((rune: int32) => Awaitable<bool>) | undefined;

export async function ContainsFunc(
  text: gostring,
  predicate: Predicate,
): Promise<bool> {
  return await IndexFunc(text, predicate) >= 0;
}

export async function IndexFunc(
  text: gostring,
  predicate: Predicate,
): Promise<int64> {
  return findByPredicate(text, predicate, true, false);
}

export async function LastIndexFunc(
  text: gostring,
  predicate: Predicate,
): Promise<int64> {
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
    index += Math.max(1, width);
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
  return index < 0 ? "" : text.slice(index);
}

export async function TrimRightFunc(
  text: gostring,
  predicate: Predicate,
): Promise<gostring> {
  const index = await findByPredicate(text, predicate, false, true);
  if (index < 0) {
    return "";
  }
  const [, width] = decodeRuneAt(text, index);
  return text.slice(0, index + Math.max(1, width));
}

async function findByPredicate(
  text: gostring,
  predicate: Predicate,
  expected: boolean,
  last: boolean,
): Promise<int64> {
  if (predicate === undefined && text.length > 0) {
    GoPanic.raiseRuntime("call of nil predicate function");
  }
  let found = -1;
  for (let index = 0; index < text.length; ) {
    const [rune, width] = decodeRuneAt(text, index);
    if (predicate !== undefined && await predicate(rune) === expected) {
      found = index;
      if (!last) {
        return found;
      }
    }
    index += Math.max(1, width);
  }
  return found;
}
