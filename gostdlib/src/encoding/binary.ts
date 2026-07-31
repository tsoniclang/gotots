import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
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

export interface ByteOrder extends GoInterfaceValue {
  PutUint16(buffer: RuntimeSlice<uint8>, value: uint16): void;
  PutUint32(buffer: RuntimeSlice<uint8>, value: uint32): void;
  PutUint64(buffer: RuntimeSlice<uint8>, value: uint64): void;
  String(): gostring;
  Uint16(buffer: RuntimeSlice<uint8>): uint16;
  Uint32(buffer: RuntimeSlice<uint8>): uint32;
  Uint64(buffer: RuntimeSlice<uint8>): uint64;
}

const bigEndian: ByteOrder = new BigEndianOrder();
const littleEndian: ByteOrder = new LittleEndianOrder();

export const state: {
  BigEndian: ByteOrder;
  LittleEndian: ByteOrder;
  NativeEndian: ByteOrder;
} = {
  BigEndian: bigEndian,
  LittleEndian: littleEndian,
  NativeEndian: nativeEndian(),
};
