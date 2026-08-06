import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type {
  int64,
  uint8,
  uint16,
  uint32,
  uint64,
} from "@gotots/gostdlib/internal/scalars.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import { GoPanic } from "@gotots/runtime/panic.js";

import * as reflect from "../../../../reflect.js";
import { ProviderError } from "../../../runtime/error.js";
import {
  Float32bits,
  Float32frombits,
  Float64bits,
  Float64frombits,
} from "../../math/scalars.js";

// CodecByteOrder is the structural byte-order contract the codec walks
// with; the public binary.ByteOrder satisfies it without a module cycle,
// and the canonical profile order satisfies the awaitable variant.
export interface CodecByteOrder {
  PutUint16(buffer: RuntimeSlice<uint8>, value: uint16): void;
  PutUint32(buffer: RuntimeSlice<uint8>, value: uint32): void;
  PutUint64(buffer: RuntimeSlice<uint8>, value: uint64): void;
  Uint16(buffer: RuntimeSlice<uint8>): uint16;
  Uint32(buffer: RuntimeSlice<uint8>): uint32;
  Uint64(buffer: RuntimeSlice<uint8>): uint64;
}

export interface AwaitableCodecByteOrder {
  PutUint16(
    buffer: RuntimeSlice<uint8>,
    value: uint16,
  ): void | Promise<void>;
  PutUint32(
    buffer: RuntimeSlice<uint8>,
    value: uint32,
  ): void | Promise<void>;
  PutUint64(
    buffer: RuntimeSlice<uint8>,
    value: uint64,
  ): void | Promise<void>;
  Uint16(buffer: RuntimeSlice<uint8>): uint16 | Promise<uint16>;
  Uint32(buffer: RuntimeSlice<uint8>): uint32 | Promise<uint32>;
  Uint64(buffer: RuntimeSlice<uint8>): uint64 | Promise<uint64>;
}

// encodedSize is the exact Go binary encoding size of one reflected
// value: fixed-size scalar kinds, structs of them, and slices of them.
// Every other kind — including platform-sized int, uint, and uintptr —
// is invalid exactly as in Go, reported as -1.
export function encodedSize(value: reflect.Value): int64 {
  const kind = value.Kind().value;
  const width = scalarWidth(kind, value);
  if (width >= 0) {
    return globalThis.BigInt(width);
  }
  if (kind === reflect.Struct.value) {
    let total = 0n;
    const count = value.NumField();
    for (let index = 0n; index < count; index++) {
      const field = encodedSize(value.Field(index));
      if (field < 0n) {
        return -1n;
      }
      total += field;
    }
    return total;
  }
  if (kind === reflect.Slice.value) {
    let total = 0n;
    const length = value.Len();
    for (let index = 0n; index < length; index++) {
      const element = encodedSize(value.Index(index));
      if (element < 0n) {
        return -1n;
      }
      total += element;
    }
    return total;
  }
  return -1n;
}

// decodeInto writes decoded bytes through the settable reflected value
// and returns the next offset.
export function decodeInto(
  order: CodecByteOrder,
  buffer: RuntimeSlice<uint8>,
  offset: number,
  value: reflect.Value,
): number {
  const kind = value.Kind().value;
  const width = scalarWidth(kind, value);
  if (width >= 0) {
    decodeScalar(order, buffer, offset, value, kind, width);
    return offset + width;
  }
  if (kind === reflect.Struct.value) {
    const count = value.NumField();
    for (let index = 0n; index < count; index++) {
      offset = decodeInto(order, buffer, offset, value.Field(index));
    }
    return offset;
  }
  if (kind === reflect.Slice.value) {
    const length = value.Len();
    for (let index = 0n; index < length; index++) {
      offset = decodeInto(order, buffer, offset, value.Index(index));
    }
    return offset;
  }
  return codecKindPanic(value);
}

// encodeFrom reads the reflected value into the buffer and returns the
// next offset.
export function encodeFrom(
  order: CodecByteOrder,
  buffer: RuntimeSlice<uint8>,
  offset: number,
  value: reflect.Value,
): number {
  const kind = value.Kind().value;
  const width = scalarWidth(kind, value);
  if (width >= 0) {
    encodeScalar(order, buffer, offset, value, kind, width);
    return offset + width;
  }
  if (kind === reflect.Struct.value) {
    const count = value.NumField();
    for (let index = 0n; index < count; index++) {
      offset = encodeFrom(order, buffer, offset, value.Field(index));
    }
    return offset;
  }
  if (kind === reflect.Slice.value) {
    const length = value.Len();
    for (let index = 0n; index < length; index++) {
      offset = encodeFrom(order, buffer, offset, value.Index(index));
    }
    return offset;
  }
  return codecKindPanic(value);
}

function scalarWidth(kind: uint64, value: reflect.Value): number {
  switch (kind) {
    case reflect.Bool.value:
    case reflect.Int8.value:
    case reflect.Uint8.value:
      return 1;
    case reflect.Int16.value:
    case reflect.Uint16.value:
      return 2;
    case reflect.Int32.value:
    case reflect.Uint32.value:
    case reflect.Float32.value:
      return 4;
    case reflect.Int64.value:
    case reflect.Uint64.value:
    case reflect.Float64.value:
      return 8;
    default:
      return -1;
  }
}

function segment(
  buffer: RuntimeSlice<uint8>,
  offset: number,
  width: number,
): RuntimeSlice<uint8> {
  return buffer.slice(offset, offset + width, null);
}

function decodeScalar(
  order: CodecByteOrder,
  buffer: RuntimeSlice<uint8>,
  offset: number,
  value: reflect.Value,
  kind: uint64,
  width: number,
): void {
  if (kind === reflect.Bool.value) {
    value.SetBool(buffer.get(offset) !== 0);
    return;
  }
  if (kind === reflect.Float32.value) {
    value.SetFloat(Float32frombits(order.Uint32(segment(buffer, offset, 4))));
    return;
  }
  if (kind === reflect.Float64.value) {
    value.SetFloat(Float64frombits(order.Uint64(segment(buffer, offset, 8))));
    return;
  }
  let raw: uint64;
  switch (width) {
    case 1:
      raw = globalThis.BigInt(buffer.get(offset));
      break;
    case 2:
      raw = globalThis.BigInt(order.Uint16(segment(buffer, offset, 2)));
      break;
    case 4:
      raw = globalThis.BigInt(order.Uint32(segment(buffer, offset, 4)));
      break;
    default:
      raw = order.Uint64(segment(buffer, offset, 8));
      break;
  }
  if (
    kind === reflect.Int8.value ||
    kind === reflect.Int16.value ||
    kind === reflect.Int32.value ||
    kind === reflect.Int64.value
  ) {
    value.SetInt(globalThis.BigInt.asIntN(width * 8, raw));
    return;
  }
  value.SetUint(raw);
}

function encodeScalar(
  order: CodecByteOrder,
  buffer: RuntimeSlice<uint8>,
  offset: number,
  value: reflect.Value,
  kind: uint64,
  width: number,
): void {
  if (kind === reflect.Bool.value) {
    buffer.set(offset, value.Bool() ? 1 : 0);
    return;
  }
  if (kind === reflect.Float32.value) {
    order.PutUint32(segment(buffer, offset, 4), Float32bits(value.Float()));
    return;
  }
  if (kind === reflect.Float64.value) {
    order.PutUint64(segment(buffer, offset, 8), Float64bits(value.Float()));
    return;
  }
  const signed =
    kind === reflect.Int8.value ||
    kind === reflect.Int16.value ||
    kind === reflect.Int32.value ||
    kind === reflect.Int64.value;
  const raw = globalThis.BigInt.asUintN(
    width * 8,
    signed ? value.Int() : value.Uint(),
  );
  switch (width) {
    case 1:
      buffer.set(offset, globalThis.Number(raw));
      break;
    case 2:
      order.PutUint16(segment(buffer, offset, 2), globalThis.Number(raw));
      break;
    case 4:
      order.PutUint32(segment(buffer, offset, 4), globalThis.Number(raw));
      break;
    default:
      order.PutUint64(segment(buffer, offset, 8), raw);
      break;
  }
}

function codecKindPanic(value: reflect.Value): never {
  return GoPanic.raise(
    new ProviderError(
      `binary: unsupported codec kind ${value.Kind().String()}`,
    ),
  );
}

// decodeIntoAwaited mirrors decodeInto over an awaitable byte order, as
// the canonical profile boundary supplies.
export async function decodeIntoAwaited(
  order: AwaitableCodecByteOrder,
  buffer: RuntimeSlice<uint8>,
  offset: number,
  value: reflect.Value,
): Promise<number> {
  const kind = value.Kind().value;
  const width = scalarWidth(kind, value);
  if (width >= 0) {
    await decodeScalarAwaited(order, buffer, offset, value, kind, width);
    return offset + width;
  }
  if (kind === reflect.Struct.value) {
    const count = value.NumField();
    for (let index = 0n; index < count; index++) {
      offset = await decodeIntoAwaited(
        order,
        buffer,
        offset,
        value.Field(index),
      );
    }
    return offset;
  }
  if (kind === reflect.Slice.value) {
    const length = value.Len();
    for (let index = 0n; index < length; index++) {
      offset = await decodeIntoAwaited(
        order,
        buffer,
        offset,
        value.Index(index),
      );
    }
    return offset;
  }
  return codecKindPanic(value);
}

// encodeFromAwaited mirrors encodeFrom over an awaitable byte order.
export async function encodeFromAwaited(
  order: AwaitableCodecByteOrder,
  buffer: RuntimeSlice<uint8>,
  offset: number,
  value: reflect.Value,
): Promise<number> {
  const kind = value.Kind().value;
  const width = scalarWidth(kind, value);
  if (width >= 0) {
    await encodeScalarAwaited(order, buffer, offset, value, kind, width);
    return offset + width;
  }
  if (kind === reflect.Struct.value) {
    const count = value.NumField();
    for (let index = 0n; index < count; index++) {
      offset = await encodeFromAwaited(
        order,
        buffer,
        offset,
        value.Field(index),
      );
    }
    return offset;
  }
  if (kind === reflect.Slice.value) {
    const length = value.Len();
    for (let index = 0n; index < length; index++) {
      offset = await encodeFromAwaited(
        order,
        buffer,
        offset,
        value.Index(index),
      );
    }
    return offset;
  }
  return codecKindPanic(value);
}

async function decodeScalarAwaited(
  order: AwaitableCodecByteOrder,
  buffer: RuntimeSlice<uint8>,
  offset: number,
  value: reflect.Value,
  kind: uint64,
  width: number,
): Promise<void> {
  if (kind === reflect.Bool.value) {
    value.SetBool(buffer.get(offset) !== 0);
    return;
  }
  if (kind === reflect.Float32.value) {
    value.SetFloat(
      Float32frombits(await order.Uint32(segment(buffer, offset, 4))),
    );
    return;
  }
  if (kind === reflect.Float64.value) {
    value.SetFloat(
      Float64frombits(await order.Uint64(segment(buffer, offset, 8))),
    );
    return;
  }
  let raw: uint64;
  switch (width) {
    case 1:
      raw = globalThis.BigInt(buffer.get(offset));
      break;
    case 2:
      raw = globalThis.BigInt(
        await order.Uint16(segment(buffer, offset, 2)),
      );
      break;
    case 4:
      raw = globalThis.BigInt(
        await order.Uint32(segment(buffer, offset, 4)),
      );
      break;
    default:
      raw = await order.Uint64(segment(buffer, offset, 8));
      break;
  }
  if (
    kind === reflect.Int8.value ||
    kind === reflect.Int16.value ||
    kind === reflect.Int32.value ||
    kind === reflect.Int64.value
  ) {
    value.SetInt(globalThis.BigInt.asIntN(width * 8, raw));
    return;
  }
  value.SetUint(raw);
}

async function encodeScalarAwaited(
  order: AwaitableCodecByteOrder,
  buffer: RuntimeSlice<uint8>,
  offset: number,
  value: reflect.Value,
  kind: uint64,
  width: number,
): Promise<void> {
  if (kind === reflect.Bool.value) {
    buffer.set(offset, value.Bool() ? 1 : 0);
    return;
  }
  if (kind === reflect.Float32.value) {
    await order.PutUint32(
      segment(buffer, offset, 4),
      Float32bits(value.Float()),
    );
    return;
  }
  if (kind === reflect.Float64.value) {
    await order.PutUint64(
      segment(buffer, offset, 8),
      Float64bits(value.Float()),
    );
    return;
  }
  const signed =
    kind === reflect.Int8.value ||
    kind === reflect.Int16.value ||
    kind === reflect.Int32.value ||
    kind === reflect.Int64.value;
  const raw = globalThis.BigInt.asUintN(
    width * 8,
    signed ? value.Int() : value.Uint(),
  );
  switch (width) {
    case 1:
      buffer.set(offset, globalThis.Number(raw));
      break;
    case 2:
      await order.PutUint16(
        segment(buffer, offset, 2),
        globalThis.Number(raw),
      );
      break;
    case 4:
      await order.PutUint32(
        segment(buffer, offset, 4),
        globalThis.Number(raw),
      );
      break;
    default:
      await order.PutUint64(segment(buffer, offset, 8), raw);
      break;
  }
}
