import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";

import {
  type ByteOrder as ProviderByteOrder,
  Read,
  Write,
} from "../../encoding/binary.js";
import type {
  ProviderReaderInterface,
  ProviderWriterInterface,
} from "./provider-io-contract.js";
import type { ProviderErrorInterface } from "./provider-error.js";

export type {
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
  return Read(reader, order, data);
}

export function EncodingBinaryWriteDirect(
  writer: ProviderWriterInterface<ProviderErrorInterface> | undefined,
  order: ProviderByteOrder | undefined,
  data: GoInterfaceValue | undefined,
): ProviderErrorInterface | undefined {
  return Write(writer, order, data);
}
