import type { int8 } from "@gotots/gostdlib/internal/scalars.js";

import { Accuracy, Float, Int } from "../../math/big.js";
import {
  floatRepresentationAssign,
  floatRepresentationCopy,
  intRepresentationAssign,
  intRepresentationCopy,
} from "../portable/math/big.js";

export class MathBigAccuracyValueOperations {
  static $project(source: Accuracy): int8 {
    return source.value;
  }

  static $wrap(source: int8): Accuracy {
    return new Accuracy(source);
  }
}

export type MathBigFloatStorage = Float;

export class MathBigFloatOperations {
  static $assign(target: Float, source: Float): void {
    floatRepresentationAssign(target, source);
  }

  static $copy(source: Float): Float {
    return floatRepresentationCopy(source);
  }

  static $zero(): Float {
    return new Float();
  }

  static $storageOf(source: Float): MathBigFloatStorage {
    return source;
  }

  static $fromStorage(source: MathBigFloatStorage): Float {
    return source;
  }
}

export type MathBigIntStorage = Int;

export class MathBigIntOperations {
  static $assign(target: Int, source: Int): void {
    intRepresentationAssign(target, source);
  }

  static $copy(source: Int): Int {
    return intRepresentationCopy(source);
  }

  static $zero(): Int {
    return new Int();
  }

  static $storageOf(source: Int): MathBigIntStorage {
    return source;
  }

  static $fromStorage(source: MathBigIntStorage): Int {
    return source;
  }
}
