import type { GoError } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int64, uint8 } from "@gotots/runtime/scalars.js";

import {
  BufferedReaderState,
  BufferedWriterState,
} from "./internal/portable/io/buffer.js";
import type {
  Reader as IoReader,
  Writer as IoWriter,
} from "./io.js";

class BufferedReader {
  constructor(
    private readonly readOperation: (
      destination: RuntimeSlice<uint8>,
    ) => [int64, GoError | undefined],
    private readonly readByteOperation: () => [uint8, GoError | undefined],
    private readonly readBytesOperation: (
      delimiter: uint8,
    ) => [RuntimeSlice<uint8>, GoError | undefined],
  ) {}

  Read(destination: RuntimeSlice<uint8>): [int64, GoError | undefined] {
    return this.readOperation(destination);
  }

  ReadByte(): [uint8, GoError | undefined] {
    return this.readByteOperation();
  }

  ReadBytes(delimiter: uint8): [RuntimeSlice<uint8>, GoError | undefined] {
    return this.readBytesOperation(delimiter);
  }
}

export type Reader = BufferedReader;

export const Reader = Object.freeze({
  Read(
    receiver: Reader | undefined,
    destination: RuntimeSlice<uint8>,
  ): [int64, GoError | undefined] {
    return requireReader(receiver).Read(destination);
  },

  ReadByte(receiver: Reader | undefined): [uint8, GoError | undefined] {
    return requireReader(receiver).ReadByte();
  },

  ReadBytes(
    receiver: Reader | undefined,
    delimiter: uint8,
  ): [RuntimeSlice<uint8>, GoError | undefined] {
    return requireReader(receiver).ReadBytes(delimiter);
  },
});

function requireReader(receiver: Reader | undefined): Reader {
  if (receiver === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return receiver;
}

class BufferedWriter {
  constructor(
    private readonly flushOperation: () => GoError | undefined,
    private readonly writeOperation: (
      source: RuntimeSlice<uint8>,
    ) => [int64, GoError | undefined],
    private readonly writeByteOperation: (value: uint8) => GoError | undefined,
  ) {}

  Flush(): GoError | undefined {
    return this.flushOperation();
  }

  Write(source: RuntimeSlice<uint8>): [int64, GoError | undefined] {
    return this.writeOperation(source);
  }

  WriteByte(value: uint8): GoError | undefined {
    return this.writeByteOperation(value);
  }
}

export type Writer = BufferedWriter;

export const Writer = Object.freeze({
  Flush(receiver: Writer | undefined): GoError | undefined {
    return requireWriter(receiver).Flush();
  },

  Write(
    receiver: Writer | undefined,
    source: RuntimeSlice<uint8>,
  ): [int64, GoError | undefined] {
    return requireWriter(receiver).Write(source);
  },

  WriteByte(receiver: Writer | undefined, value: uint8): GoError | undefined {
    return requireWriter(receiver).WriteByte(value);
  },
});

function requireWriter(receiver: Writer | undefined): Writer {
  if (receiver === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return receiver;
}

export function NewReader(source: IoReader | undefined): Reader | undefined {
  if (source === undefined) {
    return undefined;
  }
  const state = new BufferedReaderState(source);
  return new BufferedReader(
    (destination) => state.Read(destination),
    () => state.ReadByte(),
    (delimiter) => state.ReadBytes(delimiter),
  );
}

export function NewWriter(target: IoWriter | undefined): Writer | undefined {
  if (target === undefined) {
    return undefined;
  }
  const state = new BufferedWriterState(target);
  return new BufferedWriter(
    () => state.Flush(),
    (source) => state.Write(source),
    (value) => state.WriteByte(value),
  );
}
