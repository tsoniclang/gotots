import { Buffer } from "../../bytes.js";
import {
  bufferRepresentationAssign,
  bufferRepresentationCopy,
} from "../portable/bytes/buffer.js";

export class BytesBufferOperations {
  static $copy(source: Buffer): Buffer {
    return bufferRepresentationCopy(source);
  }

  static $assign(target: Buffer, source: Buffer): void {
    bufferRepresentationAssign(target, source);
  }
}
