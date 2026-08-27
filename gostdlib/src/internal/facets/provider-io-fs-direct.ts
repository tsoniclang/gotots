import type { GoInterfaceValue } from "@gotots/runtime/interface-value.js";
import { GoPanic, type GoRecovery } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  bool,
  gostring,
  int,
  int64,
  uint8,
} from "@gotots/gostdlib/internal/scalars.js";

import { New } from "../../errors.js";
import { PathError, type FileMode } from "../../io/fs.js";
import { Join as JoinPath } from "../../path.js";
import type { Time } from "../../time.js";
import { hostInteger } from "../host-integer.js";
import { DirectoryFile } from "../portable/io/filesystem.js";
import { ProviderInterfaceValue } from "../portable/io/value.js";
import { goInterfaceEqual } from "../runtime/interface.js";
import { byteSlice, sliceValues } from "../runtime/slice.js";
import type { ProviderErrorInterface } from "./provider-error.js";
import type {
  InterfaceContract,
  InterfaceView,
} from "./provider-support.js";

export type { ProviderErrorInterface } from "./provider-error.js";

export interface ProviderFileInfo extends GoInterfaceValue {
  IsDir(recovery?: GoRecovery): bool;
  ModTime(recovery?: GoRecovery): Time;
  Mode(recovery?: GoRecovery): FileMode;
  Name(recovery?: GoRecovery): gostring;
  Size(recovery?: GoRecovery): int64;
  Sys(recovery?: GoRecovery): GoInterfaceValue | undefined;
}

export interface ProviderDirEntry extends GoInterfaceValue {
  Info(
    recovery?: GoRecovery,
  ): [ProviderFileInfo | undefined, ProviderErrorInterface | undefined];
  IsDir(recovery?: GoRecovery): bool;
  Name(recovery?: GoRecovery): gostring;
  Type(recovery?: GoRecovery): FileMode;
}

export interface ProviderFile extends GoInterfaceValue {
  Close(recovery?: GoRecovery): ProviderErrorInterface | undefined;
  Read(
    buffer: RuntimeSlice<uint8>,
    recovery?: GoRecovery,
  ): [int, ProviderErrorInterface | undefined];
  Stat(
    recovery?: GoRecovery,
  ): [ProviderFileInfo | undefined, ProviderErrorInterface | undefined];
}

export interface ProviderFS extends GoInterfaceValue {
  Open(
    name: gostring,
    recovery?: GoRecovery,
  ): [ProviderFile | undefined, ProviderErrorInterface | undefined];
}

export interface ProviderReadFileFS extends ProviderFS {
  ReadFile(
    name: gostring,
    recovery?: GoRecovery,
  ): [RuntimeSlice<uint8>, ProviderErrorInterface | undefined];
}

export interface ProviderReadDirFS extends ProviderFS {
  ReadDir(
    name: gostring,
    recovery?: GoRecovery,
  ): [
    RuntimeSlice<ProviderDirEntry | undefined>,
    ProviderErrorInterface | undefined,
  ];
}

export interface ProviderReadDirFile extends ProviderFile {
  ReadDir(
    count: int,
    recovery?: GoRecovery,
  ): [
    RuntimeSlice<ProviderDirEntry | undefined>,
    ProviderErrorInterface | undefined,
  ];
}

export interface ProviderStatFS extends ProviderFS {
  Stat(
    name: gostring,
    recovery?: GoRecovery,
  ): [ProviderFileInfo | undefined, ProviderErrorInterface | undefined];
}

type ProviderWalkDirFunc = ((
  path: gostring,
  entry: ProviderDirEntry | undefined,
  failure: ProviderErrorInterface | undefined,
  recovery?: GoRecovery,
) => ProviderErrorInterface | undefined) | undefined;

const directInfoDirEntryType = Object.freeze({ comparable: true });

class DirectInfoDirEntry extends ProviderInterfaceValue
  implements ProviderDirEntry {
  override readonly $go$methods: ReadonlySet<object>;

  constructor(
    private readonly information: ProviderFileInfo,
    contract: readonly object[],
  ) {
    super(directInfoDirEntryType);
    this.$go$methods = new Set(contract);
  }

  Info(): [ProviderFileInfo, undefined] {
    return [this.information, undefined];
  }

  IsDir(): bool {
    return this.information.IsDir();
  }

  Name(): gostring {
    return this.information.Name();
  }

  Type(): FileMode {
    return this.information.Mode().Type();
  }
}

export function IoFsFileInfoToDirEntryDirect(
  information: ProviderFileInfo | undefined,
  dirEntryContract: InterfaceContract,
): ProviderDirEntry | undefined {
  return information === undefined
    ? undefined
    : new DirectInfoDirEntry(information, dirEntryContract);
}

export function IoFsReadFileDirect(
  fileSystem: ProviderFS | undefined,
  name: gostring,
  eof: ProviderErrorInterface | undefined,
  asReadFileFS: InterfaceView<ProviderFS, ProviderReadFileFS>,
): [RuntimeSlice<uint8>, ProviderErrorInterface | undefined] {
  const selected = requireFileSystem(fileSystem);
  const readFileSystem = asReadFileFS(selected);
  if (readFileSystem !== undefined) {
    return readFileSystem.ReadFile(name);
  }
  const [file, openFailure] = selected.Open(name);
  if (openFailure !== undefined || file === undefined) {
    return [RuntimeSlice.nil<uint8>(), openFailure];
  }
  try {
    const values: number[] = [];
    for (;;) {
      const buffer = RuntimeSlice.make<uint8>(32 * 1024, 32 * 1024, 0);
      const [count, readFailure] = file.Read(buffer);
      const hostCount = hostInteger(count);
      for (let index = 0; index < hostCount; index += 1) {
        values.push(buffer.get(index));
      }
      if (readFailure !== undefined) {
        return [
          byteSlice(values),
          goInterfaceEqual(readFailure, eof) ? undefined : readFailure,
        ];
      }
    }
  } finally {
    file.Close();
  }
}

export function IoFsStatDirect(
  fileSystem: ProviderFS | undefined,
  name: gostring,
  asStatFS: InterfaceView<ProviderFS, ProviderStatFS>,
): [ProviderFileInfo | undefined, ProviderErrorInterface | undefined] {
  const selected = requireFileSystem(fileSystem);
  const statFileSystem = asStatFS(selected);
  if (statFileSystem !== undefined) {
    return statFileSystem.Stat(name);
  }
  const [file, openFailure] = selected.Open(name);
  if (openFailure !== undefined || file === undefined) {
    return [undefined, openFailure];
  }
  try {
    return file.Stat();
  } finally {
    file.Close();
  }
}

export function IoFsReadDirDirect(
  fileSystem: ProviderFS | undefined,
  name: gostring,
  asReadDirFS: InterfaceView<ProviderFS, ProviderReadDirFS>,
  asReadDirFile: InterfaceView<ProviderFile, ProviderReadDirFile>,
): [
  RuntimeSlice<ProviderDirEntry | undefined>,
  ProviderErrorInterface | undefined,
] {
  const selected = requireFileSystem(fileSystem);
  const readDirFileSystem = asReadDirFS(selected);
  if (readDirFileSystem !== undefined) {
    return readDirFileSystem.ReadDir(name);
  }
  const [file, openFailure] = selected.Open(name);
  if (openFailure !== undefined || file === undefined) {
    return [RuntimeSlice.nil<ProviderDirEntry | undefined>(), openFailure];
  }
  try {
    const readDirFile = file instanceof DirectoryFile
      ? file
      : asReadDirFile(file);
    if (readDirFile === undefined) {
      return [
        RuntimeSlice.nil<ProviderDirEntry | undefined>(),
        new PathError("readdir", name, New("not implemented")),
      ];
    }
    const [entries, readFailure] = readDirFile.ReadDir(-1n);
    return [
      RuntimeSlice.literal(sortDirectoryEntries(sliceValues(entries))),
      readFailure,
    ];
  } finally {
    file.Close();
  }
}

export function IoFsWalkDirDirect(
  fileSystem: ProviderFS | undefined,
  root: gostring,
  visit: ProviderWalkDirFunc,
  skipAll: ProviderErrorInterface | undefined,
  skipDir: ProviderErrorInterface | undefined,
  asReadDirFS: InterfaceView<ProviderFS, ProviderReadDirFS>,
  asReadDirFile: InterfaceView<ProviderFile, ProviderReadDirFile>,
  asStatFS: InterfaceView<ProviderFS, ProviderStatFS>,
  dirEntryContract: InterfaceContract,
): ProviderErrorInterface | undefined {
  const [information, statFailure] = IoFsStatDirect(
    fileSystem,
    root,
    asStatFS,
  );
  let failure: ProviderErrorInterface | undefined;
  if (statFailure !== undefined) {
    failure = invokeWalkDir(visit, root, undefined, statFailure);
  } else {
    const entry = IoFsFileInfoToDirEntryDirect(
      information,
      dirEntryContract,
    );
    failure = walk(
      fileSystem,
      root,
      entry,
      visit,
      skipDir,
      asReadDirFS,
      asReadDirFile,
    );
  }
  return goInterfaceEqual(failure, skipDir) || goInterfaceEqual(failure, skipAll)
    ? undefined
    : failure;
}

function walk(
  fileSystem: ProviderFS | undefined,
  path: gostring,
  entry: ProviderDirEntry | undefined,
  visit: ProviderWalkDirFunc,
  skipDir: ProviderErrorInterface | undefined,
  asReadDirFS: InterfaceView<ProviderFS, ProviderReadDirFS>,
  asReadDirFile: InterfaceView<ProviderFile, ProviderReadDirFile>,
): ProviderErrorInterface | undefined {
  let failure = invokeWalkDir(visit, path, entry, undefined);
  if (entry === undefined) {
    if (failure !== undefined) {
      return failure;
    }
    return GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  const isDirectory = entry.IsDir();
  if (failure !== undefined || !isDirectory) {
    return goInterfaceEqual(failure, skipDir) && isDirectory
      ? undefined
      : failure;
  }
  const [entries, readFailure] = IoFsReadDirDirect(
    fileSystem,
    path,
    asReadDirFS,
    asReadDirFile,
  );
  if (readFailure !== undefined) {
    failure = invokeWalkDir(visit, path, entry, readFailure);
    if (failure !== undefined) {
      return goInterfaceEqual(failure, skipDir) ? undefined : failure;
    }
  }
  for (const child of sliceValues(entries)) {
    if (child === undefined) {
      return GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
    }
    const childFailure = walk(
      fileSystem,
      JoinPath(RuntimeSlice.literal([path, child.Name()])),
      child,
      visit,
      skipDir,
      asReadDirFS,
      asReadDirFile,
    );
    if (goInterfaceEqual(childFailure, skipDir)) {
      break;
    }
    if (childFailure !== undefined) {
      return childFailure;
    }
  }
  return undefined;
}

function invokeWalkDir(
  visit: ProviderWalkDirFunc,
  path: gostring,
  entry: ProviderDirEntry | undefined,
  failure: ProviderErrorInterface | undefined,
): ProviderErrorInterface | undefined {
  if (visit === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return visit(path, entry, failure);
}

function requireFileSystem(fileSystem: ProviderFS | undefined): ProviderFS {
  if (fileSystem === undefined) {
    GoPanic.raiseRuntime("invalid memory address or nil pointer dereference");
  }
  return fileSystem;
}

function sortDirectoryEntries(
  entries: (ProviderDirEntry | undefined)[],
): (ProviderDirEntry | undefined)[] {
  const named = entries.map((entry) => ({
    entry,
    name: entry === undefined ? "" : entry.Name(),
  }));
  named.sort((left, right) =>
    left.name < right.name ? -1 : left.name > right.name ? 1 : 0);
  return named.map(({ entry }) => entry);
}
