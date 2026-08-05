import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  GoError,
} from "@gotots/runtime/interface-value.js";
import type {
  gostring,
  int,
  int32,
  uint8,
} from "@gotots/gostdlib/internal/scalars.js";
import {
  hostInteger,
  integerFromHost,
} from "../../host-integer.js";

import { sliceValues } from "../../runtime/slice.js";
import { encodeRune } from "../utf8/codec.js";

let assignBuilderRepresentation: (target: Builder, source: Builder) => void;
let copyBuilderRepresentation: (source: Builder) => Builder;

export class Builder {
  #value: gostring;
  #owner: Builder | undefined;

  constructor() {
    this.#value = "";
    this.#owner = undefined;
  }

  static {
    assignBuilderRepresentation = (target: Builder, source: Builder): void => {
      const value = source.#value;
      const owner = source.#owner;
      target.#value = value;
      target.#owner = owner;
    };
    copyBuilderRepresentation = (source: Builder): Builder => {
      const result = new Builder();
      result.#value = source.#value;
      result.#owner = source.#owner;
      return result;
    };
  }

  static Grow(receiver: Builder | undefined, count: int): void {
    const builder = requireBuilder(receiver);
    builder.#copyCheck();
    if (count < 0n) {
      GoPanic.raiseRuntime("strings.Builder.Grow: negative count");
    }
    hostInteger(count);
    void builder;
  }

  static Len(receiver: Builder | undefined): int {
    return integerFromHost(requireBuilder(receiver).#value.length);
  }

  static Reset(receiver: Builder | undefined): void {
    const builder = requireBuilder(receiver);
    builder.#value = "";
    builder.#owner = undefined;
  }

  static String(receiver: Builder | undefined): gostring {
    return requireBuilder(receiver).#value;
  }

  static Write(
    receiver: Builder | undefined,
    source: RuntimeSlice<uint8>,
  ): [int, GoError | undefined] {
    const builder = requireBuilder(receiver);
    builder.#copyCheck();
    const bytes = sliceValues(source);
    let appended = "";
    for (const byte of bytes) {
      appended += String.fromCharCode(byte);
    }
    builder.#value += appended;
    return [integerFromHost(bytes.length), undefined];
  }

  static WriteByte(receiver: Builder | undefined, value: uint8): GoError | undefined {
    const builder = requireBuilder(receiver);
    builder.#copyCheck();
    builder.#value += String.fromCharCode(value);
    return undefined;
  }

  static WriteRune(
    receiver: Builder | undefined,
    rune: int32,
  ): [int, GoError | undefined] {
    const encoded = encodeRune(rune);
    const builder = requireBuilder(receiver);
    builder.#copyCheck();
    builder.#value += encoded;
    return [integerFromHost(encoded.length), undefined];
  }

  static WriteString(
    receiver: Builder | undefined,
    text: gostring,
  ): [int, GoError | undefined] {
    const builder = requireBuilder(receiver);
    builder.#copyCheck();
    builder.#value += text;
    return [integerFromHost(text.length), undefined];
  }

  #copyCheck(): void {
    if (this.#owner === undefined) {
      this.#owner = this;
      return;
    }
    if (this.#owner !== this) {
      GoPanic.raiseRuntime(
        "strings: illegal use of non-zero Builder copied by value",
      );
    }
  }
}

export function builderRepresentationAssign(
  target: Builder,
  source: Builder,
): void {
  assignBuilderRepresentation(target, source);
}

export function builderRepresentationCopy(source: Builder): Builder {
  return copyBuilderRepresentation(source);
}

function requireBuilder(receiver: Builder | undefined): Builder {
  if (receiver === undefined) {
    GoPanic.raiseRuntime("nil *strings.Builder");
  }
  return receiver;
}
