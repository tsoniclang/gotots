import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import {
  mkdtempSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import test from "node:test";
import { Unix } from "../src/time.js";

test("time Unix agrees with Go on nanosecond normalization", () => {
  const root = mkdtempSync(join(tmpdir(), "gotots-time-unix-"));
  const source = join(root, "main.go");
  try {
    writeFileSync(source, goProgram);
    const goResult = spawnSync(
      "go",
      ["run", source],
      { encoding: "utf8" },
    );
    assert.equal(goResult.status, 0, goResult.stderr);
    assert.equal(providerResult(), goResult.stdout.trim());
  } finally {
    rmSync(root, {
      force: true,
      recursive: true,
    });
  }
});

function providerResult(): string {
  return [
    Unix(0, 0),
    Unix(0, -1),
    Unix(1, 1_500_001),
    Unix(-2, 2_000_000_001),
    Unix(2, -1_000_000_001),
  ].map((value) => `${value.UnixMilli()}:${value.UnixNano()}`).join("|");
}

const goProgram = `
package main

import (
  "fmt"
  "time"
)

func main() {
  values := []time.Time{
    time.Unix(0, 0),
    time.Unix(0, -1),
    time.Unix(1, 1_500_001),
    time.Unix(-2, 2_000_000_001),
    time.Unix(2, -1_000_000_001),
  }
  for index, value := range values {
    if index > 0 {
      fmt.Print("|")
    }
    fmt.Printf("%d:%d", value.UnixMilli(), value.UnixNano())
  }
  fmt.Println()
}
`;
