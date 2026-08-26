import { GoPanic } from "@gotots/runtime/panic.js";
import { GoMapHash } from "@gotots/runtime/map.js";
import type { GoRecovery } from "@gotots/runtime/panic.js";
import type { gostring } from "@gotots/gostdlib/internal/scalars.js";
import type { uint32 } from "@gotots/gostdlib/internal/scalars.js";

import { FileMode, PathError } from "../../io/fs.js";
import type { CanonicalError } from "./provider-io-contract.js";
import type { ProviderErrorInterface } from "./provider-error.js";
import { goInterfaceEqual } from "../runtime/interface.js";

export type { CanonicalError } from "./provider-io-contract.js";
export type { ProviderErrorInterface } from "./provider-error.js";

export class IoFsFileModeValueOperations {
  static $project(source: FileMode): uint32 {
    return source.value;
  }

  static $wrap(source: uint32): FileMode {
    return new FileMode(source);
  }
}

export { PathError as IoFsPathErrorOperations };
export type IoFsPathErrorStorage = PathError;

export class CanonicalPathError<Failure extends CanonicalError> {
  constructor(
    public Op: gostring,
    public Path: gostring,
    public Err: Failure | undefined,
  ) {}

  static $make<Failure extends CanonicalError>(
    operation: gostring,
    path: gostring,
    failure: Failure | undefined,
  ): CanonicalPathError<Failure> {
    return new CanonicalPathError(operation, path, failure);
  }

  static $copy<Failure extends CanonicalError>(
    source: CanonicalPathError<Failure>,
  ): CanonicalPathError<Failure> {
    return new CanonicalPathError(source.Op, source.Path, source.Err);
  }

  static $assign<Failure extends CanonicalError>(
    target: CanonicalPathError<Failure>,
    source: CanonicalPathError<Failure>,
  ): void {
    const operation = source.Op;
    const path = source.Path;
    const failure = source.Err;
    target.Op = operation;
    target.Path = path;
    target.Err = failure;
  }

  static $equal<Failure extends CanonicalError>(
    left: CanonicalPathError<Failure>,
    right: CanonicalPathError<Failure>,
  ): boolean {
    return left.Op === right.Op &&
      left.Path === right.Path &&
      goInterfaceEqual(left.Err, right.Err);
  }

  static $hash<Failure extends CanonicalError>(
    source: CanonicalPathError<Failure>,
  ): number {
    let hash = GoMapHash.string(source.Op);
    hash = GoMapHash.mix(hash, GoMapHash.string(source.Path));
    return GoMapHash.mix(hash, source.Err?.$go$hash() ?? 0);
  }

  static $storageOf<Failure extends CanonicalError>(
    source: CanonicalPathError<Failure>,
  ): CanonicalPathError<Failure> {
    return source;
  }

  static $fromStorage<Failure extends CanonicalError>(
    source: CanonicalPathError<Failure>,
  ): CanonicalPathError<Failure> {
    return source;
  }

  static async Error<Failure extends CanonicalError>(
    receiver: CanonicalPathError<Failure> | undefined,
    _recovery?: GoRecovery,
  ): Promise<gostring> {
    if (receiver === undefined) {
      return "<nil>";
    }
    return await receiver.Error();
  }

  static Unwrap<Failure extends CanonicalError>(
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

export class DirectPathError<Failure extends ProviderErrorInterface> {
  constructor(
    public Op: gostring,
    public Path: gostring,
    public Err: Failure | undefined,
  ) {}

  static $make<Failure extends ProviderErrorInterface>(
    operation: gostring,
    path: gostring,
    failure: Failure | undefined,
  ): DirectPathError<Failure> {
    return new DirectPathError(operation, path, failure);
  }

  static $copy<Failure extends ProviderErrorInterface>(
    source: DirectPathError<Failure>,
  ): DirectPathError<Failure> {
    return new DirectPathError(source.Op, source.Path, source.Err);
  }

  static $assign<Failure extends ProviderErrorInterface>(
    target: DirectPathError<Failure>,
    source: DirectPathError<Failure>,
  ): void {
    target.Op = source.Op;
    target.Path = source.Path;
    target.Err = source.Err;
  }

  static $equal<Failure extends ProviderErrorInterface>(
    left: DirectPathError<Failure>,
    right: DirectPathError<Failure>,
  ): boolean {
    return left.Op === right.Op && left.Path === right.Path &&
      goInterfaceEqual(left.Err, right.Err);
  }

  static $hash<Failure extends ProviderErrorInterface>(
    source: DirectPathError<Failure>,
  ): number {
    let hash = GoMapHash.string(source.Op);
    hash = GoMapHash.mix(hash, GoMapHash.string(source.Path));
    return GoMapHash.mix(hash, source.Err?.$go$hash() ?? 0);
  }

  static $storageOf<Failure extends ProviderErrorInterface>(
    source: DirectPathError<Failure>,
  ): DirectPathError<Failure> {
    return source;
  }

  static $fromStorage<Failure extends ProviderErrorInterface>(
    source: DirectPathError<Failure>,
  ): DirectPathError<Failure> {
    return source;
  }

  static Error<Failure extends ProviderErrorInterface>(
    receiver: DirectPathError<Failure> | undefined,
    _recovery?: GoRecovery,
  ): gostring {
    return receiver === undefined ? "<nil>" : receiver.Error();
  }

  static Unwrap<Failure extends ProviderErrorInterface>(
    receiver: DirectPathError<Failure> | undefined,
  ): Failure | undefined {
    if (receiver === undefined) {
      GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
    }
    return receiver.Unwrap();
  }

  Error(_recovery?: GoRecovery): gostring {
    const detail = this.Err === undefined ? "<nil>" : this.Err.Error();
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
