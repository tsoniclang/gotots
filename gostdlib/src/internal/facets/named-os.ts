import { File, Process } from "../../os.js";
import {
  assignFileValue,
  fileValueEqual,
  fileValueHash,
} from "../node/os/file.js";

export class OsFileOperations {
  static $copy(source: File): File {
    const target = new File();
    assignFileValue(target, source);
    return target;
  }

  static $assign(target: File, source: File): void {
    assignFileValue(target, source);
  }

  static $equal(left: File, right: File): boolean {
    return fileValueEqual(left, right);
  }

  static $hash(source: File): number {
    return fileValueHash(source);
  }
}

export class OsProcessOperations {
  static $copy(source: Process): Process {
    return new Process(source.Pid);
  }

  static $assign(target: Process, source: Process): void {
    target.Pid = source.Pid;
  }
}
