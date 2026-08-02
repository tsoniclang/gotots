import {
  mkdirSync,
  lstatSync,
  openSync,
  readdirSync,
  rmSync,
  rmdirSync,
  statSync,
  utimesSync,
} from "node:fs";
import type {
  Dirent,
  Stats,
} from "node:fs";
import {
  basename,
  join,
  resolve,
  sep,
} from "node:path";
import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  gostring,
  int64,
} from "@gotots/runtime/scalars.js";
import type {
  DirEntry,
  FS,
  File as FsFile,
  FileInfo,
} from "../../../io/fs.js";
import { FileMode } from "../../../io/fs.js";
import { Time, UnixMilli } from "../../../time.js";
import { ProviderInterfaceValue } from "../../portable/io/value.js";
import {
  attachFileDescriptor,
  closeFile,
  readFile,
} from "./file.js";
import { nodeError } from "./error.js";

const modeDirectory = 0x80000000;
const modeDevice = 0x04000000;
const modeNamedPipe = 0x02000000;
const modeSocket = 0x01000000;
const modeSymlink = 0x08000000;
const modeCharDevice = 0x00200000;
const modeIrregular = 0x00080000;

export type FileFactory<T extends object> = (
  descriptor: number,
  name: string,
) => T;

export function create<T extends object>(
  path: gostring,
  factory: FileFactory<T>,
): [T | undefined, GoError | undefined] {
  return open(path, 0x242, new FileMode(0o666), factory);
}

export function open<T extends object>(
  path: gostring,
  flags: int64,
  permissions: FileMode,
  factory: FileFactory<T>,
): [T | undefined, GoError | undefined] {
  try {
    const descriptor = openSync(path, flags, permissions.value);
    return [factory(descriptor, path), undefined];
  } catch {
    if (statSync(path, { throwIfNoEntry: false }) === undefined) {
      return [undefined, nodeError("not-exist", "open", path)];
    }
    return [undefined, nodeError("operation", "open", path)];
  }
}

export function makeDirectories(
  path: gostring,
  permissions: FileMode,
): GoError | undefined {
  if (path.length === 0) {
    return nodeError("not-exist", "mkdir", path);
  }
  try {
    mkdirSync(path, {
      recursive: true,
      mode: permissions.value,
    });
    return undefined;
  } catch {
    return nodeError("operation", "mkdir", path);
  }
}

export function remove(path: gostring): GoError | undefined {
  const information = lstatSync(path, { throwIfNoEntry: false });
  if (information === undefined) {
    return nodeError("not-exist", "remove", path);
  }
  try {
    if (information.isDirectory()) {
      rmdirSync(path);
    } else {
      rmSync(path);
    }
    return undefined;
  } catch {
    return nodeError("operation", "remove", path);
  }
}

export function removeAll(path: gostring): GoError | undefined {
  try {
    rmSync(path, {
      force: true,
      recursive: true,
    });
    return undefined;
  } catch {
    return nodeError("operation", "removeall", path);
  }
}

export function changeTimes(
  path: gostring,
  accessTime: Time,
  modificationTime: Time,
): GoError | undefined {
  try {
    utimesSync(
      path,
      accessTime.UnixMilli() / 1000,
      modificationTime.UnixMilli() / 1000,
    );
    return undefined;
  } catch {
    if (statSync(path, { throwIfNoEntry: false }) === undefined) {
      return nodeError("not-exist", "chtimes", path);
    }
    return nodeError("operation", "chtimes", path);
  }
}

export function stat(
  path: gostring,
): [FileInfo | undefined, GoError | undefined] {
  try {
    const information = statSync(path, { throwIfNoEntry: false });
    if (information === undefined) {
      return [undefined, nodeError("not-exist", "stat", path)];
    }
    return [new NodeFileInfo(path, information), undefined];
  } catch {
    return [undefined, nodeError("operation", "stat", path)];
  }
}

export function lstat(
  path: gostring,
): [FileInfo | undefined, GoError | undefined] {
  try {
    const information = lstatSync(path, { throwIfNoEntry: false });
    if (information === undefined) {
      return [undefined, nodeError("not-exist", "lstat", path)];
    }
    return [new NodeFileInfo(path, information), undefined];
  } catch {
    return [undefined, nodeError("operation", "lstat", path)];
  }
}

export function readDirectory(
  path: gostring,
): [RuntimeSlice<DirEntry | undefined>, GoError | undefined] {
  try {
    const information = statSync(path, { throwIfNoEntry: false });
    if (information === undefined) {
      return [
        RuntimeSlice.nil<DirEntry | undefined>(),
        nodeError("not-exist", "readdir", path),
      ];
    }
    if (!information.isDirectory()) {
      return [
        RuntimeSlice.nil<DirEntry | undefined>(),
        nodeError("not-directory", "readdir", path),
      ];
    }
    const entries = readdirSync(path, { withFileTypes: true });
    entries.sort((left, right): number => Buffer.compare(
      Buffer.from(left.name),
      Buffer.from(right.name),
    ));
    return [
      RuntimeSlice.literal(entries.map(
        (entry): DirEntry => new NodeDirectoryEntry(path, entry),
      )),
      undefined,
    ];
  } catch {
    return [
      RuntimeSlice.nil<DirEntry | undefined>(),
      nodeError("operation", "readdir", path),
    ];
  }
}

export function directoryFileSystem(root: gostring): FS {
  return new NodeDirectoryFS(root);
}

const directoryFileSystemType = Object.freeze({ comparable: true });

class NodeDirectoryFS extends ProviderInterfaceValue implements FS {
  constructor(private readonly root: string) {
    super(directoryFileSystemType);
  }

  Open(name: gostring): [FsFile | undefined, GoError | undefined] {
    const path = resolveFileSystemPath(this.root, name);
    if (path === undefined) {
      return [undefined, nodeError("invalid", "open", name)];
    }
    const [file, error] = open(
      path,
      0,
      new FileMode(0),
      (descriptor: number, openedPath: string): object => {
        const value = {};
        attachFileDescriptor(value, descriptor, openedPath);
        return value;
      },
    );
    if (file === undefined) {
      return [undefined, error];
    }
    return [new NodeFileSystemFile(file, path), undefined];
  }
}

const fileSystemFileType = Object.freeze({ comparable: true });

class NodeFileSystemFile extends ProviderInterfaceValue implements FsFile {
  constructor(
    private readonly file: object,
    private readonly path: string,
  ) {
    super(fileSystemFileType);
  }

  Close(): GoError | undefined {
    return closeFile(this.file);
  }

  Read(
    buffer: Parameters<typeof readFile>[1],
  ): ReturnType<typeof readFile> {
    return readFile(this.file, buffer);
  }

  Stat(): [FileInfo | undefined, GoError | undefined] {
    return stat(this.path);
  }
}

const fileInfoType = Object.freeze({ comparable: true });

class NodeFileInfo extends ProviderInterfaceValue implements FileInfo {
  constructor(
    private readonly path: string,
    private readonly information: Stats,
  ) {
    super(fileInfoType);
  }

  IsDir(): boolean {
    return this.information.isDirectory();
  }

  ModTime(): Time {
    return UnixMilli(this.information.mtimeMs);
  }

  Mode(): FileMode {
    let mode = this.information.mode & 0o777;
    if (this.information.isDirectory()) {
      mode |= modeDirectory;
    } else if (this.information.isSymbolicLink()) {
      mode |= modeSymlink;
    }
    return new FileMode(mode >>> 0);
  }

  Name(): gostring {
    return basename(this.path);
  }

  Size(): int64 {
    return this.information.size;
  }

  Sys(): GoInterfaceValue | undefined {
    return undefined;
  }
}

const directoryEntryType = Object.freeze({ comparable: true });

class NodeDirectoryEntry extends ProviderInterfaceValue implements DirEntry {
  constructor(
    private readonly directory: string,
    private readonly entry: Dirent,
  ) {
    super(directoryEntryType);
  }

  Info(): [FileInfo | undefined, GoError | undefined] {
    return lstat(join(this.directory, this.entry.name));
  }

  IsDir(): boolean {
    return this.entry.isDirectory();
  }

  Name(): gostring {
    return this.entry.name;
  }

  Type(): FileMode {
    return directoryEntryMode(this.entry);
  }
}

function directoryEntryMode(entry: Dirent): FileMode {
  if (entry.isDirectory()) {
    return new FileMode(modeDirectory);
  }
  if (entry.isSymbolicLink()) {
    return new FileMode(modeSymlink);
  }
  if (entry.isBlockDevice()) {
    return new FileMode(modeDevice);
  }
  if (entry.isCharacterDevice()) {
    return new FileMode((modeDevice | modeCharDevice) >>> 0);
  }
  if (entry.isFIFO()) {
    return new FileMode(modeNamedPipe);
  }
  if (entry.isSocket()) {
    return new FileMode(modeSocket);
  }
  if (entry.isFile()) {
    return new FileMode(0);
  }
  return new FileMode(modeIrregular);
}

function resolveFileSystemPath(
  root: string,
  name: string,
): string | undefined {
  if (name === ".") {
    return resolve(root);
  }
  if (
    name.length === 0
    || name.startsWith("/")
    || name.startsWith("../")
    || name.startsWith("./")
    || name.includes("/../")
    || name.includes("/./")
    || name.includes("//")
    || name.endsWith("/")
    || name.endsWith("/..")
    || name.endsWith("/.")
    || name.includes("\\")
  ) {
    return undefined;
  }
  const resolvedRoot = resolve(root);
  const path = resolve(resolvedRoot, name);
  if (path !== resolvedRoot && !path.startsWith(`${resolvedRoot}${sep}`)) {
    return undefined;
  }
  return path;
}
