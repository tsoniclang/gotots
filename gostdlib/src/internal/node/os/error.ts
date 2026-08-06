import type { GoError } from "@gotots/runtime/interface-value.js";
import { Is } from "../../../errors.js";
import { state as fsState } from "../../../io/fs.js";
import { ENOENT, ENOTDIR } from "../../../syscall.js";
import { WrappedProviderError } from "../../portable/errors/tree.js";
import { errnoError } from "../../portable/syscall/errno.js";

export type NodeErrorKind =
  | "closed"
  | "invalid"
  | "not-directory"
  | "not-exist"
  | "operation"
  | "permission";

const nodeProviderErrorType = Object.freeze({ comparable: true });

export class NodeProviderError extends WrappedProviderError {
  constructor(
    readonly kind: NodeErrorKind,
    private readonly operation: string,
    private readonly path: string | undefined = undefined,
  ) {
    super(nodeProviderErrorType);
  }

  Error(): string {
    const target = this.Unwrap().Error();
    return this.path === undefined
      ? `${this.operation}: ${target}`
      : `${this.operation} ${this.path}: ${target}`;
  }

  Unwrap(): GoError {
    switch (this.kind) {
      case "closed":
        return fsState.ErrClosed;
      case "not-exist":
        return errnoError(ENOENT);
      case "not-directory":
        return errnoError(ENOTDIR);
      case "permission":
        return fsState.ErrPermission;
      case "invalid":
      case "operation":
        return fsState.ErrInvalid;
    }
  }
}

export function nodeError(
  kind: NodeErrorKind,
  operation: string,
  path: string | undefined = undefined,
): NodeProviderError {
  return new NodeProviderError(kind, operation, path);
}

export function isNotExistError(error: GoError | undefined): boolean {
  return Is(error, fsState.ErrNotExist);
}
