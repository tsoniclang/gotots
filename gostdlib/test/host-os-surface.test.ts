import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import * as flag from "../src/flag.js";
import * as os from "../src/os.js";
import * as exec from "../src/os/exec.js";
import * as signal from "../src/os/signal.js";
import { PathError as FsPathError } from "../src/io/fs.js";
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
    "O_APPEND",
    "O_CREATE",
    "O_TRUNC",
    "O_WRONLY",
    "Open",
    "OpenFile",
    "PathError",
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
    "EAGAIN",
    "EINTR",
    "EINVAL",
    "ENOENT",
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
  assert.deepEqual(staticGoMembers(FsPathError), ["Error", "Unwrap"]);
  assert.deepEqual(staticSupportMembers(FsPathError), [
    "$assign",
    "$copy",
    "$equal",
    "$fromStorage",
    "$hash",
    "$make",
    "$storageOf",
  ]);
  assert.deepEqual(staticMembers(exec.Cmd), ["Output"]);
  assert.deepEqual(staticMembers(flag.FlagSet), ["Bool", "Parse", "String"]);
  assert.deepEqual(
    instanceMembers(syscall.Errno),
    ["Error", "Is", "Temporary", "Timeout"],
  );
  assert.deepEqual(instanceMembers(syscall.Signal), ["Signal", "String"]);
  assert.equal(os.PathError, FsPathError);
  assert.notEqual(os.O_APPEND & os.O_WRONLY, os.O_APPEND);
  assert.notEqual(os.O_CREATE, 0);
  assert.notEqual(os.O_TRUNC, 0);
});

test("public declarations expose only certified provider support modules", () => {
  const supportModules = [
    "@gotots/gostdlib/internal/scalars.js",
    "./internal/runtime/pointer.js",
  ] as const;
  const importCounts = new Map(supportModules.map((module) => [module, 0]));
  for (const path of [
    "../src/flag.d.ts",
    "../src/os.d.ts",
    "../src/os/exec.d.ts",
    "../src/os/signal.d.ts",
    "../src/syscall.d.ts",
  ]) {
    const declaration = readFileSync(new URL(path, import.meta.url), "utf8");
    let publicDeclaration = declaration;
    for (const module of supportModules) {
      importCounts.set(
        module,
        (importCounts.get(module) ?? 0) + declaration.split(module).length - 1,
      );
      publicDeclaration = publicDeclaration.replaceAll(module, "");
    }
    assert.equal(
      publicDeclaration.includes("/internal/"),
      false,
      path,
    );
  }
  for (const [module, count] of importCounts) {
    assert.ok(count > 0, module);
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

function staticGoMembers(value: Function): string[] {
  return staticMembers(value).filter((name: string): boolean => !name.startsWith("$"));
}

function staticSupportMembers(value: Function): string[] {
  return staticMembers(value).filter((name: string): boolean => name.startsWith("$"));
}

function instanceMembers(value: Function): string[] {
  return Object.getOwnPropertyNames(value.prototype)
    .filter((name: string): boolean => name !== "constructor")
    .sort();
}
