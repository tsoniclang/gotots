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
  runGzipSourceSync,
} from "../portable/compress/gzip/header.js";
import { bytes, writeBytes } from "../runtime/slice.js";
import { goInterfaceEqual } from "../runtime/interface.js";
import { Time, UnixMilli } from "../../time.js";
import {
  CanonicalBoundaryErrorAsync,
} from "./provider-io-contract.js";
import type {
  CanonicalErrorAsync,
  CanonicalReaderSourceAsync,
  CanonicalReaderSourceSync,
} from "./provider-io-contract.js";

export type {
  CanonicalErrorAsync,
  CanonicalReaderSourceAsync,
  CanonicalReaderSourceSync,
} from "./provider-io-contract.js";

export interface CanonicalFlateReaderSync<
  Failure extends GoInterfaceValue,
> extends GoInterfaceValue {
  Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int64, Failure | undefined];
  ReadByte(recovery?: GoRecovery): [uint8, Failure | undefined];
}

export interface CanonicalFlateReaderReadAsync<
  Failure extends GoInterfaceValue,
> extends GoInterfaceValue {
  Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Awaitable<[int64, Failure | undefined]>;
  ReadByte(recovery?: GoRecovery): Awaitable<[uint8, Failure | undefined]>;
}

export interface CanonicalReadCloserReadAsync<
  Failure extends GoInterfaceValue,
> extends GoInterfaceValue {
  Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Awaitable<[int64, Failure | undefined]>;
  Close(recovery?: GoRecovery): Awaitable<Failure | undefined>;
}

export interface CanonicalReadCloserCloseAsync<
  Failure extends GoInterfaceValue,
> extends GoInterfaceValue {
  Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Awaitable<[int64, Failure | undefined]>;
  Close(recovery?: GoRecovery): Awaitable<Failure | undefined>;
}

class CanonicalGzipPayload<Failure extends CanonicalErrorAsync> {
  private decoded: Uint8Array | undefined;
  private offset = 0;
  private terminalFailure: CanonicalErrorAsync | undefined;
  private closed = false;

  constructor(
    private readonly eof: Failure,
    private readonly invalidError: (message: string) => CanonicalErrorAsync,
  ) {}

  Close(): CanonicalErrorAsync | undefined {
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
    sourceFailure: CanonicalErrorAsync | undefined,
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
  ): [int64, CanonicalErrorAsync | undefined] {
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

class CanonicalGzipReaderAsyncState<Failure extends CanonicalErrorAsync> {
  private readonly payload: CanonicalGzipPayload<Failure>;

  constructor(
    private readonly sourceState: GzipSourceState<CanonicalErrorAsync>,
    private readonly source: CanonicalReaderSourceAsync<Failure>,
    eof: Failure,
    invalidError: (message: string) => CanonicalErrorAsync,
  ) {
    this.payload = new CanonicalGzipPayload(eof, invalidError);
  }

  Close(): CanonicalErrorAsync | undefined {
    return this.payload.Close();
  }

  async Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Promise<[int64, CanonicalErrorAsync | undefined]> {
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

class CanonicalGzipReaderSyncState<Failure extends CanonicalErrorAsync> {
  private readonly payload: CanonicalGzipPayload<Failure>;

  constructor(
    private readonly sourceState: GzipSourceState<CanonicalErrorAsync>,
    private readonly source: CanonicalReaderSourceSync<Failure>,
    eof: Failure,
    invalidError: (message: string) => CanonicalErrorAsync,
  ) {
    this.payload = new CanonicalGzipPayload(eof, invalidError);
  }

  Close(): CanonicalErrorAsync | undefined {
    return this.payload.Close();
  }

  Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int64, CanonicalErrorAsync | undefined] {
    if (this.payload.NeedsLoad(destination)) {
      const [encoded, sourceFailure] = runGzipSourceSync(
        this.sourceState.beginDrain(),
        (target) => this.source.Read(target, recovery),
      );
      this.payload.Load(encoded, sourceFailure);
    }
    return this.payload.Read(destination);
  }
}

export class CanonicalGzipReaderReadAsync<
  FlateReader extends GoInterfaceValue,
  Failure extends CanonicalErrorAsync,
  ReadCloser extends GoInterfaceValue,
> {
  declare private readonly flateReaderContract: FlateReader;
  declare private readonly readCloserContract: ReadCloser;

  constructor(
    public Header: Header,
    private readonly state: CanonicalGzipReaderAsyncState<Failure>,
  ) {}

  static Close<
    FlateReader extends GoInterfaceValue,
    Failure extends CanonicalErrorAsync,
    ReadCloser extends GoInterfaceValue,
  >(
    receiver: CanonicalGzipReaderReadAsync<
      FlateReader,
      Failure,
      ReadCloser
    > | undefined,
    recovery?: GoRecovery,
  ): CanonicalErrorAsync | undefined {
    return requireValue(receiver, "gzip.Reader").Close(recovery);
  }

  static Read<
    FlateReader extends GoInterfaceValue,
    Failure extends CanonicalErrorAsync,
    ReadCloser extends GoInterfaceValue,
  >(
    receiver: CanonicalGzipReaderReadAsync<
      FlateReader,
      Failure,
      ReadCloser
    > | undefined,
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Promise<[int64, CanonicalErrorAsync | undefined]> {
    return requireValue(receiver, "gzip.Reader").Read(destination, recovery);
  }

  Close(_recovery?: GoRecovery): CanonicalErrorAsync | undefined {
    return this.state.Close();
  }

  async Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): Promise<[int64, CanonicalErrorAsync | undefined]> {
    return this.state.Read(destination, recovery);
  }
}

export class CanonicalGzipReaderCloseAsync<
  FlateReader extends GoInterfaceValue,
  Failure extends CanonicalErrorAsync,
  ReadCloser extends GoInterfaceValue,
> {
  declare private readonly flateReaderContract: FlateReader;
  declare private readonly readCloserContract: ReadCloser;

  constructor(
    public Header: Header,
    private readonly state: CanonicalGzipReaderSyncState<Failure>,
  ) {}

  static Close<
    FlateReader extends GoInterfaceValue,
    Failure extends CanonicalErrorAsync,
    ReadCloser extends GoInterfaceValue,
  >(
    receiver: CanonicalGzipReaderCloseAsync<
      FlateReader,
      Failure,
      ReadCloser
    > | undefined,
    recovery?: GoRecovery,
  ): Promise<CanonicalErrorAsync | undefined> {
    return requireValue(receiver, "gzip.Reader").Close(recovery);
  }

  static Read<
    FlateReader extends GoInterfaceValue,
    Failure extends CanonicalErrorAsync,
    ReadCloser extends GoInterfaceValue,
  >(
    receiver: CanonicalGzipReaderCloseAsync<
      FlateReader,
      Failure,
      ReadCloser
    > | undefined,
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int64, CanonicalErrorAsync | undefined] {
    return requireValue(receiver, "gzip.Reader").Read(destination, recovery);
  }

  async Close(_recovery?: GoRecovery): Promise<CanonicalErrorAsync | undefined> {
    return this.state.Close();
  }

  Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int64, CanonicalErrorAsync | undefined] {
    return this.state.Read(destination, recovery);
  }
}

async function initializeGzipReaderAsync<Failure extends CanonicalErrorAsync>(
  source: CanonicalReaderSourceAsync<Failure> | undefined,
  eof: Failure | undefined,
  unexpectedEOF: Failure | undefined,
  noProgress: Failure | undefined,
  errorContract: readonly object[],
): Promise<[
  Header | undefined,
  CanonicalGzipReaderAsyncState<Failure> | undefined,
  CanonicalErrorAsync | undefined,
]> {
  const invalidError = (message: string): CanonicalErrorAsync =>
    new CanonicalBoundaryErrorAsync(message, errorContract);
  if (source === undefined) {
    return [undefined, undefined, invalidError("gzip: nil Reader")];
  }
  const canonicalEOF = requireValue(eof, "io.EOF");
  const sourceState = new GzipSourceState<CanonicalErrorAsync>(
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
    new CanonicalGzipReaderAsyncState(
      sourceState,
      source,
      canonicalEOF,
      invalidError,
    ),
    undefined,
  ];
}

function initializeGzipReaderSync<Failure extends CanonicalErrorAsync>(
  source: CanonicalReaderSourceSync<Failure> | undefined,
  eof: Failure | undefined,
  unexpectedEOF: Failure | undefined,
  noProgress: Failure | undefined,
  errorContract: readonly object[],
): [
  Header | undefined,
  CanonicalGzipReaderSyncState<Failure> | undefined,
  CanonicalErrorAsync | undefined,
] {
  const invalidError = (message: string): CanonicalErrorAsync =>
    new CanonicalBoundaryErrorAsync(message, errorContract);
  if (source === undefined) {
    return [undefined, undefined, invalidError("gzip: nil Reader")];
  }
  const canonicalEOF = requireValue(eof, "io.EOF");
  const sourceState = new GzipSourceState<CanonicalErrorAsync>(
    canonicalEOF,
    requireValue(unexpectedEOF, "io.ErrUnexpectedEOF"),
    requireValue(noProgress, "io.ErrNoProgress"),
    () => invalidError("gzip: invalid header"),
  );
  const [header, failure] = runGzipSourceSync(
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
    new CanonicalGzipReaderSyncState(
      sourceState,
      source,
      canonicalEOF,
      invalidError,
    ),
    undefined,
  ];
}

export async function GzipNewReaderCanonicalReadAsync<
  FlateReader extends GoInterfaceValue,
  Failure extends CanonicalErrorAsync,
  ReadCloser extends GoInterfaceValue,
>(
  source: CanonicalReaderSourceAsync<Failure> | undefined,
  eof: Failure | undefined,
  unexpectedEOF: Failure | undefined,
  noProgress: Failure | undefined,
  errorContract: readonly object[],
): Promise<[
  CanonicalGzipReaderReadAsync<FlateReader, Failure, ReadCloser> | undefined,
  CanonicalErrorAsync | undefined,
]> {
  const [header, state, failure] = await initializeGzipReaderAsync(
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
    new CanonicalGzipReaderReadAsync<FlateReader, Failure, ReadCloser>(
      header,
      state,
    ),
    undefined,
  ];
}

export async function GzipNewReaderCanonicalCloseAsync<
  FlateReader extends GoInterfaceValue,
  Failure extends CanonicalErrorAsync,
  ReadCloser extends GoInterfaceValue,
>(
  source: CanonicalReaderSourceSync<Failure> | undefined,
  eof: Failure | undefined,
  unexpectedEOF: Failure | undefined,
  noProgress: Failure | undefined,
  errorContract: readonly object[],
): Promise<[
  CanonicalGzipReaderCloseAsync<FlateReader, Failure, ReadCloser> | undefined,
  CanonicalErrorAsync | undefined,
]> {
  const [header, state, failure] = initializeGzipReaderSync(
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
    new CanonicalGzipReaderCloseAsync<FlateReader, Failure, ReadCloser>(
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
