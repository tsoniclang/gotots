import type { int64 } from "@gotots/runtime/scalars.js";

import { ErrorHandling } from "../../flag.js";

export class FlagErrorHandlingValueOperations {
  static $project(source: ErrorHandling): int64 {
    return source.value;
  }

  static $wrap(source: int64): ErrorHandling {
    return new ErrorHandling(source);
  }
}
