import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { Awaitable, int64, uint8 } from "@gotots/runtime/scalars.js";

import { Header } from "../../compress/gzip.js";
import { decodeGzip } from "../node/compress/gzip/decode.js";
import {
  GzipSourceState,
  runGzipSourceAsync,
} from "../portable/compress/gzip/header.js";
import { bytes, writeBytes } from "../runtime/slice.js";
import { goInterfaceEqual } from "../runtime/interface.js";
import { Time, UnixMilli } from "../../time.js";
import {
  CanonicalBoundaryError,
} from "./provider-io-contract.js";
import type {
  CanonicalError,
  CanonicalReader,
} from "./provider-io-contract.js";
import type { InterfaceContract } from "./provider-support.js";

export type {
  CanonicalError,
  CanonicalReader,
} from "./provider-io-contract.js";

export interface CanonicalFlateReader<
  Failure extends GoInterfaceValue,
> extends GoInterfaceValue {
  Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Awaitable<[int64, Failure | undefined]>;
  ReadByte(recovery?: GoRecovery): Awaitable<[uint8, Failure | undefined]>;
}

export interface CanonicalReadCloser<
  Failure extends GoInterfaceValue,
> extends GoInterfaceValue {
  Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Awaitable<[int64, Failure | undefined]>;
  Close(recovery?: GoRecovery): Awaitable<Failure | undefined>;
}

class CanonicalGzipPayload<Failure extends CanonicalError> {
  private decoded: Uint8Array | undefined;
  private offset = 0;
  private terminalFailure: CanonicalError | undefined;
  private closed = false;

  constructor(
    private readonly eof: Failure,
    private readonly invalidError: (message: string) => CanonicalError,
  ) {}

  Close(): CanonicalError | undefined {
    this.closed = true;
    return undefined;
  }

  NeedsLoad(destination: RuntimeSlice<uint8>): boolean {
    return !this.closed
      && destination.length !== 0
      && this.decoded === undefined
      && this.terminalFailure === undefined;
  }

  Load(
    encoded: RuntimeSlice<uint8>,
    sourceFailure: CanonicalError | undefined,
  ): void {
    if (sourceFailure !== undefined && !goInterfaceEqual(sourceFailure, this.eof)) {
      this.terminalFailure = sourceFailure;
      return;
    }
    try {
      this.decoded = decodeGzip(bytes(encoded));
    } catch {
      this.terminalFailure = this.invalidError("gzip: invalid checksum");
    }
  }

  Read(
    destination: RuntimeSlice<uint8>,
  ): [int64, CanonicalError | undefined] {
    if (this.closed) {
      return [0, this.invalidError("gzip: reader is closed")];
    }
    if (destination.length === 0) {
      return [0, undefined];
    }
    if (this.terminalFailure !== undefined) {
      return [0, this.terminalFailure];
    }
    if (this.decoded === undefined) {
      GoPanic.raiseRuntime("gzip: buffered reader was not initialized");
    }
    if (this.offset >= this.decoded.length) {
      return [0, this.eof];
    }
    const count = writeBytes(
      destination,
      this.decoded.subarray(this.offset),
    );
    this.offset += count;
    return [count, undefined];
  }
}

class CanonicalGzipReaderState<Failure extends CanonicalError> {
  private readonly payload: CanonicalGzipPayload<Failure>;

  constructor(
    private readonly sourceState: GzipSourceState<CanonicalError>,
    private readonly source: CanonicalReader<Failure>,
    eof: Failure,
    invalidError: (message: string) => CanonicalError,
  ) {
    this.payload = new CanonicalGzipPayload(eof, invalidError);
  }

  Close(): CanonicalError | undefined {
    return this.payload.Close();
  }

  async Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Promise<[int64, CanonicalError | undefined]> {
    if (this.payload.NeedsLoad(destination)) {
      const [encoded, sourceFailure] = await runGzipSourceAsync(
        this.sourceState.beginDrain(),
        (target) => this.source.Read(target, recovery),
      );
      this.payload.Load(encoded, sourceFailure);
    }
    return this.payload.Read(destination);
  }
}

export class CanonicalGzipReader<
  FlateReader extends GoInterfaceValue,
  Failure extends CanonicalError,
  ReadCloser extends GoInterfaceValue,
> {
  declare private readonly flateReaderContract: FlateReader;
  declare private readonly readCloserContract: ReadCloser;

  constructor(
    public Header: Header,
    private readonly state: CanonicalGzipReaderState<Failure>,
  ) {}

  static Close<
    FlateReader extends GoInterfaceValue,
    Failure extends CanonicalError,
    ReadCloser extends GoInterfaceValue,
  >(
    receiver: CanonicalGzipReader<
      FlateReader,
      Failure,
      ReadCloser
    > | undefined,
    recovery?: GoRecovery,
  ): CanonicalError | undefined {
    return requireValue(receiver, "gzip.Reader").Close(recovery);
  }

  static Read<
    FlateReader extends GoInterfaceValue,
    Failure extends CanonicalError,
    ReadCloser extends GoInterfaceValue,
  >(
    receiver: CanonicalGzipReader<
      FlateReader,
      Failure,
      ReadCloser
    > | undefined,
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Promise<[int64, CanonicalError | undefined]> {
    return requireValue(receiver, "gzip.Reader").Read(destination, recovery);
  }

  Close(_recovery?: GoRecovery): CanonicalError | undefined {
    return this.state.Close();
  }

  async Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Promise<[int64, CanonicalError | undefined]> {
    return this.state.Read(destination, recovery);
  }
}

async function initializeGzipReader<Failure extends CanonicalError>(
  source: CanonicalReader<Failure> | undefined,
  eof: Failure | undefined,
  unexpectedEOF: Failure | undefined,
  noProgress: Failure | undefined,
  errorContract: InterfaceContract,
): Promise<[
  Header | undefined,
  CanonicalGzipReaderState<Failure> | undefined,
  CanonicalError | undefined,
]> {
  const invalidError = (message: string): CanonicalError =>
    new CanonicalBoundaryError(message, errorContract);
  if (source === undefined) {
    return [undefined, undefined, invalidError("gzip: nil Reader")];
  }
  const canonicalEOF = requireValue(eof, "io.EOF");
  const sourceState = new GzipSourceState<CanonicalError>(
    canonicalEOF,
    requireValue(unexpectedEOF, "io.ErrUnexpectedEOF"),
    requireValue(noProgress, "io.ErrNoProgress"),
    () => invalidError("gzip: invalid header"),
  );
  const [header, failure] = await runGzipSourceAsync(
    sourceState.beginHeader(),
    (destination) => source.Read(destination),
  );
  if (failure !== undefined) {
    return [undefined, undefined, failure];
  }
  if (header === undefined) {
    return [undefined, undefined, invalidError("gzip: invalid header")];
  }
  return [
    new Header(
      header.comment,
      header.extra,
      header.modificationTimeSeconds === 0
        ? new Time()
        : UnixMilli(header.modificationTimeSeconds * 1000),
      header.name,
      header.operatingSystem,
    ),
    new CanonicalGzipReaderState(
      sourceState,
      source,
      canonicalEOF,
      invalidError,
    ),
    undefined,
  ];
}

export async function GzipNewReaderCanonical<
  FlateReader extends GoInterfaceValue,
  Failure extends CanonicalError,
  ReadCloser extends GoInterfaceValue,
>(
  source: CanonicalReader<Failure> | undefined,
  eof: Failure | undefined,
  unexpectedEOF: Failure | undefined,
  noProgress: Failure | undefined,
  errorContract: InterfaceContract,
): Promise<[
  CanonicalGzipReader<FlateReader, Failure, ReadCloser> | undefined,
  CanonicalError | undefined,
]> {
  const [header, state, failure] = await initializeGzipReader(
    source,
    eof,
    unexpectedEOF,
    noProgress,
    errorContract,
  );
  if (header === undefined || state === undefined) {
    return [undefined, failure];
  }
  return [
    new CanonicalGzipReader<FlateReader, Failure, ReadCloser>(
      header,
      state,
    ),
    undefined,
  ];
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
