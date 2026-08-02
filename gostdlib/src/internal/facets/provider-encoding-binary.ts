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

import type { ByteOrder } from "../../encoding/binary.js";
import type { Reader, Writer } from "../../io.js";
import {
  unsupportedReflectionMessage,
  unsupportedReflectionOperation,
} from "../portable/encoding/binary/reflection.js";
import {
  CanonicalBoundaryErrorAsync,
  CanonicalBoundaryErrorSync,
} from "./provider-io-contract.js";
import type {
  CanonicalErrorAsync,
  CanonicalErrorSync,
  CanonicalReaderSourceAsync,
  CanonicalReaderSourceSync,
  CanonicalWriterTargetAsync,
  CanonicalWriterTargetSync,
} from "./provider-io-contract.js";

export type {
  CanonicalErrorAsync,
  CanonicalErrorSync,
  CanonicalReaderSourceAsync,
  CanonicalReaderSourceSync,
  CanonicalWriterTargetAsync,
  CanonicalWriterTargetSync,
} from "./provider-io-contract.js";

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

export function EncodingBinaryReadCanonicalSyncReaderSyncError(
  _reader: CanonicalReaderSourceSync<CanonicalErrorSync> | undefined,
  _order: ByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
  errorContract: readonly object[],
): CanonicalErrorSync | undefined {
  return new CanonicalBoundaryErrorSync(
    unsupportedReflectionMessage("Read"),
    errorContract,
  );
}

export function EncodingBinaryReadCanonicalAsyncReaderSyncError(
  _reader: CanonicalReaderSourceAsync<CanonicalErrorSync> | undefined,
  _order: ByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
  errorContract: readonly object[],
): CanonicalErrorSync | undefined {
  return new CanonicalBoundaryErrorSync(
    unsupportedReflectionMessage("Read"),
    errorContract,
  );
}

export function EncodingBinaryReadCanonicalSyncReaderAsyncError(
  _reader: CanonicalReaderSourceSync<CanonicalErrorAsync> | undefined,
  _order: ByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
  errorContract: readonly object[],
): CanonicalErrorAsync | undefined {
  return new CanonicalBoundaryErrorAsync(
    unsupportedReflectionMessage("Read"),
    errorContract,
  );
}

export function EncodingBinaryReadCanonicalAsyncReaderAsyncError(
  _reader: CanonicalReaderSourceAsync<CanonicalErrorAsync> | undefined,
  _order: ByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
  errorContract: readonly object[],
): CanonicalErrorAsync | undefined {
  return new CanonicalBoundaryErrorAsync(
    unsupportedReflectionMessage("Read"),
    errorContract,
  );
}

export function EncodingBinaryReadCanonicalOrderSyncReaderSyncError(
  _reader: CanonicalReaderSourceSync<CanonicalErrorSync> | undefined,
  _order: CanonicalByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
  errorContract: readonly object[],
): CanonicalErrorSync | undefined {
  return new CanonicalBoundaryErrorSync(
    unsupportedReflectionMessage("Read"),
    errorContract,
  );
}

export function EncodingBinaryReadCanonicalOrderAsyncReaderSyncError(
  _reader: CanonicalReaderSourceAsync<CanonicalErrorSync> | undefined,
  _order: CanonicalByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
  errorContract: readonly object[],
): CanonicalErrorSync | undefined {
  return new CanonicalBoundaryErrorSync(
    unsupportedReflectionMessage("Read"),
    errorContract,
  );
}

export function EncodingBinaryReadCanonicalOrderSyncReaderAsyncError(
  _reader: CanonicalReaderSourceSync<CanonicalErrorAsync> | undefined,
  _order: CanonicalByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
  errorContract: readonly object[],
): CanonicalErrorAsync | undefined {
  return new CanonicalBoundaryErrorAsync(
    unsupportedReflectionMessage("Read"),
    errorContract,
  );
}

export function EncodingBinaryReadCanonicalOrderAsyncReaderAsyncError(
  _reader: CanonicalReaderSourceAsync<CanonicalErrorAsync> | undefined,
  _order: CanonicalByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
  errorContract: readonly object[],
): CanonicalErrorAsync | undefined {
  return new CanonicalBoundaryErrorAsync(
    unsupportedReflectionMessage("Read"),
    errorContract,
  );
}

export function EncodingBinaryWriteCanonicalSyncWriterSyncError(
  _writer: CanonicalWriterTargetSync<CanonicalErrorSync> | undefined,
  _order: ByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
  errorContract: readonly object[],
): CanonicalErrorSync | undefined {
  return new CanonicalBoundaryErrorSync(
    unsupportedReflectionMessage("Write"),
    errorContract,
  );
}

export function EncodingBinaryWriteCanonicalAsyncWriterSyncError(
  _writer: CanonicalWriterTargetAsync<CanonicalErrorSync> | undefined,
  _order: ByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
  errorContract: readonly object[],
): CanonicalErrorSync | undefined {
  return new CanonicalBoundaryErrorSync(
    unsupportedReflectionMessage("Write"),
    errorContract,
  );
}

export function EncodingBinaryWriteCanonicalSyncWriterAsyncError(
  _writer: CanonicalWriterTargetSync<CanonicalErrorAsync> | undefined,
  _order: ByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
  errorContract: readonly object[],
): CanonicalErrorAsync | undefined {
  return new CanonicalBoundaryErrorAsync(
    unsupportedReflectionMessage("Write"),
    errorContract,
  );
}

export function EncodingBinaryWriteCanonicalAsyncWriterAsyncError(
  _writer: CanonicalWriterTargetAsync<CanonicalErrorAsync> | undefined,
  _order: ByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
  errorContract: readonly object[],
): CanonicalErrorAsync | undefined {
  return new CanonicalBoundaryErrorAsync(
    unsupportedReflectionMessage("Write"),
    errorContract,
  );
}

export function EncodingBinaryWriteCanonicalOrderSyncWriterSyncError(
  _writer: CanonicalWriterTargetSync<CanonicalErrorSync> | undefined,
  _order: CanonicalByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
  errorContract: readonly object[],
): CanonicalErrorSync | undefined {
  return new CanonicalBoundaryErrorSync(
    unsupportedReflectionMessage("Write"),
    errorContract,
  );
}

export function EncodingBinaryWriteCanonicalOrderAsyncWriterSyncError(
  _writer: CanonicalWriterTargetAsync<CanonicalErrorSync> | undefined,
  _order: CanonicalByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
  errorContract: readonly object[],
): CanonicalErrorSync | undefined {
  return new CanonicalBoundaryErrorSync(
    unsupportedReflectionMessage("Write"),
    errorContract,
  );
}

export function EncodingBinaryWriteCanonicalOrderSyncWriterAsyncError(
  _writer: CanonicalWriterTargetSync<CanonicalErrorAsync> | undefined,
  _order: CanonicalByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
  errorContract: readonly object[],
): CanonicalErrorAsync | undefined {
  return new CanonicalBoundaryErrorAsync(
    unsupportedReflectionMessage("Write"),
    errorContract,
  );
}

export function EncodingBinaryWriteCanonicalOrderAsyncWriterAsyncError(
  _writer: CanonicalWriterTargetAsync<CanonicalErrorAsync> | undefined,
  _order: CanonicalByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
  errorContract: readonly object[],
): CanonicalErrorAsync | undefined {
  return new CanonicalBoundaryErrorAsync(
    unsupportedReflectionMessage("Write"),
    errorContract,
  );
}
