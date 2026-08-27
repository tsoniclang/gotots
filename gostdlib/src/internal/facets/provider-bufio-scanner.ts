import { GoPanic } from "@gotots/runtime/panic.js";
import type { bool, gostring } from "@gotots/gostdlib/internal/scalars.js";

import { ScannerState } from "../portable/io/scanner.js";
import type { ProviderReaderInterface } from "./provider-io-contract.js";
import type { ProviderErrorInterface } from "./provider-error.js";

export type { ProviderReaderInterface } from "./provider-io-contract.js";
export type { ProviderErrorInterface } from "./provider-error.js";

export class DirectBufioScanner<
  Failure extends ProviderErrorInterface,
  Source extends ProviderReaderInterface<Failure>,
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

  static $copy<
    Failure extends ProviderErrorInterface,
    Source extends ProviderReaderInterface<Failure>,
  >(
    source: DirectBufioScanner<Failure, Source>,
  ): DirectBufioScanner<Failure, Source> {
    const target = new DirectBufioScanner(
      source.#source,
      source.#badReadCount,
      source.#tooLong,
      source.#eof,
      source.#noProgress,
    );
    target.#state = source.#state.copy();
    return target;
  }

  static $assign<
    Failure extends ProviderErrorInterface,
    Source extends ProviderReaderInterface<Failure>,
  >(
    target: DirectBufioScanner<Failure, Source>,
    source: DirectBufioScanner<Failure, Source>,
  ): void {
    target.#badReadCount = source.#badReadCount;
    target.#tooLong = source.#tooLong;
    target.#eof = source.#eof;
    target.#noProgress = source.#noProgress;
    target.#source = source.#source;
    target.#state = source.#state.copy();
  }

  static Err<
    Failure extends ProviderErrorInterface,
    Source extends ProviderReaderInterface<Failure>,
  >(
    receiver: DirectBufioScanner<Failure, Source> | undefined,
  ): Failure | undefined {
    return requireScanner(receiver).Err();
  }

  static Scan<
    Failure extends ProviderErrorInterface,
    Source extends ProviderReaderInterface<Failure>,
  >(
    receiver: DirectBufioScanner<Failure, Source> | undefined,
  ): bool {
    return requireScanner(receiver).Scan();
  }

  static Text<
    Failure extends ProviderErrorInterface,
    Source extends ProviderReaderInterface<Failure>,
  >(
    receiver: DirectBufioScanner<Failure, Source> | undefined,
  ): gostring {
    return requireScanner(receiver).Text();
  }

  Err(): Failure | undefined {
    return this.#state.Err();
  }

  Scan(): bool {
    for (;;) {
      const step = this.#state.Next();
      if (step.kind === "result") {
        return step.value;
      }
      const [count, failure] = requireSource(this.#source).Read(step.target);
      this.#state.AcceptRead(step.target, count, failure);
    }
  }

  Text(): gostring {
    return this.#state.Text();
  }
}

export class BufioScannerOperations {
  static $copy<
    Failure extends ProviderErrorInterface,
    Source extends ProviderReaderInterface<Failure>,
  >(
    source: DirectBufioScanner<Failure, Source>,
  ): DirectBufioScanner<Failure, Source> {
    return DirectBufioScanner.$copy(source);
  }

  static $assign<
    Failure extends ProviderErrorInterface,
    Source extends ProviderReaderInterface<Failure>,
  >(
    target: DirectBufioScanner<Failure, Source>,
    source: DirectBufioScanner<Failure, Source>,
  ): void {
    DirectBufioScanner.$assign(target, source);
  }
}

export function NewScannerDirect<
  Failure extends ProviderErrorInterface,
  Source extends ProviderReaderInterface<Failure>,
>(
  source: Source | undefined,
  badReadCount: Failure | undefined,
  tooLong: Failure | undefined,
  eof: Failure | undefined,
  noProgress: Failure | undefined,
): DirectBufioScanner<Failure, Source> {
  return new DirectBufioScanner(
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
  Failure extends ProviderErrorInterface,
  Source extends ProviderReaderInterface<Failure>,
>(
  receiver: DirectBufioScanner<Failure, Source> | undefined,
): DirectBufioScanner<Failure, Source> {
  if (receiver === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return receiver;
}
