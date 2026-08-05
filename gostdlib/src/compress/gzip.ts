import type { GoError } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  gostring,
  int,
  uint8,
} from "@gotots/gostdlib/internal/scalars.js";
import { integerFromHost } from "../internal/host-integer.js";

import { decodeGzip } from "../internal/node/compress/gzip/decode.js";
import { GzipSource } from "../internal/portable/compress/gzip/header.js";
import { ProviderError } from "../internal/runtime/error.js";
import { bytes, writeBytes } from "../internal/runtime/slice.js";
import { state as ioState, type Reader as IoReader } from "../io.js";
import { Time, UnixMilli } from "../time.js";

export class Header {
  constructor(
    public Comment: gostring,
    public Extra: RuntimeSlice<uint8>,
    public ModTime: Time,
    public Name: gostring,
    public OS: uint8,
  ) {}
}

class GzipReader {
  private decoded: Uint8Array | undefined;
  private offset = 0;
  private terminalFailure: GoError | undefined;
  private closed = false;

  constructor(
    public Header: Header,
    private readonly load: () => [Uint8Array | undefined, GoError | undefined],
  ) {}

  Close(): GoError | undefined {
    this.closed = true;
    return undefined;
  }

  Read(destination: RuntimeSlice<uint8>): [int, GoError | undefined] {
    if (this.closed) {
      return [0n, new ProviderError("gzip: reader is closed")];
    }
    if (destination.length === 0) {
      return [0n, undefined];
    }
    if (this.decoded === undefined && this.terminalFailure === undefined) {
      const [decoded, failure] = this.load();
      this.decoded = decoded;
      this.terminalFailure = failure;
    }
    if (this.terminalFailure !== undefined) {
      return [0n, this.terminalFailure];
    }
    const decoded = this.decoded;
    if (decoded === undefined || this.offset >= decoded.length) {
      return [0n, ioState.EOF];
    }
    const count = writeBytes(destination, decoded.subarray(this.offset));
    this.offset += count;
    return [integerFromHost(count), undefined];
  }
}

export type Reader = GzipReader;

export const Reader = Object.freeze({
  Close(receiver: Reader | undefined): GoError | undefined {
    return requireReader(receiver).Close();
  },

  Read(
    receiver: Reader | undefined,
    destination: RuntimeSlice<uint8>,
  ): [int, GoError | undefined] {
    return requireReader(receiver).Read(destination);
  },
});

function requireReader(receiver: Reader | undefined): Reader {
  if (receiver === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return receiver;
}

export function NewReader(
  source: IoReader | undefined,
): [Reader | undefined, GoError | undefined] {
  if (source === undefined) {
    return [undefined, new ProviderError("gzip: nil Reader")];
  }
  const gzipSource = new GzipSource(source);
  const [header, failure] = gzipSource.ReadHeader();
  if (failure !== undefined) {
    return [undefined, failure];
  }
  if (header === undefined) {
    return [undefined, new ProviderError("gzip: invalid header")];
  }
  return [
    new GzipReader(
      new Header(
        header.comment,
        header.extra,
        header.modificationTimeSeconds === 0
          ? new Time()
          : UnixMilli(integerFromHost(header.modificationTimeSeconds * 1000)),
        header.name,
        header.operatingSystem,
      ),
      (): [Uint8Array | undefined, GoError | undefined] => {
        const [encoded, sourceFailure] = gzipSource.Drain();
        if (sourceFailure !== undefined && sourceFailure !== ioState.EOF) {
          return [undefined, sourceFailure];
        }
        try {
          return [decodeGzip(bytes(encoded)), undefined];
        } catch {
          return [undefined, new ProviderError("gzip: invalid checksum")];
        }
      },
    ),
    undefined,
  ];
}
