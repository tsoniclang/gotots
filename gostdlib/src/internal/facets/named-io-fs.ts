import type { GoError } from "@gotots/runtime/interface-value.js";
import type { gostring } from "@gotots/runtime/scalars.js";

import { PathError } from "../../io/fs.js";

export type IoFsPathErrorStorage = PathError;

export class IoFsPathErrorOperations {
  static $make(
    operation: gostring,
    path: gostring,
    failure: GoError | undefined,
  ): PathError {
    return new PathError(operation, path, failure);
  }

  static $storageOf(source: PathError): IoFsPathErrorStorage {
    return source;
  }

  static $fromStorage(source: IoFsPathErrorStorage): PathError {
    return source;
  }
}
