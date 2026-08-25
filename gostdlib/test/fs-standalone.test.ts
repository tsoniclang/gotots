import assert from "node:assert/strict";
import test from "node:test";

import type {
  GoError,
  GoInterfaceValue,
} from "@gotots/runtime/interface-value.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  bool,
  int,
  int64,
  uint8,
} from "../src/internal/scalars.js";
import { integerFromHost } from "../src/internal/host-integer.js";

import { Is } from "../src/errors.js";
import { DirectoryFile } from "../src/internal/portable/io/filesystem.js";
import { ProviderInterfaceValue } from "../src/internal/portable/io/value.js";
import {
  bytes,
  sliceValues,
} from "../src/internal/runtime/slice.js";
import {
  FileInfoToDirEntry,
  FileMode,
  ModeDir,
  PathError,
  ReadDir,
  ReadFile,
  Stat,
  WalkDir,
  type DirEntry,
  type FS,
  type File,
  type FileInfo,
} from "../src/io/fs.js";
import { state as ioState } from "../src/io.js";
import { Time } from "../src/time.js";

const infoType = Object.freeze({ comparable: true });
const entryType = Object.freeze({ comparable: true });
const fileType = Object.freeze({ comparable: true });
const fileSystemType = Object.freeze({ comparable: true });

class TestInfo extends ProviderInterfaceValue implements FileInfo {
  constructor(
    private readonly name: string,
    private readonly size: int64,
    private readonly mode: FileMode,
  ) {
    super(infoType);
  }

  IsDir(): bool {
    return this.mode.IsDir();
  }

  ModTime(): Time {
    return new Time();
  }

  Mode(): FileMode {
    return this.mode;
  }

  Name(): string {
    return this.name;
  }

  Size(): int64 {
    return this.size;
  }

  Sys(): GoInterfaceValue | undefined {
    return undefined;
  }
}

class TestEntry extends ProviderInterfaceValue implements DirEntry {
  constructor(private readonly information: FileInfo) {
    super(entryType);
  }

  Info(): [FileInfo, undefined] {
    return [this.information, undefined];
  }

  IsDir(): bool {
    return this.information.IsDir();
  }

  Name(): string {
    return this.information.Name();
  }

  Type(): FileMode {
    return this.information.Mode().Type();
  }
}

class TestFile extends ProviderInterfaceValue implements File {
  private offset = 0;

  constructor(
    private readonly information: FileInfo,
    private readonly data: Uint8Array,
  ) {
    super(fileType);
  }

  Close(): GoError | undefined {
    return undefined;
  }

  Read(destination: RuntimeSlice<uint8>): [int, GoError | undefined] {
    if (this.offset >= this.data.length) {
      return [0n, ioState.EOF];
    }
    const count = Math.min(destination.length, this.data.length - this.offset);
    for (let index = 0; index < count; index += 1) {
      destination.set(index, this.data[this.offset + index] ?? 0);
    }
    this.offset += count;
    return [
      integerFromHost(count),
      this.offset === this.data.length ? ioState.EOF : undefined,
    ];
  }

  Stat(): [FileInfo, undefined] {
    return [this.information, undefined];
  }
}

class TestDirectory extends DirectoryFile {
  constructor(
    private readonly information: FileInfo,
    private readonly entries: readonly DirEntry[],
  ) {
    super();
  }

  Close(): GoError | undefined {
    return undefined;
  }

  Read(): [int, GoError] {
    return [0n, ioState.EOF];
  }

  Stat(): [FileInfo, undefined] {
    return [this.information, undefined];
  }

  ReadDir(): [RuntimeSlice<DirEntry | undefined>, undefined] {
    return [RuntimeSlice.literal([...this.entries]), undefined];
  }
}

class TestFS extends ProviderInterfaceValue implements FS {
  constructor() {
    super(fileSystemType);
  }

  Open(name: string): [File | undefined, GoError | undefined] {
    const root = new TestInfo(".", 0n, ModeDir);
    const directory = new TestInfo("dir", 0n, ModeDir);
    const first = new TestInfo("a.txt", 5n, new FileMode(0));
    const nested = new TestInfo("z.txt", 1n, new FileMode(0));
    switch (name) {
      case ".":
        return [
          new TestDirectory(root, [new TestEntry(directory), new TestEntry(first)]),
          undefined,
        ];
      case "a.txt":
        return [new TestFile(first, new TextEncoder().encode("hello")), undefined];
      case "dir":
        return [new TestDirectory(directory, [new TestEntry(nested)]), undefined];
      case "dir/z.txt":
        return [new TestFile(nested, Uint8Array.of(90)), undefined];
      default:
        return [undefined, new PathError("open", name, ioState.EOF)];
    }
  }
}

test("filesystem functions read, stat, sort, and walk", async () => {
  const fileSystem = new TestFS();
  const [content, readFailure] = ReadFile(fileSystem, "a.txt");
  assert.equal(readFailure, undefined);
  assert.equal(new TextDecoder().decode(bytes(content)), "hello");

  const [information, statFailure] = Stat(fileSystem, "a.txt");
  assert.equal(statFailure, undefined);
  assert.equal(information?.Size(), 5n);
  assert.equal(FileInfoToDirEntry(information)?.Name(), "a.txt");

  const [entries, directoryFailure] = ReadDir(fileSystem, ".");
  assert.equal(directoryFailure, undefined);
  assert.deepEqual(
    sliceValues(entries).map((entry) => entry?.Name()),
    ["a.txt", "dir"],
  );

  const visited: string[] = [];
  const walkFailure = WalkDir(fileSystem, ".", (path) => {
    visited.push(path);
    return undefined;
  });
  assert.equal(walkFailure, undefined);
  assert.deepEqual(visited, [".", "a.txt", "dir", "dir/z.txt"]);
});

test("FileMode and PathError preserve selected behavior", () => {
  assert.equal(ModeDir.IsDir(), true);
  assert.equal(new FileMode(0o644).IsRegular(), true);
  const pathFailure = new PathError("open", "missing", ioState.EOF);
  assert.equal(pathFailure.Error(), "open missing: EOF");
  assert.equal(Is(pathFailure, ioState.EOF), true);
});
