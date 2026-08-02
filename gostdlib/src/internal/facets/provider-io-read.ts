import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int64, uint8 } from "@gotots/runtime/scalars.js";

import {
  readFullAsync,
  readFullSync,
} from "../portable/io/read.js";
import type {
  CanonicalReaderSourceAsync,
  CanonicalReaderSourceSync,
} from "./provider-io-contract.js";

export type {
  CanonicalErrorAsync,
  CanonicalErrorSync,
  CanonicalReaderSourceAsync,
  CanonicalReaderSourceSync,
} from "./provider-io-contract.js";

export function IoReadFullCanonicalSync<
  Failure extends GoInterfaceValue,
  Source extends CanonicalReaderSourceSync<Failure>,
>(
  reader: Source | undefined,
  destination: RuntimeSlice<uint8>,
  eof: Failure | undefined,
  unexpectedEOF: Failure | undefined,
): [int64, Failure | undefined] {
  const source = requireValue(reader, "io.Reader");
  return readFullSync(
    (target) => source.Read(target),
    destination,
    requireValue(eof, "io.EOF"),
    requireValue(unexpectedEOF, "io.ErrUnexpectedEOF"),
  );
}

export function IoReadFullCanonicalAsync<
  Failure extends GoInterfaceValue,
  Source extends CanonicalReaderSourceAsync<Failure>,
>(
  reader: Source | undefined,
  destination: RuntimeSlice<uint8>,
  eof: Failure | undefined,
  unexpectedEOF: Failure | undefined,
): Promise<[int64, Failure | undefined]> {
  const source = requireValue(reader, "io.Reader");
  return readFullAsync(
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
