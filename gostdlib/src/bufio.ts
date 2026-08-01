import type { GoError } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { bool, gostring, int64, uint8 } from "@gotots/runtime/scalars.js";

import {
  BufferedReaderState,
  BufferedWriterState,
} from "./internal/portable/io/buffer.js";
import type {
  Reader as IoReader,
  Writer as IoWriter,
} from "./io.js";
import { state as ioState } from "./io.js";
import { ProviderError } from "./internal/runtime/error.js";

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

export class Scanner {
  readonly #buffer: uint8[] = [];
  #done = false;
  #emptyReads = 0;
  #failure: GoError | undefined;
  #pendingFailure: GoError | undefined;
  #token: gostring = "";

  constructor(private readonly source: IoReader | undefined) {}

  static Err(receiver: Scanner | undefined): GoError | undefined {
    return requireScanner(receiver).#failure;
  }

  static Scan(receiver: Scanner | undefined): bool {
    return requireScanner(receiver).#scan();
  }

  static Text(receiver: Scanner | undefined): gostring {
    return requireScanner(receiver).#token;
  }

  #scan(): bool {
    if (this.#done) {
      return false;
    }
    for (;;) {
      const newline = this.#buffer.indexOf(0x0a);
      if (newline >= 0) {
        this.#token = scanLine(this.#buffer.splice(0, newline + 1), true);
        return true;
      }
      if (this.#pendingFailure !== undefined) {
        if (this.#buffer.length > 0) {
          this.#token = scanLine(this.#buffer.splice(0), false);
          return true;
        }
        this.#done = true;
        if (this.#pendingFailure !== ioState.EOF) {
          this.#failure = this.#pendingFailure;
        }
        return false;
      }
      if (this.#buffer.length >= 64 * 1024) {
        this.#failure = new ProviderError("bufio.Scanner: token too long");
        this.#done = true;
        return false;
      }
      if (this.source === undefined) {
        return GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
      }
      const readBuffer = RuntimeSlice.make<uint8>(4096, 4096, 0);
      const [count, failure] = this.source.Read(readBuffer);
      if (!Number.isInteger(count) || count < 0 || count > readBuffer.length) {
        this.#failure = new ProviderError(
          "bufio.Scanner: Read returned impossible count",
        );
        this.#done = true;
        return false;
      }
      for (let index = 0; index < count; index += 1) {
        this.#buffer.push(readBuffer.get(index));
      }
      if (failure !== undefined) {
        this.#pendingFailure = failure;
      }
      if (count === 0 && failure === undefined) {
        this.#emptyReads += 1;
        if (this.#emptyReads > 100) {
          this.#failure = ioState.ErrNoProgress;
          this.#done = true;
          return false;
        }
      } else {
        this.#emptyReads = 0;
      }
    }
  }
}

function requireScanner(receiver: Scanner | undefined): Scanner {
  if (receiver === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return receiver;
}

function scanLine(source: readonly uint8[], terminated: boolean): gostring {
  let end = source.length - (terminated ? 1 : 0);
  if (end > 0 && source[end - 1] === 0x0d) {
    end -= 1;
  }
  let result = "";
  for (let index = 0; index < end; index += 1) {
    result += String.fromCharCode(source[index] ?? 0);
  }
  return result;
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

export function NewScanner(source: IoReader | undefined): Scanner {
  return new Scanner(source);
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
