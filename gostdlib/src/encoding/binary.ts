import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  gostring,
  uint16,
  uint32,
  uint64,
  uint8,
} from "@gotots/gostdlib/internal/scalars.js";

import { nativeEndian } from "../internal/node/encoding/binary/native.js";
import {
  BigEndianOrder,
  LittleEndianOrder,
} from "../internal/portable/encoding/binary/byte-order.js";
import {
  decodeInto,
  encodeFrom,
  encodedSize,
} from "../internal/portable/encoding/binary/reflection-codec.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice as RuntimeSliceValue } from "@gotots/runtime/slice.js";
import { New as newError } from "../errors.js";
import { ReadFull } from "../io.js";
import type { Reader, Writer } from "../io.js";
import * as reflect from "../reflect.js";

export interface ByteOrder extends GoInterfaceValue {
  PutUint16(buffer: RuntimeSlice<uint8>, value: uint16): void;
  PutUint32(buffer: RuntimeSlice<uint8>, value: uint32): void;
  PutUint64(buffer: RuntimeSlice<uint8>, value: uint64): void;
  String(): gostring;
  Uint16(buffer: RuntimeSlice<uint8>): uint16;
  Uint32(buffer: RuntimeSlice<uint8>): uint32;
  Uint64(buffer: RuntimeSlice<uint8>): uint64;
}

export interface AppendByteOrder extends GoInterfaceValue {
  AppendUint16(buffer: RuntimeSlice<uint8>, value: uint16): RuntimeSlice<uint8>;
  AppendUint32(buffer: RuntimeSlice<uint8>, value: uint32): RuntimeSlice<uint8>;
  AppendUint64(buffer: RuntimeSlice<uint8>, value: uint64): RuntimeSlice<uint8>;
  String(): gostring;
}

interface BinaryEndian extends ByteOrder, AppendByteOrder {
  GoString(): gostring;
}

const bigEndian: BinaryEndian = new BigEndianOrder();
const littleEndian: BinaryEndian = new LittleEndianOrder();

export const state: {
  BigEndian: BinaryEndian;
  LittleEndian: BinaryEndian;
  NativeEndian: BinaryEndian;
} = {
  BigEndian: bigEndian,
  LittleEndian: littleEndian,
  NativeEndian: nativeEndian(),
};

export function Read(
  reader: Reader | undefined,
  order: ByteOrder | undefined,
  data: GoInterfaceValue | undefined,
): GoError | undefined {
  if (order === undefined) {
    return GoPanic.raiseRuntime(
      "invalid memory address or nil pointer dereference",
    );
  }
  const boxed = reflect.ValueOf(data);
  let target = boxed;
  if (boxed.Kind().value === reflect.Pointer.value) {
    target = boxed.Elem();
  } else if (boxed.Kind().value !== reflect.Slice.value) {
    return invalidCodecType("binary.Read", boxed);
  }
  const size = encodedSize(target);
  if (size < 0n) {
    return invalidCodecType("binary.Read", boxed);
  }
  const buffer = RuntimeSliceValue.make<uint8>(size, null, 0);
  const outcome = ReadFull(reader, buffer);
  if (outcome[1] !== undefined) {
    return outcome[1];
  }
  decodeInto(order, buffer, 0, target);
  return undefined;
}

function invalidCodecType(operation: gostring, value: reflect.Value): GoError {
  return newError(`${operation}: invalid type ${codecTypeText(value)}`);
}

function codecTypeText(value: reflect.Value): gostring {
  const kind = value.Kind().value;
  const type = kind === reflect.Invalid.value ? undefined : value.Type();
  return type === undefined ? "<nil>" : type.String();
}

export function Write(
  writer: Writer | undefined,
  order: ByteOrder | undefined,
  data: GoInterfaceValue | undefined,
): GoError | undefined {
  if (writer === undefined || order === undefined) {
    return GoPanic.raiseRuntime(
      "invalid memory address or nil pointer dereference",
    );
  }
  const value = reflect.Indirect(reflect.ValueOf(data));
  const size = encodedSize(value);
  if (size < 0n) {
    return newError(
      `binary.Write: some values are not fixed-sized in type ${codecTypeText(value)}`,
    );
  }
  const buffer = RuntimeSliceValue.make<uint8>(size, null, 0);
  encodeFrom(order, buffer, 0, value);
  const outcome = writer.Write(buffer);
  return outcome[1];
}
