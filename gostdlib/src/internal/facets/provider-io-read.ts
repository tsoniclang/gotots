import { GoPanic } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int, uint8 } from "@gotots/gostdlib/internal/scalars.js";

import { readFullSync } from "../portable/io/read.js";
import type { ProviderReaderInterface } from "./provider-io-contract.js";
import type { ProviderErrorInterface } from "./provider-error.js";

export type { ProviderReaderInterface } from "./provider-io-contract.js";
export type { ProviderErrorInterface } from "./provider-error.js";

export function IoReadFullDirect<
  Failure extends ProviderErrorInterface,
  Source extends ProviderReaderInterface<Failure>,
>(
  reader: Source | undefined,
  destination: RuntimeSlice<uint8>,
  eof: Failure | undefined,
  unexpectedEOF: Failure | undefined,
): [int, Failure | undefined] {
  const source = requireValue(reader, "io.Reader");
  return readFullSync(
    (target) => source.Read(target),
    destination,
    requireValue(eof, "io.EOF"),
    requireValue(unexpectedEOF, "io.ErrUnexpectedEOF"),
  );
}

function requireValue<Value>(
  value: Value | undefined,
  identity: string,
): Value {
  if (value === undefined) {
    GoPanic.raiseRuntime(`gostdlib provider supplied nil ${identity}`);
  }
  return value;
}
