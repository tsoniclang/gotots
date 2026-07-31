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
} from "@gotots/runtime/scalars.js";

import { nativeEndian } from "../internal/node/encoding/binary/native.js";
import {
  BigEndianOrder,
  LittleEndianOrder,
} from "../internal/portable/encoding/binary/byte-order.js";
import { ProviderError } from "../internal/runtime/error.js";
import type { Reader, Writer } from "../io.js";

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
  _reader: Reader | undefined,
  _order: ByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
): GoError | undefined {
  return new ProviderError(
    "encoding/binary.Read requires generated reflection metadata",
  );
}

export function Write(
  _writer: Writer | undefined,
  _order: ByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
): GoError | undefined {
  return new ProviderError(
    "encoding/binary.Write requires generated reflection metadata",
  );
}
