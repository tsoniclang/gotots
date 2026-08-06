import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  gostring,
  uint16,
  uint32,
  uint64,
  uint8,
} from "@gotots/gostdlib/internal/scalars.js";

import type { ByteOrder } from "../../encoding/binary.js";

export function BinaryBigEndianPutUint16(
  receiver: ByteOrder,
  buffer: RuntimeSlice<uint8>,
  value: uint16,
  _recovery?: GoRecovery,
): void {
  receiver.PutUint16(buffer, value);
}

export function BinaryBigEndianPutUint32(
  receiver: ByteOrder,
  buffer: RuntimeSlice<uint8>,
  value: uint32,
  _recovery?: GoRecovery,
): void {
  receiver.PutUint32(buffer, value);
}

export function BinaryBigEndianPutUint64(
  receiver: ByteOrder,
  buffer: RuntimeSlice<uint8>,
  value: uint64,
  _recovery?: GoRecovery,
): void {
  receiver.PutUint64(buffer, value);
}

export function BinaryBigEndianString(
  receiver: ByteOrder,
  _recovery?: GoRecovery,
): gostring {
  return receiver.String();
}

export function BinaryBigEndianUint16(
  receiver: ByteOrder,
  buffer: RuntimeSlice<uint8>,
  _recovery?: GoRecovery,
): uint16 {
  return receiver.Uint16(buffer);
}

export function BinaryBigEndianUint32(
  receiver: ByteOrder,
  buffer: RuntimeSlice<uint8>,
  _recovery?: GoRecovery,
): uint32 {
  return receiver.Uint32(buffer);
}

export function BinaryBigEndianUint64(
  receiver: ByteOrder,
  buffer: RuntimeSlice<uint8>,
  _recovery?: GoRecovery,
): uint64 {
  return receiver.Uint64(buffer);
}

export function BinaryLittleEndianPutUint16(
  receiver: ByteOrder,
  buffer: RuntimeSlice<uint8>,
  value: uint16,
  _recovery?: GoRecovery,
): void {
  receiver.PutUint16(buffer, value);
}

export function BinaryLittleEndianPutUint32(
  receiver: ByteOrder,
  buffer: RuntimeSlice<uint8>,
  value: uint32,
  _recovery?: GoRecovery,
): void {
  receiver.PutUint32(buffer, value);
}

export function BinaryLittleEndianPutUint64(
  receiver: ByteOrder,
  buffer: RuntimeSlice<uint8>,
  value: uint64,
  _recovery?: GoRecovery,
): void {
  receiver.PutUint64(buffer, value);
}

export function BinaryLittleEndianString(
  receiver: ByteOrder,
  _recovery?: GoRecovery,
): gostring {
  return receiver.String();
}

export function BinaryLittleEndianUint16(
  receiver: ByteOrder,
  buffer: RuntimeSlice<uint8>,
  _recovery?: GoRecovery,
): uint16 {
  return receiver.Uint16(buffer);
}

export function BinaryLittleEndianUint32(
  receiver: ByteOrder,
  buffer: RuntimeSlice<uint8>,
  _recovery?: GoRecovery,
): uint32 {
  return receiver.Uint32(buffer);
}

export function BinaryLittleEndianUint64(
  receiver: ByteOrder,
  buffer: RuntimeSlice<uint8>,
  _recovery?: GoRecovery,
): uint64 {
  return receiver.Uint64(buffer);
}
