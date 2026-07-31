import type { GoError } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { int64, uint8 } from "@gotots/runtime/scalars.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import { state as ioState } from "../../../io.js";

let createBuffer: (source: RuntimeSlice<uint8>) => Buffer;

export class Buffer {
  #source = RuntimeSlice.nil<uint8>();
  #offset = 0;

  static {
    createBuffer = (source: RuntimeSlice<uint8>): Buffer => {
      const buffer = new Buffer();
      buffer.#source = source;
      return buffer;
    };
  }

  static Read(
    receiver: Buffer | undefined,
    target: RuntimeSlice<uint8>,
  ): [int64, GoError | undefined] {
    const buffer = requireBuffer(receiver);
    if (buffer.#offset >= buffer.#source.length) {
      buffer.#source = buffer.#source.slice(0, 0, null);
      buffer.#offset = 0;
      return target.length === 0 ? [0, undefined] : [0, ioState.EOF];
    }

    const unread = buffer.#source.slice(
      buffer.#offset,
      buffer.#source.length,
      null,
    );
    const count = RuntimeSlice.copy(target, unread);
    buffer.#offset += count;
    return [count, undefined];
  }
}

export function NewBuffer(source: RuntimeSlice<uint8>): Buffer {
  return createBuffer(source);
}

function requireBuffer(receiver: Buffer | undefined): Buffer {
  if (receiver === undefined) {
    GoPanic.raiseRuntime("nil *bytes.Buffer");
  }
  return receiver;
}
