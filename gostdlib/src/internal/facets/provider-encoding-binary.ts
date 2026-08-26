import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  Awaitable,
  gostring,
  uint16,
  uint32,
  uint64,
  uint8,
} from "@gotots/gostdlib/internal/scalars.js";

import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice as RuntimeSliceValue } from "@gotots/runtime/slice.js";
import * as reflect from "../../reflect.js";
import {
  type ByteOrder as ProviderByteOrder,
  Read as readDirect,
  Write as writeDirect,
} from "../../encoding/binary.js";
import {
  decodeIntoAwaited,
  encodeFromAwaited,
  encodedSize,
} from "../portable/encoding/binary/reflection-codec.js";
import {
  CanonicalBoundaryError,
  type CanonicalError,
  type CanonicalReader,
  type CanonicalWriter,
  type ProviderReaderInterface,
  type ProviderWriterInterface,
} from "./provider-io-contract.js";
import type { ProviderErrorInterface } from "./provider-error.js";
import type { InterfaceContract } from "./provider-support.js";

export type {
  CanonicalError,
  CanonicalReader,
  CanonicalWriter,
  ProviderReaderInterface,
  ProviderWriterInterface,
} from "./provider-io-contract.js";
export type { ByteOrder as ProviderByteOrder } from "../../encoding/binary.js";
export type { ProviderErrorInterface } from "./provider-error.js";

export function EncodingBinaryReadDirect(
  reader: ProviderReaderInterface<ProviderErrorInterface> | undefined,
  order: ProviderByteOrder | undefined,
  data: GoInterfaceValue | undefined,
): ProviderErrorInterface | undefined {
  return readDirect(reader, order, data);
}

export function EncodingBinaryWriteDirect(
  writer: ProviderWriterInterface<ProviderErrorInterface> | undefined,
  order: ProviderByteOrder | undefined,
  data: GoInterfaceValue | undefined,
): ProviderErrorInterface | undefined {
  return writeDirect(writer, order, data);
}

export interface CanonicalByteOrder extends GoInterfaceValue {
  PutUint16(
    buffer: RuntimeSlice<uint8>,
    value: uint16,
    recovery?: GoRecovery,
  ): Awaitable<void>;
  PutUint32(
    buffer: RuntimeSlice<uint8>,
    value: uint32,
    recovery?: GoRecovery,
  ): Awaitable<void>;
  PutUint64(
    buffer: RuntimeSlice<uint8>,
    value: uint64,
    recovery?: GoRecovery,
  ): Awaitable<void>;
  String(recovery?: GoRecovery): Awaitable<gostring>;
  Uint16(buffer: RuntimeSlice<uint8>, recovery?: GoRecovery): Awaitable<uint16>;
  Uint32(buffer: RuntimeSlice<uint8>, recovery?: GoRecovery): Awaitable<uint32>;
  Uint64(buffer: RuntimeSlice<uint8>, recovery?: GoRecovery): Awaitable<uint64>;
}

export async function EncodingBinaryReadCanonical(
  reader: CanonicalReader<CanonicalError> | undefined,
  order: CanonicalByteOrder | undefined,
  data: GoInterfaceValue | undefined,
  errorContract: InterfaceContract,
): Promise<CanonicalError | undefined> {
  if (reader === undefined || order === undefined) {
    GoPanic.raiseRuntime(
      "invalid memory address or nil pointer dereference",
    );
  }
  const boxed = reflect.ValueOf(data);
  let target = boxed;
  if (boxed.Kind().value === reflect.Pointer.value) {
    target = boxed.Elem();
  } else if (boxed.Kind().value !== reflect.Slice.value) {
    return invalidCanonicalType("binary.Read", boxed, errorContract);
  }
  const size = encodedSize(target);
  if (size < 0n) {
    return invalidCanonicalType("binary.Read", boxed, errorContract);
  }
  const total = globalThis.Number(size);
  const buffer = RuntimeSliceValue.make<uint8>(size, null, 0);
  let filled = 0;
  while (filled < total) {
    const [count, failure] = await reader.Read(
      buffer.slice(filled, null, null),
    );
    filled += globalThis.Number(count);
    if (failure !== undefined) {
      return failure;
    }
    if (globalThis.Number(count) === 0 && filled < total) {
      return new CanonicalBoundaryError("unexpected EOF", errorContract);
    }
  }
  await decodeIntoAwaited(order, buffer, 0, target);
  return undefined;
}

function invalidCanonicalType(
  operation: string,
  value: reflect.Value,
  errorContract: InterfaceContract,
): CanonicalError {
  return new CanonicalBoundaryError(
    `${operation}: invalid type ${canonicalTypeText(value)}`,
    errorContract,
  );
}

function canonicalTypeText(value: reflect.Value): string {
  const kind = value.Kind().value;
  const type =
    kind === reflect.Invalid.value ? undefined : value.Type();
  return type === undefined ? "<nil>" : type.String();
}

export async function EncodingBinaryWriteCanonical(
  writer: CanonicalWriter<CanonicalError> | undefined,
  order: CanonicalByteOrder | undefined,
  data: GoInterfaceValue | undefined,
  errorContract: InterfaceContract,
): Promise<CanonicalError | undefined> {
  if (writer === undefined || order === undefined) {
    GoPanic.raiseRuntime(
      "invalid memory address or nil pointer dereference",
    );
  }
  const value = reflect.Indirect(reflect.ValueOf(data));
  const size = encodedSize(value);
  if (size < 0n) {
    return new CanonicalBoundaryError(
      `binary.Write: some values are not fixed-sized in type ${canonicalTypeText(value)}`,
      errorContract,
    );
  }
  const buffer = RuntimeSliceValue.make<uint8>(size, null, 0);
  await encodeFromAwaited(order, buffer, 0, value);
  const outcome = await writer.Write(buffer);
  return outcome[1];
}
