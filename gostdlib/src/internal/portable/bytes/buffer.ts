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

  static Available(receiver: Buffer | undefined): int64 {
    const buffer = requireBuffer(receiver);
    return buffer.#source.capacity - buffer.#source.length;
  }

  static AvailableBuffer(receiver: Buffer | undefined): RuntimeSlice<uint8> {
    const buffer = requireBuffer(receiver);
    return buffer.#source.slice(buffer.#source.length, buffer.#source.length, null);
  }

  static Grow(receiver: Buffer | undefined, count: int64): void {
    const buffer = requireBuffer(receiver);
    if (!Number.isSafeInteger(count) || count < 0) {
      GoPanic.raiseRuntime("bytes.Buffer.Grow: negative count");
    }
    const unread = buffer.#source.length - buffer.#offset;
    if (buffer.#source.capacity - buffer.#source.length >= count) {
      return;
    }
    const target = RuntimeSlice.make<uint8>(unread, unread + count, 0);
    RuntimeSlice.copy(
      target,
      buffer.#source.slice(buffer.#offset, buffer.#source.length, null),
    );
    buffer.#source = target;
    buffer.#offset = 0;
  }

  static Len(receiver: Buffer | undefined): int64 {
    const buffer = requireBuffer(receiver);
    return buffer.#source.length - buffer.#offset;
  }

  static Next(receiver: Buffer | undefined, count: int64): RuntimeSlice<uint8> {
    const buffer = requireBuffer(receiver);
    const selected = Math.max(0, Math.min(Math.trunc(count), Buffer.Len(buffer)));
    const result = buffer.#source.slice(
      buffer.#offset,
      buffer.#offset + selected,
      null,
    );
    buffer.#offset += selected;
    return result;
  }

  static Write(
    receiver: Buffer | undefined,
    source: RuntimeSlice<uint8>,
  ): [int64, GoError | undefined] {
    const buffer = requireBuffer(receiver);
    if (source.length === 0) {
      return [0, undefined];
    }
    Buffer.Grow(buffer, source.length);
    const values: uint8[] = [];
    for (let index = 0; index < source.length; index += 1) {
      values.push(source.get(index));
    }
    buffer.#source = buffer.#source.append(0, values);
    return [source.length, undefined];
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
