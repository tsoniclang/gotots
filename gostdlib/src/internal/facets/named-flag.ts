import type { int64 } from "@gotots/gostdlib/internal/scalars.js";

import { ErrorHandling, FlagSet } from "../../flag.js";
import { assignFlagSet } from "../node/flag/flag-set.js";

export class FlagSetValueOperations {
  static $assign(target: FlagSet, source: FlagSet): void {
    if (target !== source) {
      assignFlagSet(target, source);
    }
  }

  static $copy(source: FlagSet): FlagSet {
    const target = new FlagSet();
    assignFlagSet(target, source);
    return target;
  }
}

export class FlagErrorHandlingValueOperations {
  static $project(source: ErrorHandling): int64 {
    return source.value;
  }

  static $wrap(source: int64): ErrorHandling {
    return new ErrorHandling(source);
  }
}
