import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import type { bool, gostring } from "@gotots/gostdlib/internal/scalars.js";

import type { CanonicalReader } from "./provider-io-contract.js";
import {
  ScannerState,
  scannerReadBuffer,
} from "../portable/io/scanner.js";

export type {
  CanonicalError,
  CanonicalReader,
} from "./provider-io-contract.js";

export class CanonicalBufioScanner<
  Failure extends GoInterfaceValue,
  Source extends CanonicalReader<Failure>,
> {
  #badReadCount: Failure;
  #eof: Failure;
  #noProgress: Failure;
  #source: Source | undefined;
  #state: ScannerState<Failure>;
  #tooLong: Failure;

  constructor(
    source: Source | undefined,
    badReadCount: Failure,
    tooLong: Failure,
    eof: Failure,
    noProgress: Failure,
  ) {
    this.#badReadCount = badReadCount;
    this.#tooLong = tooLong;
    this.#eof = eof;
    this.#noProgress = noProgress;
    this.#source = source;
    this.#state = new ScannerState(badReadCount, tooLong, eof, noProgress);
  }

  static $copy<Failure extends GoInterfaceValue, Source extends CanonicalReader<Failure>>(
    source: CanonicalBufioScanner<Failure, Source>,
  ): CanonicalBufioScanner<Failure, Source> {
    const target = new CanonicalBufioScanner(
      source.#source,
      source.#badReadCount,
      source.#tooLong,
      source.#eof,
      source.#noProgress,
    );
    target.#state = source.#state.copy();
    return target;
  }

  static $assign<Failure extends GoInterfaceValue, Source extends CanonicalReader<Failure>>(
    target: CanonicalBufioScanner<Failure, Source>,
    source: CanonicalBufioScanner<Failure, Source>,
  ): void {
    target.#badReadCount = source.#badReadCount;
    target.#tooLong = source.#tooLong;
    target.#eof = source.#eof;
    target.#noProgress = source.#noProgress;
    target.#source = source.#source;
    target.#state = source.#state.copy();
  }

  static Err<Failure extends GoInterfaceValue, Source extends CanonicalReader<Failure>>(
    receiver: CanonicalBufioScanner<Failure, Source> | undefined,
  ): Failure | undefined {
    return requireScanner(receiver).Err();
  }

  static Scan<Failure extends GoInterfaceValue, Source extends CanonicalReader<Failure>>(
    receiver: CanonicalBufioScanner<Failure, Source> | undefined,
  ): Promise<bool> {
    return requireScanner(receiver).Scan();
  }

  static Text<Failure extends GoInterfaceValue, Source extends CanonicalReader<Failure>>(
    receiver: CanonicalBufioScanner<Failure, Source> | undefined,
  ): gostring {
    return requireScanner(receiver).Text();
  }

  Err(): Failure | undefined {
    return this.#state.Err();
  }

  async Scan(): Promise<bool> {
    for (;;) {
      const step = this.#state.Next();
      if (step.kind === "result") {
        return step.value;
      }
      const [count, failure] = await requireSource(this.#source).Read(step.target);
      this.#state.AcceptRead(step.target, count, failure);
    }
  }

  Text(): gostring {
    return this.#state.Text();
  }
}

export class BufioScannerOperations {
  static $copy<
    Failure extends GoInterfaceValue,
    Source extends CanonicalReader<Failure>,
  >(
    source: CanonicalBufioScanner<Failure, Source>,
  ): CanonicalBufioScanner<Failure, Source> {
    return CanonicalBufioScanner.$copy(source);
  }

  static $assign<
    Failure extends GoInterfaceValue,
    Source extends CanonicalReader<Failure>,
  >(
    target: CanonicalBufioScanner<Failure, Source>,
    source: CanonicalBufioScanner<Failure, Source>,
  ): void {
    CanonicalBufioScanner.$assign(target, source);
  }
}

export function NewScannerCanonical<
  Failure extends GoInterfaceValue,
  Source extends CanonicalReader<Failure>,
>(
  source: Source | undefined,
  badReadCount: Failure | undefined,
  tooLong: Failure | undefined,
  eof: Failure | undefined,
  noProgress: Failure | undefined,
): CanonicalBufioScanner<Failure, Source> {
  return new CanonicalBufioScanner(
    source,
    requireFailure(badReadCount),
    requireFailure(tooLong),
    requireFailure(eof),
    requireFailure(noProgress),
  );
}

function requireFailure<Failure>(failure: Failure | undefined): Failure {
  if (failure === undefined) {
    GoPanic.raiseRuntime("gostdlib provider supplied a nil scanner failure");
  }
  return failure;
}

function requireSource<Source>(source: Source | undefined): Source {
  if (source === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return source;
}

function requireScanner<
  Failure extends GoInterfaceValue,
  Source extends CanonicalReader<Failure>,
>(
  receiver: CanonicalBufioScanner<Failure, Source> | undefined,
): CanonicalBufioScanner<Failure, Source> {
  if (receiver === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return receiver;
}
