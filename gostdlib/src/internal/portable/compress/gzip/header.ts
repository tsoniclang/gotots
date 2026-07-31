import type { GoError } from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { uint8 } from "@gotots/runtime/scalars.js";

import type { Reader } from "../../../../io.js";
import { state as ioState } from "../../../../io.js";
import { byteSlice } from "../../../runtime/slice.js";
import { ProviderError } from "../../../runtime/error.js";
import { unexpectedEOF } from "../../io/read.js";

export interface GzipHeader {
  readonly comment: string;
  readonly extra: RuntimeSlice<uint8>;
  readonly modificationTimeSeconds: number;
  readonly name: string;
  readonly operatingSystem: uint8;
}

export class GzipSource {
  private readonly consumed: number[] = [];
  private readonly pending: number[] = [];
  private terminalFailure: GoError | undefined;

  constructor(private readonly source: Reader) {}

  ReadHeader(): [GzipHeader | undefined, GoError | undefined] {
    const [fixed, fixedFailure] = this.readExact(10);
    if (fixedFailure !== undefined) {
      return [undefined, fixedFailure];
    }
    if (
      fixed[0] !== 0x1f
      || fixed[1] !== 0x8b
      || fixed[2] !== 8
      || ((fixed[3] ?? 0) & 0xe0) !== 0
    ) {
      return [undefined, undefined];
    }

    const flags = fixed[3] ?? 0;
    let extra = RuntimeSlice.nil<uint8>();
    if ((flags & 0x04) !== 0) {
      const [lengthBytes, lengthFailure] = this.readExact(2);
      if (lengthFailure !== undefined) {
        return [undefined, lengthFailure];
      }
      const length = (lengthBytes[0] ?? 0) | ((lengthBytes[1] ?? 0) << 8);
      const [extraBytes, extraFailure] = this.readExact(length);
      if (extraFailure !== undefined) {
        return [undefined, extraFailure];
      }
      extra = byteSlice(extraBytes);
    }

    const [name, nameFailure] = (flags & 0x08) === 0
      ? ["", undefined] as const
      : this.readLatin1String();
    if (nameFailure !== undefined) {
      return [undefined, nameFailure];
    }

    const [comment, commentFailure] = (flags & 0x10) === 0
      ? ["", undefined] as const
      : this.readLatin1String();
    if (commentFailure !== undefined) {
      return [undefined, commentFailure];
    }

    if ((flags & 0x02) !== 0) {
      const [checksum, checksumFailure] = this.readExact(2);
      if (checksumFailure !== undefined) {
        return [undefined, checksumFailure];
      }
      const expected = (checksum[0] ?? 0) | ((checksum[1] ?? 0) << 8);
      const actual = crc32(this.consumed.slice(0, -2)) & 0xffff;
      if (expected !== actual) {
        return [undefined, new ProviderError("gzip: invalid header")];
      }
    }

    const modificationTimeSeconds = (
      (fixed[4] ?? 0)
      + (fixed[5] ?? 0) * 0x100
      + (fixed[6] ?? 0) * 0x1_0000
      + (fixed[7] ?? 0) * 0x1_000_000
    ) >>> 0;
    return [{
      comment,
      extra,
      modificationTimeSeconds,
      name,
      operatingSystem: fixed[9] ?? 0,
    }, undefined];
  }

  Drain(): [RuntimeSlice<uint8>, GoError | undefined] {
    while (this.terminalFailure === undefined) {
      this.fill();
      if (this.pending.length === 0 && this.terminalFailure === undefined) {
        continue;
      }
      this.pending.length = 0;
    }
    return [byteSlice(this.consumed), this.terminalFailure];
  }

  private readExact(count: number): [number[], GoError | undefined] {
    const values: number[] = [];
    while (values.length < count) {
      if (this.pending.length === 0) {
        this.fill();
      }
      while (this.pending.length > 0 && values.length < count) {
        const value = this.pending.shift();
        if (value !== undefined) {
          values.push(value);
        }
      }
      if (values.length < count && this.terminalFailure !== undefined) {
        return [
          values,
          values.length > 0 && this.terminalFailure === ioState.EOF
            ? unexpectedEOF
            : this.terminalFailure,
        ];
      }
    }
    return [values, undefined];
  }

  private readLatin1String(): [string, GoError | undefined] {
    const values: number[] = [];
    for (;;) {
      const [bytes, failure] = this.readExact(1);
      if (failure !== undefined) {
        return ["", failure];
      }
      const value = bytes[0] ?? 0;
      if (value === 0) {
        return [String.fromCharCode(...values), undefined];
      }
      if (values.length >= 511) {
        return ["", new ProviderError("gzip: invalid header")];
      }
      values.push(value);
    }
  }

  private fill(): void {
    if (this.terminalFailure !== undefined) {
      return;
    }
    const buffer = RuntimeSlice.make<uint8>(4096, 4096, 0);
    const [count, failure] = this.source.Read(buffer);
    for (let index = 0; index < count; index += 1) {
      const value = buffer.get(index);
      this.pending.push(value);
      this.consumed.push(value);
    }
    this.terminalFailure = failure;
  }
}

function crc32(values: readonly number[]): number {
  let checksum = 0xffff_ffff;
  for (const value of values) {
    checksum ^= value;
    for (let bit = 0; bit < 8; bit += 1) {
      checksum = (checksum >>> 1) ^ ((checksum & 1) === 0 ? 0 : 0xedb8_8320);
    }
  }
  return (checksum ^ 0xffff_ffff) >>> 0;
}
