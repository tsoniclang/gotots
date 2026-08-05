import type { int64, uint64 } from "@gotots/gostdlib/internal/scalars.js";

import { Errno, Signal } from "../../syscall.js";

export class SyscallErrnoValueOperations {
  static $project(source: Errno): uint64 {
    return source.value;
  }

  static $wrap(source: uint64): Errno {
    return new Errno(source);
  }
}

export class SyscallSignalValueOperations {
  static $project(source: Signal): int64 {
    return source.value;
  }

  static $wrap(source: int64): Signal {
    return new Signal(source);
  }
}
