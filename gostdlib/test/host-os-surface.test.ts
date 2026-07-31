import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import * as flag from "../src/flag.js";
import * as os from "../src/os.js";
import * as exec from "../src/os/exec.js";
import * as signal from "../src/os/signal.js";
import * as syscall from "../src/syscall.js";

test("host OS modules expose only selected clean Go names", () => {
  assert.deepEqual(Object.keys(os).sort(), [
    "Chtimes",
    "Create",
    "DirFS",
    "Executable",
    "Exit",
    "File",
    "FindProcess",
    "Getenv",
    "Getpid",
    "Getwd",
    "IsNotExist",
    "Lstat",
    "MkdirAll",
    "OpenFile",
    "Process",
    "ProcessState",
    "ReadDir",
    "Remove",
    "RemoveAll",
    "Stat",
    "TempDir",
    "UserCacheDir",
    "state",
  ]);
  assert.deepEqual(Object.keys(exec).sort(), ["Cmd", "Command"]);
  assert.deepEqual(Object.keys(signal), ["NotifyContext"]);
  assert.deepEqual(Object.keys(syscall).sort(), [
    "Credential",
    "EINTR",
    "ENOTDIR",
    "EPERM",
    "Errno",
    "SIGINT",
    "SIGTERM",
    "Signal",
    "SysProcAttr",
    "SysProcIDMap",
  ]);
  assert.deepEqual(Object.keys(flag).sort(), [
    "ContinueOnError",
    "ErrorHandling",
    "FlagSet",
    "NewFlagSet",
  ]);

  assert.deepEqual(staticMembers(os.File), [
    "Close",
    "Fd",
    "Read",
    "Write",
    "WriteString",
  ]);
  assert.deepEqual(staticMembers(os.Process), ["Signal"]);
  assert.deepEqual(staticMembers(exec.Cmd), ["Output"]);
  assert.deepEqual(staticMembers(flag.FlagSet), ["Bool", "Parse", "String"]);
  assert.deepEqual(instanceMembers(syscall.Errno), ["Error"]);
  assert.deepEqual(instanceMembers(syscall.Signal), ["Signal", "String"]);
});

test("public declarations do not expose internal Node ownership", () => {
  for (const path of [
    "../src/flag.d.ts",
    "../src/os.d.ts",
    "../src/os/exec.d.ts",
    "../src/os/signal.d.ts",
    "../src/syscall.d.ts",
  ]) {
    const declaration = readFileSync(new URL(path, import.meta.url), "utf8");
    assert.equal(declaration.includes("/internal/"), false, path);
  }
});

function staticMembers(value: Function): string[] {
  return Object.getOwnPropertyNames(value)
    .filter((name: string): boolean => (
      name !== "length"
      && name !== "name"
      && name !== "prototype"
    ))
    .sort();
}

function instanceMembers(value: Function): string[] {
  return Object.getOwnPropertyNames(value.prototype)
    .filter((name: string): boolean => name !== "constructor")
    .sort();
}
