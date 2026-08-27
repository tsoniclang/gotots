import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { RuntimeSlice } from "@gotots/runtime/slice.js";
import type { int, uint8 } from "@gotots/gostdlib/internal/scalars.js";
import { integerFromHost } from "../host-integer.js";

import { Header } from "../../compress/gzip.js";
import { Time, UnixMilli } from "../../time.js";
import { decodeGzip } from "../node/compress/gzip/decode.js";
import {
  GzipSourceState,
  runGzipSourceSync,
} from "../portable/compress/gzip/header.js";
import { timeRepresentationCopy } from "../portable/time/time.js";
import { bytes, writeBytes } from "../runtime/slice.js";
import { goInterfaceEqual } from "../runtime/interface.js";
import { CanonicalBoundaryError } from "./provider-io-contract.js";
import type { ProviderReaderInterface } from "./provider-io-contract.js";
import type { ProviderErrorInterface } from "./provider-error.js";
import type { InterfaceContract } from "./provider-support.js";

export type { ProviderReaderInterface } from "./provider-io-contract.js";
export type { ProviderErrorInterface } from "./provider-error.js";

export interface ProviderFlateReader<
  Failure extends ProviderErrorInterface,
> extends GoInterfaceValue {
  Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int, Failure | undefined];
  ReadByte(recovery?: GoRecovery): [uint8, Failure | undefined];
}

export interface ProviderReadCloser<
  Failure extends ProviderErrorInterface,
> extends GoInterfaceValue {
  Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int, Failure | undefined];
  Close(recovery?: GoRecovery): Failure | undefined;
}

class DirectGzipPayload<Failure extends ProviderErrorInterface> {
  #closed = false;
  #decoded: Uint8Array | undefined;
  #offset = 0;
  #terminalFailure: ProviderErrorInterface | undefined;

  constructor(
    private readonly eof: Failure,
    private readonly invalidError: (message: string) => ProviderErrorInterface,
  ) {}

  Close(): ProviderErrorInterface | undefined {
    this.#closed = true;
    return undefined;
  }

  NeedsLoad(destination: RuntimeSlice<uint8>): boolean {
    return !this.#closed && destination.length !== 0 &&
      this.#decoded === undefined && this.#terminalFailure === undefined;
  }

  Load(
    encoded: RuntimeSlice<uint8>,
    sourceFailure: ProviderErrorInterface | undefined,
  ): void {
    if (sourceFailure !== undefined && !goInterfaceEqual(sourceFailure, this.eof)) {
      this.#terminalFailure = sourceFailure;
      return;
    }
    try {
      this.#decoded = decodeGzip(bytes(encoded));
    } catch {
      this.#terminalFailure = this.invalidError("gzip: invalid checksum");
    }
  }

  Read(
    destination: RuntimeSlice<uint8>,
  ): [int, ProviderErrorInterface | undefined] {
    if (this.#closed) {
      return [0n, this.invalidError("gzip: reader is closed")];
    }
    if (destination.length === 0) {
      return [0n, undefined];
    }
    if (this.#terminalFailure !== undefined) {
      return [0n, this.#terminalFailure];
    }
    if (this.#decoded === undefined) {
      GoPanic.raiseRuntime("gzip: buffered reader was not initialized");
    }
    if (this.#offset >= this.#decoded.length) {
      return [0n, this.eof];
    }
    const count = writeBytes(destination, this.#decoded.subarray(this.#offset));
    this.#offset += count;
    return [integerFromHost(count), undefined];
  }
}

class DirectGzipReaderState<Failure extends ProviderErrorInterface> {
  readonly #payload: DirectGzipPayload<Failure>;

  constructor(
    private readonly sourceState: GzipSourceState<ProviderErrorInterface>,
    private readonly source: ProviderReaderInterface<Failure>,
    eof: Failure,
    invalidError: (message: string) => ProviderErrorInterface,
  ) {
    this.#payload = new DirectGzipPayload(eof, invalidError);
  }

  Close(): ProviderErrorInterface | undefined {
    return this.#payload.Close();
  }

  Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int, ProviderErrorInterface | undefined] {
    if (this.#payload.NeedsLoad(destination)) {
      const [encoded, sourceFailure] = runGzipSourceSync(
        this.sourceState.beginDrain(),
        (target) => this.source.Read(target, recovery),
      );
      this.#payload.Load(encoded, sourceFailure);
    }
    return this.#payload.Read(destination);
  }
}

export class DirectGzipReader<
  FlateReader extends ProviderFlateReader<Failure>,
  Failure extends ProviderErrorInterface,
  ReadCloser extends ProviderReadCloser<Failure>,
> {
  declare private readonly flateReaderContract: FlateReader;
  declare private readonly readCloserContract: ReadCloser;

  constructor(
    public Header: Header,
    private state: DirectGzipReaderState<Failure>,
  ) {}

  static $copy<
    FlateReader extends ProviderFlateReader<Failure>,
    Failure extends ProviderErrorInterface,
    ReadCloser extends ProviderReadCloser<Failure>,
  >(
    source: DirectGzipReader<FlateReader, Failure, ReadCloser>,
  ): DirectGzipReader<FlateReader, Failure, ReadCloser> {
    return new DirectGzipReader(copyHeader(source.Header), source.state);
  }

  static $assign<
    FlateReader extends ProviderFlateReader<Failure>,
    Failure extends ProviderErrorInterface,
    ReadCloser extends ProviderReadCloser<Failure>,
  >(
    target: DirectGzipReader<FlateReader, Failure, ReadCloser>,
    source: DirectGzipReader<FlateReader, Failure, ReadCloser>,
  ): void {
    target.Header = copyHeader(source.Header);
    target.state = source.state;
  }

  static Close<
    FlateReader extends ProviderFlateReader<Failure>,
    Failure extends ProviderErrorInterface,
    ReadCloser extends ProviderReadCloser<Failure>,
  >(
    receiver: DirectGzipReader<FlateReader, Failure, ReadCloser> | undefined,
    recovery?: GoRecovery,
  ): ProviderErrorInterface | undefined {
    return requireValue(receiver, "gzip.Reader").Close(recovery);
  }

  static Read<
    FlateReader extends ProviderFlateReader<Failure>,
    Failure extends ProviderErrorInterface,
    ReadCloser extends ProviderReadCloser<Failure>,
  >(
    receiver: DirectGzipReader<FlateReader, Failure, ReadCloser> | undefined,
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int, ProviderErrorInterface | undefined] {
    return requireValue(receiver, "gzip.Reader").Read(destination, recovery);
  }

  Close(_recovery?: GoRecovery): ProviderErrorInterface | undefined {
    return this.state.Close();
  }

  Read(
    destination: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int, ProviderErrorInterface | undefined] {
    return this.state.Read(destination, recovery);
  }
}

export function GzipNewReaderDirect<
  FlateReader extends ProviderFlateReader<Failure>,
  Failure extends ProviderErrorInterface,
  ReadCloser extends ProviderReadCloser<Failure>,
>(
  source: ProviderReaderInterface<Failure> | undefined,
  eof: Failure | undefined,
  unexpectedEOF: Failure | undefined,
  noProgress: Failure | undefined,
  errorContract: InterfaceContract,
): [
  DirectGzipReader<FlateReader, Failure, ReadCloser> | undefined,
  ProviderErrorInterface | undefined,
] {
  const invalidError = (message: string): ProviderErrorInterface =>
    new CanonicalBoundaryError(message, errorContract);
  if (source === undefined) {
    return [undefined, invalidError("gzip: nil Reader")];
  }
  const canonicalEOF = requireValue(eof, "io.EOF");
  const sourceState = new GzipSourceState<ProviderErrorInterface>(
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
    return [undefined, failure];
  }
  if (header === undefined) {
    return [undefined, invalidError("gzip: invalid header")];
  }
  return [
    new DirectGzipReader(
      new Header(
        header.comment,
        header.extra,
        header.modificationTimeSeconds === 0
          ? new Time()
          : UnixMilli(integerFromHost(header.modificationTimeSeconds * 1000)),
        header.name,
        header.operatingSystem,
      ),
      new DirectGzipReaderState(sourceState, source, canonicalEOF, invalidError),
    ),
    undefined,
  ];
}

function copyHeader(source: Header): Header {
  return new Header(
    source.Comment,
    source.Extra.slice(0, source.Extra.length, source.Extra.capacity),
    timeRepresentationCopy(source.ModTime),
    source.Name,
    source.OS,
  );
}

function requireValue<Value>(value: Value | undefined, identity: string): Value {
  if (value === undefined) {
    GoPanic.raiseRuntime(`gostdlib provider supplied nil ${identity}`);
  }
  return value;
}
