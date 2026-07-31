import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { once } from "node:events";
import test from "node:test";
import {
  Executable,
  FindProcess,
  Getenv,
  Getpid,
  Getwd,
  Process,
  UserCacheDir,
} from "../src/os.js";
import { SIGTERM } from "../src/syscall.js";

test("os environment and process facts come from the Node host", () => {
  const key = "GOTOTS_GOSTDLIB_TEST_VALUE";
  const previous = process.env[key];
  process.env[key] = "selected";
  try {
    assert.equal(Getenv(key), "selected");
    assert.equal(Getenv(`${key}_MISSING`), "");
    assert.deepEqual(Executable(), [process.execPath, undefined]);
    assert.deepEqual(Getwd(), [process.cwd(), undefined]);
    assert.equal(Getpid(), process.pid);
    assert.equal(UserCacheDir()[1], undefined);
  } finally {
    if (previous === undefined) {
      delete process.env[key];
    } else {
      process.env[key] = previous;
    }
  }
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
  const [selected, findError] = FindProcess(child.pid);
  assert.equal(findError, undefined);
  assert.ok(selected !== undefined);
  assert.equal(Process.Signal(selected, SIGTERM), undefined);
  const [code, signal] = await once(child, "exit");
  assert.equal(code, null);
  assert.equal(signal, "SIGTERM");
});
