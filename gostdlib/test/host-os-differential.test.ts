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
  Open,
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
  const payload = Buffer.from([
    0xe2, 0x80, 0x9c,
    0xc3, 0xa9,
    0xf0, 0x9f, 0x99, 0x82,
    0x00, 0xff,
  ]);
  assert.deepEqual(
    File.WriteString(file, payload.toString("latin1")),
    [11n, undefined],
  );
  assert.equal(File.Close(file), undefined);
  const [information, statError] = Stat(path);
  assert.equal(statError, undefined);
  const [opened, openError] = Open(path);
  assert.equal(openError, undefined);
  assert.ok(opened !== undefined);
  const contents = RuntimeSlice.make<number>(11, null, 0);
  assert.deepEqual(File.Read(opened, contents), [11n, undefined]);
  assert.equal(File.Close(opened), undefined);

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
    Buffer.from(sliceValues(contents)).toString("hex"),
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
  payload := string([]byte{0xe2, 0x80, 0x9c, 0xc3, 0xa9, 0xf0, 0x9f, 0x99, 0x82, 0x00, 0xff})
  if _, err := file.WriteString(payload); err != nil {
    panic(err)
  }
  if err := file.Close(); err != nil {
    panic(err)
  }
  information, err := os.Stat(path)
  if err != nil {
    panic(err)
  }
  opened, err := os.Open(path)
  if err != nil {
    panic(err)
  }
  contents := make([]byte, len(payload))
  if _, err := opened.Read(contents); err != nil {
    panic(err)
  }
  if err := opened.Close(); err != nil {
    panic(err)
  }
  output, err := exec.Command("printf", "child").Output()
  if err != nil {
    panic(err)
  }
  missing := os.IsNotExist(os.Remove(filepath.Join(root, "missing")))
  fmt.Printf("%t|%s|%d|%x|%s|%t|%s\\n", *verbose, *name, information.Size(), contents, string(output), missing, syscall.SIGINT.String())
}
`;
