import type { GoError } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { int, uint8 } from "@gotots/gostdlib/internal/scalars.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";

import { hostInteger, integerFromHost } from "../../host-integer.js";

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
  ): [int, GoError | undefined] {
    const buffer = requireBuffer(receiver);
    if (buffer.#offset >= buffer.#source.length) {
      buffer.#source = buffer.#source.slice(0, 0, null);
      buffer.#offset = 0;
      return target.length === 0 ? [0n, undefined] : [0n, ioState.EOF];
    }

    const unread = buffer.#source.slice(
      buffer.#offset,
      buffer.#source.length,
      null,
    );
    const count = RuntimeSlice.copy(target, unread);
    buffer.#offset += count;
    return [integerFromHost(count), undefined];
  }

  static Available(receiver: Buffer | undefined): int {
    const buffer = requireBuffer(receiver);
    return integerFromHost(buffer.#source.capacity - buffer.#source.length);
  }

  static AvailableBuffer(receiver: Buffer | undefined): RuntimeSlice<uint8> {
    const buffer = requireBuffer(receiver);
    return buffer.#source.slice(buffer.#source.length, buffer.#source.length, null);
  }

  static Grow(receiver: Buffer | undefined, count: int): void {
    const buffer = requireBuffer(receiver);
    if (count < 0n) {
      GoPanic.raiseRuntime("bytes.Buffer.Grow: negative count");
    }
    const hostCount = hostInteger(count);
    const unread = buffer.#source.length - buffer.#offset;
    if (buffer.#source.capacity - buffer.#source.length >= hostCount) {
      return;
    }
    const target = RuntimeSlice.make<uint8>(unread, unread + hostCount, 0);
    RuntimeSlice.copy(
      target,
      buffer.#source.slice(buffer.#offset, buffer.#source.length, null),
    );
    buffer.#source = target;
    buffer.#offset = 0;
  }

  static Len(receiver: Buffer | undefined): int {
    const buffer = requireBuffer(receiver);
    return integerFromHost(buffer.#source.length - buffer.#offset);
  }

  static Next(receiver: Buffer | undefined, count: int): RuntimeSlice<uint8> {
    const buffer = requireBuffer(receiver);
    const selected = Math.max(
      0,
      Math.min(hostInteger(count), hostInteger(Buffer.Len(buffer))),
    );
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
  ): [int, GoError | undefined] {
    const buffer = requireBuffer(receiver);
    if (source.length === 0) {
      return [0n, undefined];
    }
    Buffer.Grow(buffer, integerFromHost(source.length));
    const values: uint8[] = [];
    for (let index = 0; index < source.length; index += 1) {
      values.push(source.get(index));
    }
    buffer.#source = buffer.#source.append(0, values);
    return [integerFromHost(source.length), undefined];
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
