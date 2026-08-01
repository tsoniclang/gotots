import type { GoError, GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int64, uint8 } from "@gotots/runtime/scalars.js";

import { New } from "./errors.js";
import {
  noProgress,
  readFull,
  shortBuffer,
  shortWrite,
  unexpectedEOF,
} from "./internal/portable/io/read.js";
import { ProviderInterfaceValue } from "./internal/portable/io/value.js";

export interface Closer extends GoInterfaceValue {
  Close(): GoError | undefined;
}

export interface Reader extends GoInterfaceValue {
  Read(buffer: RuntimeSlice<uint8>): [int64, GoError | undefined];
}

export interface Writer extends GoInterfaceValue {
  Write(buffer: RuntimeSlice<uint8>): [int64, GoError | undefined];
}

export interface ReadCloser extends Reader, Closer {}

export interface WriteCloser extends Writer, Closer {}

export interface ReadWriter extends Reader, Writer {}

export interface ReadWriteCloser extends Reader, Writer, Closer {}

const discardType = Object.freeze({});

class DiscardWriter extends ProviderInterfaceValue implements Writer {
  constructor() {
    super(discardType);
  }

  Write(buffer: RuntimeSlice<uint8>): [int64, GoError | undefined] {
    return [buffer.length, undefined];
  }
}

export const state: {
  Discard: Writer;
  EOF: GoError;
  ErrShortWrite: GoError;
  ErrShortBuffer: GoError;
  ErrUnexpectedEOF: GoError;
  ErrNoProgress: GoError;
} = {
  Discard: new DiscardWriter(),
  EOF: New("EOF"),
  ErrShortWrite: shortWrite,
  ErrShortBuffer: shortBuffer,
  ErrUnexpectedEOF: unexpectedEOF,
  ErrNoProgress: noProgress,
};

export function ReadFull(
  reader: Reader | undefined,
  buffer: RuntimeSlice<uint8>,
): [int64, GoError | undefined] {
  if (reader === undefined) {
    return [0, New("invalid nil Reader")];
  }
  return readFull(reader, buffer, state.EOF);
}
