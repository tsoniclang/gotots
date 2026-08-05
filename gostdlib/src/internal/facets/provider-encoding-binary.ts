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

import { providerPlaceholderMessage } from "../runtime/placeholder.js";
import {
  CanonicalBoundaryError,
  type CanonicalError,
  type CanonicalReader,
  type CanonicalWriter,
} from "./provider-io-contract.js";
import type { InterfaceContract } from "./provider-support.js";

export type {
  CanonicalError,
  CanonicalReader,
  CanonicalWriter,
} from "./provider-io-contract.js";

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

export function EncodingBinaryReadCanonical(
  _reader: CanonicalReader<CanonicalError> | undefined,
  _order: CanonicalByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
  errorContract: InterfaceContract,
): CanonicalError | undefined {
  return new CanonicalBoundaryError(
    providerPlaceholderMessage(
      "encoding/binary.Read requires generated reflection metadata",
    ),
    errorContract,
  );
}

export function EncodingBinaryWriteCanonical(
  _writer: CanonicalWriter<CanonicalError> | undefined,
  _order: CanonicalByteOrder | undefined,
  _data: GoInterfaceValue | undefined,
  errorContract: InterfaceContract,
): CanonicalError | undefined {
  return new CanonicalBoundaryError(
    providerPlaceholderMessage(
      "encoding/binary.Write requires generated reflection metadata",
    ),
    errorContract,
  );
}
