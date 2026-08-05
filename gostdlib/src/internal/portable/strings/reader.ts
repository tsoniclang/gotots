import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { GoError } from "@gotots/runtime/interface-value.js";
import type { gostring, int, uint8 } from "@gotots/gostdlib/internal/scalars.js";
import { integerFromHost } from "../../host-integer.js";

import { state as ioState } from "../../../io.js";

let createReader: (source: gostring) => Reader;

export class Reader {
  readonly #source: gostring;
  #offset = 0;

  private constructor(source: gostring) {
    this.#source = source;
  }

  static {
    createReader = (source: gostring): Reader => new Reader(source);
  }

  static Read(
    receiver: Reader | undefined,
    target: RuntimeSlice<uint8>,
  ): [int, GoError | undefined] {
    if (receiver === undefined) {
      GoPanic.raiseRuntime("nil *strings.Reader");
    }
    if (receiver.#offset >= receiver.#source.length) {
      return [0n, ioState.EOF];
    }
    const count = Math.min(target.length, receiver.#source.length - receiver.#offset);
    for (let index = 0; index < count; index += 1) {
      target.set(index, receiver.#source.charCodeAt(receiver.#offset + index));
    }
    receiver.#offset += count;
    return [integerFromHost(count), undefined];
  }
}

export function NewReader(text: gostring): Reader {
  return createReader(text);
}
