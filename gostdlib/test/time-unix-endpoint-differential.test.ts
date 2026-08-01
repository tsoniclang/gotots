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
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import {
  Parse,
  Time,
  Unix,
} from "../src/time.js";

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

test("time parsing agrees with Go on instants, layouts, and rejection", () => {
  const root = mkdtempSync(join(tmpdir(), "gotots-time-parse-"));
  const source = join(root, "main.go");
  try {
    writeFileSync(source, goParseProgram);
    const goResult = spawnSync(
      "go",
      ["run", source],
      { encoding: "utf8" },
    );
    assert.equal(goResult.status, 0, goResult.stderr);
    assert.equal(providerParseResult(), goResult.stdout.trim());
  } finally {
    rmSync(root, {
      force: true,
      recursive: true,
    });
  }
});

function providerResult(): string {
  const values = [
    Unix(0, 0),
    Unix(0, -1),
    Unix(1, 1_500_001),
    Unix(-2, 2_000_000_001),
    Unix(2, -1_000_000_001),
  ];
  const instants = values.map((value) => (
    `${value.Unix()}:${value.UnixMilli()}:${value.UnixNano()}:${value.Nanosecond()}`
      + `:${value.UTC().Format("2006-01-02T15:04:05.000000000Z07:00")}`
  )).join("|");
  return [
    instants,
    timeTextLine(new Time()),
    timeTextLine(Unix(0, -1).UTC()),
  ].join("\n");
}

function timeTextLine(value: Time): string {
  const [appended, appendFailure] = value.AppendText(
    RuntimeSlice.literal([0x70, 0x3d]),
  );
  const [marshaled, marshalFailure] = value.MarshalText();
  const [json, jsonFailure] = value.MarshalJSON();
  return `append=${decodeBytes(appended)},ok=${appendFailure === undefined}`
    + `;marshal=${decodeBytes(marshaled)},ok=${marshalFailure === undefined}`
    + `;json=${decodeBytes(json)},ok=${jsonFailure === undefined}`;
}

function decodeBytes(value: RuntimeSlice<number>): string {
  return new TextDecoder().decode(Uint8Array.from(
    Array.from({ length: value.length }, (_, index) => value.get(index)),
  ));
}

function providerParseResult(): string {
  const unmarshaled = new Time();
  const unmarshalFailure = unmarshaled.UnmarshalText(RuntimeSlice.literal(
    Array.from(new TextEncoder().encode("2024-01-02T03:04:05.123456789+02:30")),
  ));
  const [calendar, calendarFailure] = Parse(
    "2006-01-02 15:04:05",
    "2024-02-29 23:07:08",
  );
  const [yearDay, yearDayFailure] = Parse(
    "2006-002 15:04:05.999999999Z07:00",
    "2024-060 01:02:03.456789123+02:30",
  );
  const invalid = new Time();
  const invalidFailure = invalid.UnmarshalText(RuntimeSlice.literal(
    Array.from(new TextEncoder().encode("not-a-time")),
  ));
  return [
    parseLine(unmarshaled, unmarshalFailure === undefined),
    parseLine(calendar, calendarFailure === undefined),
    parseLine(yearDay, yearDayFailure === undefined),
    `error=${invalidFailure !== undefined};zero=${invalid.IsZero()}`,
  ].join("\n");
}

function parseLine(value: Time, ok: boolean): string {
  return `ok=${ok};unixmilli=${value.UnixMilli()};format=${value.Format(
    "2006-01-02T15:04:05.000000000Z07:00",
  )}`;
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
    fmt.Printf("%d:%d:%d:%d:%s", value.Unix(), value.UnixMilli(), value.UnixNano(), value.Nanosecond(), value.UTC().Format("2006-01-02T15:04:05.000000000Z07:00"))
  }
  fmt.Println()
  for _, value := range []time.Time{{}, time.Unix(0, -1).UTC()} {
    appended, appendErr := value.AppendText([]byte("p="))
    marshaled, marshalErr := value.MarshalText()
    json, jsonErr := value.MarshalJSON()
    fmt.Printf("append=%s,ok=%t;marshal=%s,ok=%t;json=%s,ok=%t\\n", appended, appendErr == nil, marshaled, marshalErr == nil, json, jsonErr == nil)
  }
}
`;

const goParseProgram = `
package main

import (
  "fmt"
  "time"
)

func line(value time.Time, err error) {
  fmt.Printf("ok=%t;unixmilli=%d;format=%s\\n", err == nil, value.UnixMilli(), value.Format("2006-01-02T15:04:05.000000000Z07:00"))
}

func main() {
  var unmarshaled time.Time
  err := unmarshaled.UnmarshalText([]byte("2024-01-02T03:04:05.123456789+02:30"))
  line(unmarshaled, err)
  calendar, err := time.Parse("2006-01-02 15:04:05", "2024-02-29 23:07:08")
  line(calendar, err)
  yearDay, err := time.Parse("2006-002 15:04:05.999999999Z07:00", "2024-060 01:02:03.456789123+02:30")
  line(yearDay, err)
  var invalid time.Time
  err = invalid.UnmarshalText([]byte("not-a-time"))
  fmt.Printf("error=%t;zero=%t\\n", err != nil, invalid.IsZero())
}
`;
