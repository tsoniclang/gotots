import type { int32, uint8 } from "@gotots/runtime/scalars.js";

import { Kind, NameOff, TFlag, TypeOff } from "../../internal/abi.js";

export class InternalABIKindValueOperations {
  static $project(source: Kind): uint8 {
    return source.value;
  }

  static $wrap(source: uint8): Kind {
    return new Kind(source);
  }
}

export class InternalABINameOffValueOperations {
  static $project(source: NameOff): int32 {
    return source.value;
  }

  static $wrap(source: int32): NameOff {
    return new NameOff(source);
  }
}

export class InternalABITFlagValueOperations {
  static $project(source: TFlag): uint8 {
    return source.value;
  }

  static $wrap(source: uint8): TFlag {
    return new TFlag(source);
  }
}

export class InternalABITypeOffValueOperations {
  static $project(source: TypeOff): int32 {
    return source.value;
  }

  static $wrap(source: int32): TypeOff {
    return new TypeOff(source);
  }
}
