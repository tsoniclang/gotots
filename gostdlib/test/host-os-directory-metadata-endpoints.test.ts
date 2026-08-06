import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import {
  mkdirSync,
  mkdtempSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { Is } from "../src/errors.js";
import { ModeSymlink } from "../src/io/fs.js";
import {
  Lstat,
  ReadDir,
  Stat,
} from "../src/os.js";
import { errnoError } from "../src/internal/portable/syscall/errno.js";
import { sliceValues } from "../src/internal/runtime/slice.js";
import { ENOTDIR } from "../src/syscall.js";

test("os Lstat and ReadDir agree with Go on deterministic metadata and errors", () => {
  const root = mkdtempSync(join(tmpdir(), "gotots-os-endpoints-"));
  const source = join(root, "main.go");
  const directory = join(root, "fixture");
  try {
    mkdirSync(directory);
    writeFileSync(join(directory, "zeta.txt"), "z");
    writeFileSync(join(directory, "alpha.txt"), "a");
    symlinkSync("alpha.txt", join(directory, "link.txt"));
    writeFileSync(source, goProgram);

    const goResult = spawnSync(
      "go",
      ["run", source, directory],
      { encoding: "utf8" },
    );
    assert.equal(goResult.status, 0, goResult.stderr);
    assert.equal(providerResult(directory), goResult.stdout.trim());
  } finally {
    rmSync(root, {
      force: true,
      recursive: true,
    });
  }
});

function providerResult(directory: string): string {
  const link = join(directory, "link.txt");
  const [linkInformation, linkError] = Lstat(link);
  const [targetInformation, targetError] = Stat(link);
  const [entries, readError] = ReadDir(directory);
  const [notDirectoryEntries, notDirectoryError] = ReadDir(
    join(directory, "alpha.txt"),
  );

  assert.equal(linkError, undefined);
  assert.equal(targetError, undefined);
  assert.equal(readError, undefined);
  assert.ok(linkInformation !== undefined);
  assert.ok(targetInformation !== undefined);

  return [
    linkInformation.Mode().Type().value === ModeSymlink.value,
    targetInformation.Mode().Type().value === ModeSymlink.value,
    sliceValues(entries).map((entry) => entry?.Name()).join(","),
    notDirectoryEntries.isNil(),
    Is(notDirectoryError, errnoError(ENOTDIR)),
  ].join("|");
}

const goProgram = `
package main

import (
  "errors"
  "fmt"
  "os"
  "path/filepath"
  "strings"
  "syscall"
)

func main() {
  directory := os.Args[1]
  link := filepath.Join(directory, "link.txt")
  linkInformation, linkError := os.Lstat(link)
  if linkError != nil {
    panic(linkError)
  }
  targetInformation, targetError := os.Stat(link)
  if targetError != nil {
    panic(targetError)
  }
  entries, readError := os.ReadDir(directory)
  if readError != nil {
    panic(readError)
  }
  names := make([]string, len(entries))
  for index, entry := range entries {
    names[index] = entry.Name()
  }
  failedEntries, notDirectoryError := os.ReadDir(filepath.Join(directory, "alpha.txt"))
  fmt.Printf(
    "%t|%t|%s|%t|%t\\n",
    linkInformation.Mode().Type() == os.ModeSymlink,
    targetInformation.Mode().Type() == os.ModeSymlink,
    strings.Join(names, ","),
    failedEntries == nil,
    errors.Is(notDirectoryError, syscall.ENOTDIR),
  )
}
`;
