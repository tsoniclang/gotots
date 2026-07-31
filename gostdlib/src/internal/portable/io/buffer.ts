import type { GoError } from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int64, uint8 } from "@gotots/runtime/scalars.js";

import type { Reader, Writer } from "../../../io.js";
import { byteSlice, writeBytes } from "../../runtime/slice.js";
import { ProviderError } from "../../runtime/error.js";
import { shortWrite } from "./read.js";

const defaultBufferSize = 4096;

export class BufferedReaderState {
  private readonly buffered: number[] = [];
  private pendingFailure: GoError | undefined;

  constructor(private readonly source: Reader) {}

  Read(destination: RuntimeSlice<uint8>): [int64, GoError | undefined] {
    if (destination.length === 0) {
      return [0, undefined];
    }
    this.fill();
    if (this.buffered.length === 0) {
      return [0, this.takeFailure()];
    }

    const count = Math.min(destination.length, this.buffered.length);
    writeBytes(destination, this.buffered.slice(0, count));
    this.buffered.splice(0, count);
    return [count, undefined];
  }

  ReadByte(): [uint8, GoError | undefined] {
    this.fill();
    const value = this.buffered.shift();
    if (value === undefined) {
      return [0, this.takeFailure()];
    }
    return [value, undefined];
  }

  ReadBytes(delimiter: uint8): [RuntimeSlice<uint8>, GoError | undefined] {
    const values: number[] = [];
    for (;;) {
      const [value, failure] = this.ReadByte();
      if (failure !== undefined) {
        return [byteSlice(values), failure];
      }
      values.push(value);
      if (value === delimiter) {
        return [byteSlice(values), undefined];
      }
    }
  }

  private fill(): void {
    if (this.buffered.length > 0 || this.pendingFailure !== undefined) {
      return;
    }
    for (let attempt = 0; attempt < 100; attempt += 1) {
      const target = RuntimeSlice.make<uint8>(defaultBufferSize, defaultBufferSize, 0);
      const [count, failure] = this.source.Read(target);
      for (let index = 0; index < count; index += 1) {
        this.buffered.push(target.get(index));
      }
      this.pendingFailure = failure;
      if (count > 0 || failure !== undefined) {
        return;
      }
    }
    this.pendingFailure = new ProviderError("multiple Read calls return no data or error");
  }

  private takeFailure(): GoError | undefined {
    const failure = this.pendingFailure;
    this.pendingFailure = undefined;
    return failure;
  }
}

export class BufferedWriterState {
  private readonly buffered: number[] = [];
  private pendingFailure: GoError | undefined;

  constructor(private readonly target: Writer) {}

  Flush(): GoError | undefined {
    if (this.pendingFailure !== undefined) {
      return this.pendingFailure;
    }
    while (this.buffered.length > 0) {
      const source = byteSlice(this.buffered);
      const [count, failure] = this.target.Write(source);
      if (count < 0 || count > this.buffered.length) {
        this.pendingFailure = shortWrite;
        return this.pendingFailure;
      }
      this.buffered.splice(0, count);
      if (count < source.length && failure === undefined) {
        this.pendingFailure = shortWrite;
        return this.pendingFailure;
      }
      if (failure !== undefined) {
        this.pendingFailure = failure;
        return failure;
      }
      if (count === 0) {
        this.pendingFailure = shortWrite;
        return this.pendingFailure;
      }
    }
    return undefined;
  }

  Write(source: RuntimeSlice<uint8>): [int64, GoError | undefined] {
    if (this.pendingFailure !== undefined) {
      return [0, this.pendingFailure];
    }

    let accepted = 0;
    while (accepted < source.length) {
      if (this.buffered.length === defaultBufferSize) {
        const failure = this.Flush();
        if (failure !== undefined) {
          return [accepted, failure];
        }
      }
      const available = defaultBufferSize - this.buffered.length;
      const count = Math.min(available, source.length - accepted);
      for (let index = 0; index < count; index += 1) {
        this.buffered.push(source.get(accepted + index));
      }
      accepted += count;
    }
    return [accepted, undefined];
  }

  WriteByte(value: uint8): GoError | undefined {
    if (this.buffered.length === defaultBufferSize) {
      const failure = this.Flush();
      if (failure !== undefined) {
        return failure;
      }
    }
    this.buffered.push(value);
    return undefined;
  }
}
