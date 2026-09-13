import { textValues } from "./text.js";
import { GoString } from "@gotots/runtime/string-value.js";
import { fromHostString, toHostString } from "../src/internal/portable/utf8/codec.js";
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { once } from "node:events";
import { parse, relative, sep } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";
import { integerFromHost } from "../src/internal/host-integer.js";
import { OsProcessOperations } from "../src/internal/facets/named-os.js";
import {
  DirFS,
  Executable,
  FindProcess,
  Getenv,
  Getpid,
  Getwd,
  Process,
  UserCacheDir,
  state,
} from "../src/os.js";
import { sliceValues } from "../src/internal/runtime/slice.js";
import { SIGTERM } from "../src/syscall.js";

test("os environment and process facts come from the Node host", () => {
  const key = "GOTOTS_GOSTDLIB_TEST_VALUE";
  const previous = process.env[key];
  process.env[key] = "selected";
  try {
    assert.equal((Getenv(GoString.fromText(key)))?.text(), "selected");
    assert.equal((Getenv(GoString.fromText(`${key}_MISSING`)))?.text(), "");
    assert.deepEqual(textValues(Executable()), [process.execPath, undefined]);
    assert.deepEqual(textValues(Getwd()), [process.cwd(), undefined]);
    assert.equal(Getpid(), integerFromHost(process.pid));
    assert.equal(UserCacheDir()[1], undefined);
  } finally {
    if (previous === undefined) {
      delete process.env[key];
    } else {
      process.env[key] = previous;
    }
  }
});

test("os.Args uses the generated entry as Go argument zero", () => {
  assert.deepEqual(sliceValues(state.Args).map(toHostString), process.argv.slice(1));
});

test("Process value operations preserve Go struct assignment", () => {
  const source = new Process(17n);
  const target = new Process(3n);
  OsProcessOperations.$assign(target, source);
  assert.equal(target.Pid, 17n);
  const copied = OsProcessOperations.$copy(source);
  assert.notEqual(copied, source);
  assert.equal(copied.Pid, 17n);
});

test("DirFS accepts descendants of a filesystem root", () => {
  const sourcePath = fileURLToPath(import.meta.url);
  const root = parse(sourcePath).root;
  const name = relative(root, sourcePath).split(sep).join("/");
  const fileSystem = DirFS(fromHostString(root));
  assert.ok(fileSystem !== undefined);
  const [file, failure] = fileSystem.Open(fromHostString(name));
  assert.equal(failure, undefined);
  assert.ok(file !== undefined);
  assert.equal(file.Close(), undefined);
});

test("Process.Signal targets only an isolated child process", async () => {
  const child = spawn(
    process.execPath,
    ["-e", "setInterval(() => {}, 1000)"],
    {
      stdio: "ignore",
    },
  );
  await once(child, "spawn");
  assert.ok(child.pid !== undefined);
  const [selected, findError] = FindProcess(integerFromHost(child.pid));
  assert.equal(findError, undefined);
  assert.ok(selected !== undefined);
  assert.equal(Process.Signal(selected, SIGTERM), undefined);
  const [code, signal] = await once(child, "exit");
  assert.equal(code, null);
  assert.equal(signal, "SIGTERM");
});
