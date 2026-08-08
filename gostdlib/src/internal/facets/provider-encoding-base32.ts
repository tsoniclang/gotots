import type { Encoding } from "../portable/encoding/base32.js";
import {
  encodingRepresentationAssign,
  encodingRepresentationCopy,
} from "../portable/encoding/base32.js";

export class Base32EncodingOperations {
  static $copy(source: Encoding): Encoding {
    return encodingRepresentationCopy(source);
  }

  static $assign(target: Encoding, source: Encoding): void {
    encodingRepresentationAssign(target, source);
  }
}
