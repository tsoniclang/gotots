import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  gostring,
  uint16,
  uint32,
  uint64,
  uint8,
} from "@gotots/runtime/scalars.js";

import type { Reader, Writer } from "../../io.js";
import { unsupportedReflectionOperation } from "../portable/encoding/binary/reflection.js";

export interface CanonicalByteOrder extends GoInterfaceValue {
  PutUint16(
    buffer: RuntimeSlice<uint8>,
    value: uint16,
    recovery?: GoRecovery,
  ): void;
  PutUint32(
    buffer: RuntimeSlice<uint8>,
    value: uint32,
    recovery?: GoRecovery,
  ): void;
  PutUint64(
    buffer: RuntimeSlice<uint8>,
    value: uint64,
    recovery?: GoRecovery,
  ): void;
  String(recovery?: GoRecovery): Promise<gostring>;
  Uint16(buffer: RuntimeSlice<uint8>, recovery?: GoRecovery): uint16;
  Uint32(buffer: RuntimeSlice<uint8>, recovery?: GoRecovery): uint32;
  Uint64(buffer: RuntimeSlice<uint8>, recovery?: GoRecovery): uint64;
}

export function EncodingBinaryReadCanonical(
  _reader: Reader | undefined,
  _order: CanonicalByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
): GoError | undefined {
  return unsupportedReflectionOperation("Read");
}

export function EncodingBinaryWriteCanonical(
  _writer: Writer | undefined,
  _order: CanonicalByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
): GoError | undefined {
  return unsupportedReflectionOperation("Write");
}
