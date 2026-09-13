import { Buffer } from "node:buffer";
import { GoString } from "@gotots/runtime/string-value.js";
import { fromHostBytes, fromHostString, toHostBytes } from "../../portable/utf8/codec.js";
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
  int,
  int64,
} from "@gotots/gostdlib/internal/scalars.js";
import {
  hostInteger,
  integerFromHost,
} from "../../host-integer.js";
import { state as ioState } from "../../../io.js";
import type {
  DirEntry,
  FS,
  File as FsFile,
  FileInfo,
} from "../../../io/fs.js";
import { FileMode } from "../../../io/fs.js";
import { Time, UnixMilli } from "../../../time.js";
import { DirectoryFile } from "../../portable/io/filesystem.js";
import { ProviderInterfaceValue } from "../../portable/io/value.js";
import { sliceValues } from "../../runtime/slice.js";
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
  return open(path, 0x242n, new FileMode(0o666), factory);
}

export function open<T extends object>(
  sourcePath: gostring,
  flags: int,
  permissions: FileMode,
  factory: FileFactory<T>,
): [T | undefined, GoError | undefined] {
  const path = sourcePath.text();
  const rawPath = Buffer.from(toHostBytes(sourcePath));
  try {
    const descriptor = openSync(rawPath, hostInteger(flags), permissions.value);
    return [factory(descriptor, path), undefined];
  } catch {
    if (statSync(rawPath, { throwIfNoEntry: false }) === undefined) {
      return [undefined, nodeError("not-exist", "open", path)];
    }
    return [undefined, nodeError("operation", "open", path)];
  }
}

export function makeDirectories(
  sourcePath: gostring,
  permissions: FileMode,
): GoError | undefined {
  const path = sourcePath.text();
  const rawPath = Buffer.from(toHostBytes(sourcePath));
  if (path.length === 0) {
    return nodeError("not-exist", "mkdir", path);
  }
  try {
    mkdirSync(rawPath, {
      recursive: true,
      mode: permissions.value,
    });
    return undefined;
  } catch {
    return nodeError("operation", "mkdir", path);
  }
}

export function remove(sourcePath: gostring): GoError | undefined {
  const path = sourcePath.text();
  const rawPath = Buffer.from(toHostBytes(sourcePath));
  const information = lstatSync(rawPath, { throwIfNoEntry: false });
  if (information === undefined) {
    return nodeError("not-exist", "remove", path);
  }
  try {
    if (information.isDirectory()) {
      rmdirSync(rawPath);
    } else {
      rmSync(rawPath);
    }
    return undefined;
  } catch {
    return nodeError("operation", "remove", path);
  }
}

export function removeAll(sourcePath: gostring): GoError | undefined {
  const path = sourcePath.text();
  const rawPath = Buffer.from(toHostBytes(sourcePath));
  try {
    rmSync(rawPath, {
      force: true,
      recursive: true,
    });
    return undefined;
  } catch {
    return nodeError("operation", "removeall", path);
  }
}

export function changeTimes(
  sourcePath: gostring,
  accessTime: Time,
  modificationTime: Time,
): GoError | undefined {
  const path = sourcePath.text();
  const rawPath = Buffer.from(toHostBytes(sourcePath));
  try {
    utimesSync(
      rawPath,
      hostInteger(accessTime.UnixMilli()) / 1000,
      hostInteger(modificationTime.UnixMilli()) / 1000,
    );
    return undefined;
  } catch {
    if (statSync(rawPath, { throwIfNoEntry: false }) === undefined) {
      return nodeError("not-exist", "chtimes", path);
    }
    return nodeError("operation", "chtimes", path);
  }
}

export function stat(
  sourcePath: gostring,
): [FileInfo | undefined, GoError | undefined] {
  const path = sourcePath.text();
  const rawPath = Buffer.from(toHostBytes(sourcePath));
  try {
    const information = statSync(rawPath, { throwIfNoEntry: false });
    if (information === undefined) {
      return [undefined, nodeError("not-exist", "stat", path)];
    }
    return [new NodeFileInfo(path, information), undefined];
  } catch {
    return [undefined, nodeError("operation", "stat", path)];
  }
}

export function lstat(
  sourcePath: gostring,
): [FileInfo | undefined, GoError | undefined] {
  const path = sourcePath.text();
  const rawPath = Buffer.from(toHostBytes(sourcePath));
  try {
    const information = lstatSync(rawPath, { throwIfNoEntry: false });
    if (information === undefined) {
      return [undefined, nodeError("not-exist", "lstat", path)];
    }
    return [new NodeFileInfo(path, information), undefined];
  } catch {
    return [undefined, nodeError("operation", "lstat", path)];
  }
}

export function readDirectory(
  sourcePath: gostring,
): [RuntimeSlice<DirEntry | undefined>, GoError | undefined] {
  const path = sourcePath.text();
  const rawPath = Buffer.from(toHostBytes(sourcePath));
  try {
    const information = statSync(rawPath, { throwIfNoEntry: false });
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
    const entries = readdirSync(rawPath, { withFileTypes: true, encoding: "buffer" });
    entries.sort((left, right): number => Buffer.compare(
      left.name,
      right.name,
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
  return new NodeDirectoryFS(root.text());
}

const directoryFileSystemType = Object.freeze({ comparable: true });

class NodeDirectoryFS extends ProviderInterfaceValue implements FS {
  constructor(private readonly root: string) {
    super(directoryFileSystemType);
  }

  Open(name: gostring): [FsFile | undefined, GoError | undefined] {
    const path = resolveFileSystemPath(this.root, name.text());
    if (path === undefined) {
      return [undefined, nodeError("invalid", "open", name.text())];
    }
    const [file, error] = open(
      GoString.fromText(path),
      0n,
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

class NodeFileSystemFile extends DirectoryFile implements FsFile {
  private directoryOffset = 0;

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
    return stat(GoString.fromText(this.path));
  }

  ReadDir(count: int): [
    RuntimeSlice<DirEntry | undefined>,
    GoError | undefined,
  ] {
    const [entries, failure] = readDirectory(GoString.fromText(this.path));
    if (failure !== undefined) {
      return [RuntimeSlice.nil<DirEntry | undefined>(), failure];
    }
    const values = sliceValues(entries);
    const start = this.directoryOffset;
    if (count <= 0n) {
      this.directoryOffset = values.length;
      return [RuntimeSlice.literal(values.slice(start)), undefined];
    }
    if (start >= values.length) {
      return [RuntimeSlice.literal<DirEntry | undefined>([]), ioState.EOF];
    }
    const end = Math.min(values.length, start + hostInteger(count));
    this.directoryOffset = end;
    return [RuntimeSlice.literal(values.slice(start, end)), undefined];
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
    return UnixMilli(integerFromHost(this.information.mtimeMs));
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
    return GoString.fromText(basename(this.path));
  }

  Size(): int64 {
    return integerFromHost(this.information.size);
  }

  Sys(): GoInterfaceValue | undefined {
    return undefined;
  }
}

const directoryEntryType = Object.freeze({ comparable: true });

class NodeDirectoryEntry extends ProviderInterfaceValue implements DirEntry {
  constructor(
    private readonly directory: string,
    private readonly entry: Dirent<Buffer>,
  ) {
    super(directoryEntryType);
  }

  Info(): [FileInfo | undefined, GoError | undefined] {
    return lstat(GoString.fromText(join(this.directory, fromHostBytes(this.entry.name).text())));
  }

  IsDir(): boolean {
    return this.entry.isDirectory();
  }

  Name(): gostring {
    return fromHostBytes(this.entry.name);
  }

  Type(): FileMode {
    return directoryEntryMode(this.entry);
  }
}

function directoryEntryMode(entry: Dirent<Buffer>): FileMode {
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
    return resolve(fromHostString(process.cwd()).text(), root);
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
  const resolvedRoot = resolve(fromHostString(process.cwd()).text(), root);
  const path = resolve(resolvedRoot, name);
  const descendantPrefix = resolvedRoot.endsWith(sep)
    ? resolvedRoot
    : `${resolvedRoot}${sep}`;
  if (path !== resolvedRoot && !path.startsWith(descendantPrefix)) {
    return undefined;
  }
  return path;
}
