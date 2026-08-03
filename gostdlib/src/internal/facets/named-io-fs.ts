import { GoPanic } from "@gotots/runtime/panic.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { gostring } from "@gotots/runtime/scalars.js";
import type { uint32 } from "@gotots/runtime/scalars.js";

import { FileMode, PathError } from "../../io/fs.js";
import type { CanonicalErrorAsync } from "./provider-io-contract.js";

export type { CanonicalErrorAsync } from "./provider-io-contract.js";

export class IoFsFileModeValueOperations {
  static $project(source: FileMode): uint32 {
    return source.value;
  }

  static $wrap(source: uint32): FileMode {
    return new FileMode(source);
  }
}

export type IoFsPathErrorStorage = PathError;

export const IoFsPathErrorOperations = PathError;

export class CanonicalPathError<Failure extends CanonicalErrorAsync> {
  constructor(
    public Op: gostring,
    public Path: gostring,
    public Err: Failure | undefined,
  ) {}

  static $make<Failure extends CanonicalErrorAsync>(
    operation: gostring,
    path: gostring,
    failure: Failure | undefined,
  ): CanonicalPathError<Failure> {
    return new CanonicalPathError(operation, path, failure);
  }

  static $storageOf<Failure extends CanonicalErrorAsync>(
    source: CanonicalPathError<Failure>,
  ): CanonicalPathError<Failure> {
    return source;
  }

  static $fromStorage<Failure extends CanonicalErrorAsync>(
    source: CanonicalPathError<Failure>,
  ): CanonicalPathError<Failure> {
    return source;
  }

  static async Error<Failure extends CanonicalErrorAsync>(
    receiver: CanonicalPathError<Failure> | undefined,
    _recovery?: GoRecovery,
  ): Promise<gostring> {
    if (receiver === undefined) {
      return "<nil>";
    }
    return await receiver.Error();
  }

  static Unwrap<Failure extends CanonicalErrorAsync>(
    receiver: CanonicalPathError<Failure> | undefined,
  ): Failure | undefined {
    if (receiver === undefined) {
      GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
    }
    return receiver.Unwrap();
  }

  async Error(_recovery?: GoRecovery): Promise<gostring> {
    const detail = this.Err === undefined ? "<nil>" : await this.Err.Error();
    if (this.Op === "") {
      return `${this.Path}: ${detail}`;
    }
    if (this.Path === "") {
      return `${this.Op}: ${detail}`;
    }
    return `${this.Op} ${this.Path}: ${detail}`;
  }

  Unwrap(): Failure | undefined {
    return this.Err;
  }
}
