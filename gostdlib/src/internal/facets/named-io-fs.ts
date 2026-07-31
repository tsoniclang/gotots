import type { GoError } from "@gotots/runtime/interface-value.js";
import type { gostring } from "@gotots/runtime/scalars.js";
import type { uint32 } from "@gotots/runtime/scalars.js";

import { FileMode, PathError } from "../../io/fs.js";

export class IoFsFileModeValueOperations {
  static $project(source: FileMode): uint32 {
    return source.value;
  }

  static $wrap(source: uint32): FileMode {
    return new FileMode(source);
  }
}

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
