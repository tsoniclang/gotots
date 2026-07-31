import type { GoError } from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  bool,
  gostring,
  int64,
  uint64,
  uint8,
} from "@gotots/runtime/scalars.js";
import type {
  DirEntry,
  FS,
  FileInfo,
  FileMode,
} from "./io/fs.js";
import type { Time } from "./time.js";
import {
  changeTimes,
  create,
  directoryFileSystem,
  lstat,
  makeDirectories,
  open,
  readDirectory,
  remove,
  removeAll,
  stat,
} from "./internal/node/os/filesystem.js";
import {
  executable,
  environment,
  processArguments,
  temporaryDirectory,
  userCacheDirectory,
  workingDirectory,
} from "./internal/node/os/environment.js";
import {
  attachFileDescriptor,
  attachStandardFileDescriptor,
  closeFile,
  fileDescriptor,
  readFile,
  writeFile,
  writeFileString,
} from "./internal/node/os/file.js";
import {
  exitProcess,
  getProcessID,
  signalProcess,
} from "./internal/node/os/process.js";
import { isNotExistError } from "./internal/node/os/error.js";
import { stringSlice } from "./internal/runtime/slice.js";
import { SIGINT } from "./syscall.js";

export interface Signal {
  Signal(): void;
  String(): gostring;
}

export class File {
  static Close(receiver: File | undefined): GoError | undefined {
    return closeFile(receiver);
  }

  static Fd(receiver: File | undefined): uint64 {
    return fileDescriptor(receiver);
  }

  static Read(
    receiver: File | undefined,
    buffer: RuntimeSlice<uint8>,
  ): [int64, GoError | undefined] {
    return readFile(receiver, buffer);
  }

  static Write(
    receiver: File | undefined,
    buffer: RuntimeSlice<uint8>,
  ): [int64, GoError | undefined] {
    return writeFile(receiver, buffer);
  }

  static WriteString(
    receiver: File | undefined,
    text: gostring,
  ): [int64, GoError | undefined] {
    return writeFileString(receiver, text);
  }
}

export class Process {
  constructor(public Pid: int64 = 0) {}

  static Signal(
    receiver: Process | undefined,
    signal: Signal | undefined,
  ): GoError | undefined {
    return signalProcess(receiver, signal);
  }
}

export class ProcessState {}

export function Chtimes(
  name: gostring,
  accessTime: Time,
  modificationTime: Time,
): GoError | undefined {
  return changeTimes(name, accessTime, modificationTime);
}

export function Create(
  name: gostring,
): [File | undefined, GoError | undefined] {
  return create(name, newFile);
}

export function DirFS(directory: gostring): FS | undefined {
  return directoryFileSystem(directory);
}

export function Executable(): [gostring, GoError | undefined] {
  return executable();
}

export function Exit(code: int64): void {
  exitProcess(code);
}

export function FindProcess(
  pid: int64,
): [Process | undefined, GoError | undefined] {
  return [new Process(pid), undefined];
}

export function Getenv(name: gostring): gostring {
  return environment(name);
}

export function Getpid(): int64 {
  return getProcessID();
}

export function Getwd(): [gostring, GoError | undefined] {
  return workingDirectory();
}

export function IsNotExist(error: GoError | undefined): bool {
  return isNotExistError(error);
}

export function Lstat(
  name: gostring,
): [FileInfo | undefined, GoError | undefined] {
  return lstat(name);
}

export function MkdirAll(
  path: gostring,
  permissions: FileMode,
): GoError | undefined {
  return makeDirectories(path, permissions);
}

export function OpenFile(
  name: gostring,
  flags: int64,
  permissions: FileMode,
): [File | undefined, GoError | undefined] {
  return open(name, flags, permissions, newFile);
}

export function Remove(name: gostring): GoError | undefined {
  return remove(name);
}

export function RemoveAll(path: gostring): GoError | undefined {
  return removeAll(path);
}

export function ReadDir(
  name: gostring,
): [RuntimeSlice<DirEntry | undefined>, GoError | undefined] {
  return readDirectory(name);
}

export function Stat(
  name: gostring,
): [FileInfo | undefined, GoError | undefined] {
  return stat(name);
}

export function TempDir(): gostring {
  return temporaryDirectory();
}

export function UserCacheDir(): [gostring, GoError | undefined] {
  return userCacheDirectory();
}

export const state: {
  Args: RuntimeSlice<gostring>;
  Interrupt: Signal | undefined;
  Stderr: File | undefined;
  Stdin: File | undefined;
  Stdout: File | undefined;
} = {
  Args: stringSlice(processArguments()),
  Interrupt: SIGINT,
  Stderr: newStandardFile(2, "/dev/stderr"),
  Stdin: newStandardFile(0, "/dev/stdin"),
  Stdout: newStandardFile(1, "/dev/stdout"),
};

function newFile(descriptor: number, name: string): File {
  const file = new File();
  attachFileDescriptor(file, descriptor, name);
  return file;
}

function newStandardFile(descriptor: number, name: string): File {
  const file = new File();
  attachStandardFileDescriptor(file, descriptor, name);
  return file;
}
