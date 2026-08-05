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
  readonly #state: ScannerState<Failure>;

  constructor(
    private readonly source: Source | undefined,
    badReadCount: Failure,
    tooLong: Failure,
    eof: Failure,
    noProgress: Failure,
  ) {
    this.#state = new ScannerState(badReadCount, tooLong, eof, noProgress);
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
      const [count, failure] = await requireSource(this.source).Read(step.target);
      this.#state.AcceptRead(step.target, count, failure);
    }
  }

  Text(): gostring {
    return this.#state.Text();
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
