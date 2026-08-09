import type { Regexp } from "../portable/regexp/regexp.js";
import {
  regexpRepresentationAssign,
  regexpRepresentationCopy,
} from "../portable/regexp/regexp.js";

export {
  ReplaceAllStringFuncCooperative as RegexpReplaceAllStringFuncCanonical,
} from "../portable/regexp/regexp.js";

export class RegexpValueOperations {
  static $copy(source: Regexp): Regexp {
    return regexpRepresentationCopy(source);
  }

  static $assign(target: Regexp, source: Regexp): void {
    regexpRepresentationAssign(target, source);
  }
}
