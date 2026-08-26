import { Regexp } from "../portable/regexp/regexp.js";
import {
  regexpRepresentationAssign,
  regexpRepresentationCopy,
} from "../portable/regexp/regexp.js";

export const RegexpReplaceAllStringFuncCanonical = Regexp.ReplaceAllStringFunc;

export class RegexpValueOperations {
  static $copy(source: Regexp): Regexp {
    return regexpRepresentationCopy(source);
  }

  static $assign(target: Regexp, source: Regexp): void {
    regexpRepresentationAssign(target, source);
  }
}
