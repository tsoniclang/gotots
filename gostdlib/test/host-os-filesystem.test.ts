import { GoString } from "@gotots/runtime/string-value.js";
import { fromHostString, toHostString } from "../src/internal/portable/utf8/codec.js";
import assert from "node:assert/strict";
import {
  mkdtempSync,
  readFileSync,
  rmSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import { hostInteger } from "../src/internal/host-integer.js";
import { FileMode } from "../src/io/fs.js";
import { state as ioState } from "../src/io.js";
import {
  Chtimes,
  Create,
  DirFS,
  File,
  IsNotExist,
  MkdirAll,
  Open,
  OpenFile,
  Remove,
  RemoveAll,
  Stat,
  TempDir,
} from "../src/os.js";
import { sliceValues } from "../src/internal/runtime/slice.js";
import { UnixMilli } from "../src/time.js";

test("os filesystem operations preserve Go result tuples and file state", () => {
  const root = mkdtempSync(join(tmpdir(), "gotots-os-"));
  try {
    const nested = join(root, "nested", "directory");
    assert.equal(MkdirAll(fromHostString(nested), new FileMode(0o755)), undefined);

    const path = join(nested, "sample.txt");
    const [created, createError] = Create(fromHostString(path));
    assert.equal(createError, undefined);
    assert.ok(created !== undefined);
    assert.deepEqual(File.WriteString(created, GoString.fromText("hello")), [5n, undefined]);
    assert.equal(File.Close(created), undefined);
    assert.notEqual(File.Close(created), undefined);
    assert.equal(readFileSync(path, "utf8"), "hello");

    const [opened, openError] = OpenFile(fromHostString(path), 0n, new FileMode(0));
    assert.equal(openError, undefined);
    assert.ok(opened !== undefined);
    const buffer = RuntimeSlice.make<number>(8, null, 0);
    const [count, readError] = File.Read(opened, buffer);
    assert.equal(readError, undefined);
    assert.equal(count, 5n);
    assert.deepEqual(sliceValues(buffer).slice(0, hostInteger(count)), [
      104,
      101,
      108,
      108,
      111,
    ]);
    assert.deepEqual(File.Read(opened, buffer), [0n, ioState.EOF]);
    assert.equal(File.Close(opened), undefined);

    const [openedSimply, simpleOpenError] = Open(fromHostString(path));
    assert.equal(simpleOpenError, undefined);
    assert.ok(openedSimply !== undefined);
    assert.deepEqual(File.Read(openedSimply, buffer), [5n, undefined]);
    assert.equal(File.Close(openedSimply), undefined);
    assert.equal(IsNotExist(Open(fromHostString(join(root, "missing")))[1]), true);

    const [information, statError] = Stat(fromHostString(path));
    assert.equal(statError, undefined);
    assert.equal((information?.Name())?.text(), "sample.txt");
    assert.equal(information?.Size(), 5n);
    assert.equal(information?.IsDir(), false);

    const timestamp = UnixMilli(1_700_000_000_000n);
    assert.equal(Chtimes(fromHostString(path), timestamp, timestamp), undefined);

    const fileSystem = DirFS(fromHostString(root));
    assert.ok(fileSystem !== undefined);
    const [fsFile, fsError] = fileSystem.Open(GoString.fromText("nested/directory/sample.txt"));
    assert.equal(fsError, undefined);
    assert.ok(fsFile !== undefined);
    assert.equal(fsFile.Stat()[0]?.Size(), 5n);
    assert.equal(fsFile.Close(), undefined);
    assert.notEqual(fileSystem.Open(GoString.fromText("./nested"))[1], undefined);

    assert.equal(Remove(fromHostString(path)), undefined);
    const missingError = Remove(fromHostString(path));
    assert.equal(IsNotExist(missingError), true);
    assert.equal(RemoveAll(fromHostString(join(root, "nested"))), undefined);
    assert.equal(toHostString(TempDir()), tmpdir());
  } finally {
    rmSync(root, {
      force: true,
      recursive: true,
    });
  }
});

test("os File.WriteString preserves exact Go string bytes", () => {
  const root = mkdtempSync(join(tmpdir(), "gotots-os-bytes-"));
  try {
    const path = join(root, "bytes.bin");
    const [created, createError] = Create(fromHostString(path));
    assert.equal(createError, undefined);
    assert.ok(created !== undefined);

    const expected = Buffer.from([
      0xe2, 0x80, 0x9c,
      0xc3, 0xa9,
      0xf0, 0x9f, 0x99, 0x82,
      0x00, 0xff,
    ]);
    const goString = expected.toString("latin1");
    assert.deepEqual(File.WriteString(created, GoString.fromText(goString)), [11n, undefined]);
    assert.equal(File.Close(created), undefined);
    assert.deepEqual(readFileSync(path), expected);
  } finally {
    rmSync(root, {
      force: true,
      recursive: true,
    });
  }
});
