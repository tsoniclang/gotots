import assert from "node:assert/strict";
import {
  mkdtempSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import {
  ContinueOnError,
  FlagSet,
  NewFlagSet,
} from "../src/flag.js";
import {
  Create,
  File,
  IsNotExist,
  MkdirAll,
  Remove,
  Stat,
} from "../src/os.js";
import {
  Cmd,
  Command,
} from "../src/os/exec.js";
import { FileMode } from "../src/io/fs.js";
import { SIGINT } from "../src/syscall.js";
import { sliceValues } from "../src/internal/runtime/slice.js";

test("host OS slice agrees with Go on selected deterministic behavior", () => {
  const fixture = mkdtempSync(join(tmpdir(), "gotots-os-diff-"));
  const goRoot = join(fixture, "go");
  const providerRoot = join(fixture, "provider");
  const source = join(fixture, "main.go");
  try {
    writeFileSync(source, goProgram);
    const goResult = spawnSync(
      "go",
      ["run", source, goRoot],
      {
        encoding: "utf8",
      },
    );
    assert.equal(goResult.status, 0, goResult.stderr);
    assert.equal(providerResult(providerRoot), goResult.stdout.trim());
  } finally {
    rmSync(fixture, {
      force: true,
      recursive: true,
    });
  }
});

function providerResult(root: string): string {
  const flags = NewFlagSet("provider", ContinueOnError);
  assert.ok(flags !== undefined);
  const verbose = FlagSet.Bool(flags, "verbose", false, "");
  const name = FlagSet.String(flags, "name", "", "");
  assert.ok(verbose !== undefined);
  assert.ok(name !== undefined);
  assert.equal(
    FlagSet.Parse(
      flags,
      RuntimeSlice.literal(["-verbose", "-name=value"]),
    ),
    undefined,
  );

  assert.equal(MkdirAll(root, new FileMode(0o755)), undefined);
  const path = join(root, "sample.txt");
  const [file, createError] = Create(path);
  assert.equal(createError, undefined);
  assert.ok(file !== undefined);
  assert.deepEqual(File.WriteString(file, "hello"), [5, undefined]);
  assert.equal(File.Close(file), undefined);
  const [information, statError] = Stat(path);
  assert.equal(statError, undefined);

  const command = Command("printf", RuntimeSlice.literal(["child"]));
  assert.ok(command !== undefined);
  const [output, outputError] = Cmd.Output(command);
  assert.equal(outputError, undefined);
  const child = Buffer.from(sliceValues(output)).toString("utf8");

  const missing = IsNotExist(Remove(join(root, "missing")));
  return [
    verbose.value,
    name.value,
    information?.Size(),
    child,
    missing,
    SIGINT.String(),
  ].join("|");
}

const goProgram = `
package main

import (
  "flag"
  "fmt"
  "os"
  "os/exec"
  "path/filepath"
  "syscall"
)

func main() {
  flags := flag.NewFlagSet("provider", flag.ContinueOnError)
  verbose := flags.Bool("verbose", false, "")
  name := flags.String("name", "", "")
  if err := flags.Parse([]string{"-verbose", "-name=value"}); err != nil {
    panic(err)
  }

  root := os.Args[1]
  if err := os.MkdirAll(root, 0755); err != nil {
    panic(err)
  }
  path := filepath.Join(root, "sample.txt")
  file, err := os.Create(path)
  if err != nil {
    panic(err)
  }
  if _, err := file.WriteString("hello"); err != nil {
    panic(err)
  }
  if err := file.Close(); err != nil {
    panic(err)
  }
  information, err := os.Stat(path)
  if err != nil {
    panic(err)
  }
  output, err := exec.Command("printf", "child").Output()
  if err != nil {
    panic(err)
  }
  missing := os.IsNotExist(os.Remove(filepath.Join(root, "missing")))
  fmt.Printf("%t|%s|%d|%s|%t|%s\\n", *verbose, *name, information.Size(), string(output), missing, syscall.SIGINT.String())
}
`;
