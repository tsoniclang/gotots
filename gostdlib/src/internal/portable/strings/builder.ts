import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  GoError,
} from "@gotots/runtime/interface-value.js";
import type {
  gostring,
  int32,
  int64,
  uint8,
} from "@gotots/runtime/scalars.js";

import { sliceValues } from "../../runtime/slice.js";
import { encodeRune } from "../utf8/codec.js";

export class Builder {
  #value: gostring;

  constructor() {
    this.#value = "";
  }

  static Grow(receiver: Builder | undefined, count: int64): void {
    const builder = requireBuilder(receiver);
    if (!Number.isSafeInteger(count) || count < 0) {
      GoPanic.raiseRuntime("strings.Builder.Grow: negative count");
    }
    void builder;
  }

  static Len(receiver: Builder | undefined): int64 {
    return requireBuilder(receiver).#value.length;
  }

  static Reset(receiver: Builder | undefined): void {
    requireBuilder(receiver).#value = "";
  }

  static String(receiver: Builder | undefined): gostring {
    return requireBuilder(receiver).#value;
  }

  static Write(
    receiver: Builder | undefined,
    source: RuntimeSlice<uint8>,
  ): [int64, GoError | undefined] {
    const builder = requireBuilder(receiver);
    const bytes = sliceValues(source);
    let appended = "";
    for (const byte of bytes) {
      appended += String.fromCharCode(byte);
    }
    builder.#value += appended;
    return [bytes.length, undefined];
  }

  static WriteByte(receiver: Builder | undefined, value: uint8): GoError | undefined {
    requireBuilder(receiver).#value += String.fromCharCode(value);
    return undefined;
  }

  static WriteRune(
    receiver: Builder | undefined,
    rune: int32,
  ): [int64, GoError | undefined] {
    const encoded = encodeRune(rune);
    requireBuilder(receiver).#value += encoded;
    return [encoded.length, undefined];
  }

  static WriteString(
    receiver: Builder | undefined,
    text: gostring,
  ): [int64, GoError | undefined] {
    requireBuilder(receiver).#value += text;
    return [text.length, undefined];
  }
}

function requireBuilder(receiver: Builder | undefined): Builder {
  if (receiver === undefined) {
    GoPanic.raiseRuntime("nil *strings.Builder");
  }
  return receiver;
}
