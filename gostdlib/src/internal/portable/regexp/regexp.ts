import type {
  GoError,
} from "@gotots/runtime/interface-value.js";
import { GoPanic } from "@gotots/runtime/panic.js";
import { RuntimeSlice } from "@gotots/runtime/slice.js";
import type {
  bool,
  gostring,
  int,
} from "@gotots/gostdlib/internal/scalars.js";

import { hostInteger, integerFromHost } from "../../host-integer.js";

import { ProviderError } from "../../runtime/error.js";
import {
  decodeRuneAt,
  fromHostString,
} from "../utf8/codec.js";
import {
  type CompiledPattern,
  translatePattern,
} from "./syntax.js";

type ByteRange = readonly [number, number];

type MatchRecord = {
  readonly ranges: readonly (ByteRange | undefined)[];
};

type ReplacementPart = {
  readonly prefix: gostring;
  readonly match: gostring;
};

type ReplacementPlan = {
  readonly parts: readonly ReplacementPart[];
  readonly suffix: gostring;
};

let createRegexp: (pattern: CompiledPattern) => Regexp;
let copyRegexpRepresentation: (source: Regexp) => Regexp;
let assignRegexpRepresentation: (target: Regexp, source: Regexp) => void;

export class Regexp {
  #pattern: CompiledPattern;

  private constructor(pattern: CompiledPattern) {
    this.#pattern = pattern;
  }

  static {
    createRegexp = (pattern: CompiledPattern): Regexp => new Regexp(pattern);
    copyRegexpRepresentation = (source): Regexp => new Regexp(source.#pattern);
    assignRegexpRepresentation = (target, source): void => {
      target.#pattern = source.#pattern;
    };
  }

  static FindStringSubmatch(
    receiver: Regexp | undefined,
    text: gostring,
  ): RuntimeSlice<gostring> {
    const regexp = requireRegexp(receiver);
    const match = regexp.#matches(text, 1n)[0];
    if (match === undefined) {
      return RuntimeSlice.nil<gostring>();
    }
    return RuntimeSlice.literal(
      match.ranges.map((range) => range === undefined ? "" : text.slice(range[0], range[1])),
    );
  }

  static MatchString(receiver: Regexp | undefined, text: gostring): bool {
    return requireRegexp(receiver).#matches(text, 1n).length > 0;
  }

  static ReplaceAllString(
    receiver: Regexp | undefined,
    source: gostring,
    replacement: gostring,
  ): gostring {
    const regexp = requireRegexp(receiver);
    return regexp.#replace(source, (match) => regexp.#expand(replacement, source, match));
  }

  static ReplaceAllStringFunc(
    receiver: Regexp | undefined,
    source: gostring,
    replacement: ((match: gostring) => gostring) | undefined,
  ): gostring {
    const regexp = requireRegexp(receiver);
    let result = "";
    const plan = regexp.#replacementPlan(source);
    for (const part of plan.parts) {
      if (replacement === undefined) {
        GoPanic.raiseRuntime("call of nil replacement function");
      }
      result += part.prefix + replacement(part.match);
    }
    return result + plan.suffix;
  }

  static Split(
    receiver: Regexp | undefined,
    source: gostring,
    count: int,
  ): RuntimeSlice<gostring> {
    if (count === 0n) {
      return RuntimeSlice.nil<gostring>();
    }
    const regexp = requireRegexp(receiver);
    if (regexp.#pattern.expression.length > 0 && source.length === 0) {
      return RuntimeSlice.literal([""]);
    }
    const matches = regexp.#matches(source, count);
    const output: gostring[] = [];
    let begin = 0;
    let end = 0;
    for (const match of matches) {
      if (count > 0n && integerFromHost(output.length) >= count - 1n) {
        break;
      }
      const whole = match.ranges[0];
      if (whole === undefined) {
        continue;
      }
      end = whole[0];
      if (whole[1] !== 0) {
        output.push(source.slice(begin, end));
      }
      begin = whole[1];
    }
    if (end !== source.length) {
      output.push(source.slice(begin));
    }
    return RuntimeSlice.literal(output);
  }

  #matches(source: gostring, limit: int): MatchRecord[] {
    if (limit === 0n) {
      return [];
    }
    const decoded = decodeText(source);
    const regexp = new globalThis.RegExp(this.#pattern.source, this.#pattern.flags);
    const matches: MatchRecord[] = [];
    while (limit < 0n || integerFromHost(matches.length) < limit) {
      const match = regexp.exec(decoded.host);
      if (match === null) {
        break;
      }
      const indices = match.indices;
      if (indices === undefined) {
        GoPanic.raiseRuntime("regexp indices are unavailable");
      }
      matches.push({
        ranges: indices.map((range) => {
          if (range === undefined) {
            return undefined;
          }
          const start = decoded.byteAtHost[range[0]];
          const end = decoded.byteAtHost[range[1]];
          return start === undefined || end === undefined ? undefined : [start, end];
        }),
      });
      if (match[0].length === 0) {
        regexp.lastIndex = advanceHostIndex(decoded.host, regexp.lastIndex);
      }
    }
    return matches;
  }

  #replace(source: gostring, replacement: (match: MatchRecord) => gostring): gostring {
    let result = "";
    let lastEnd = 0;
    for (const match of this.#matches(source, -1n)) {
      const whole = match.ranges[0];
      if (whole === undefined) {
        continue;
      }
      result += source.slice(lastEnd, whole[0]);
      if (whole[1] > lastEnd || whole[0] === 0) {
        result += replacement(match);
      }
      lastEnd = whole[1];
    }
    return result + source.slice(lastEnd);
  }

  #replacementPlan(source: gostring): ReplacementPlan {
    const parts: ReplacementPart[] = [];
    let lastEnd = 0;
    for (const match of this.#matches(source, -1n)) {
      const whole = match.ranges[0];
      if (whole === undefined) {
        continue;
      }
      if (whole[1] > lastEnd || whole[0] === 0) {
        parts.push({
          prefix: source.slice(lastEnd, whole[0]),
          match: source.slice(whole[0], whole[1]),
        });
      }
      lastEnd = whole[1];
    }
    return {
      parts,
      suffix: source.slice(lastEnd),
    };
  }

  #expand(template: gostring, source: gostring, match: MatchRecord): gostring {
    let result = "";
    for (let index = 0; index < template.length; ) {
      const dollar = template.indexOf("$", index);
      if (dollar < 0) {
        return result + template.slice(index);
      }
      result += template.slice(index, dollar);
      if (template[dollar + 1] === "$") {
        result += "$";
        index = dollar + 2;
        continue;
      }
      const variable = extractVariable(template, dollar + 1);
      if (variable === undefined) {
        result += "$";
        index = dollar + 1;
        continue;
      }
      const captureIndex = /^[0-9]+$/.test(variable.name)
        ? Number(variable.name)
        : this.#pattern.names.get(variable.name);
      const range = captureIndex === undefined ? undefined : match.ranges[captureIndex];
      if (range !== undefined) {
        result += source.slice(range[0], range[1]);
      }
      index = variable.end;
    }
    return result;
  }
}

export function regexpRepresentationCopy(source: Regexp): Regexp {
  return copyRegexpRepresentation(source);
}

export function regexpRepresentationAssign(target: Regexp, source: Regexp): void {
  assignRegexpRepresentation(target, source);
}

export function Compile(expression: gostring): [Regexp | undefined, GoError | undefined] {
  try {
    const pattern = translatePattern(expression);
    void new globalThis.RegExp(pattern.source, pattern.flags);
    return [createRegexp(pattern), undefined];
  } catch {
    return [undefined, new ProviderError(regexpErrorMessage(expression))];
  }
}

export function MustCompile(expression: gostring): Regexp {
  const [regexp, error] = Compile(expression);
  if (error !== undefined) {
    GoPanic.raise(error);
  }
  if (regexp === undefined) {
    GoPanic.raiseRuntime("regexp compilation returned no result");
  }
  return regexp;
}

function requireRegexp(regexp: Regexp | undefined): Regexp {
  if (regexp === undefined) {
    GoPanic.raiseRuntime("nil *regexp.Regexp");
  }
  return regexp;
}

type DecodedText = {
  readonly host: string;
  readonly byteAtHost: readonly number[];
};

function decodeText(source: gostring): DecodedText {
  let host = "";
  const byteAtHost: number[] = [0];
  for (let index = 0; index < source.length; ) {
    const [rune, width] = decodeRuneAt(source, index);
    const scalar = String.fromCodePoint(rune);
    host += scalar;
    if (scalar.length === 2) {
      byteAtHost.push(index);
    }
    index += Math.max(1, hostInteger(width));
    byteAtHost.push(index);
  }
  return { host, byteAtHost };
}

function extractVariable(
  template: gostring,
  start: number,
): { readonly name: string; readonly end: number } | undefined {
  if (template[start] === "{") {
    const end = template.indexOf("}", start + 1);
    const name = end < 0 ? "" : template.slice(start + 1, end);
    return end < 0 || !/^[0-9A-Za-z_]+$/.test(name)
      ? undefined
      : { name, end: end + 1 };
  }
  let end = start;
  while (end < template.length && /[0-9A-Za-z_]/.test(template[end] ?? "")) {
    end += 1;
  }
  return end === start ? undefined : { name: template.slice(start, end), end };
}

function advanceHostIndex(text: string, index: number): number {
  if (index >= text.length) {
    return index + 1;
  }
  const code = text.codePointAt(index);
  return index + (code !== undefined && code > 0xffff ? 2 : 1);
}

function regexpErrorMessage(expression: gostring): string {
  return `error parsing regexp: invalid syntax: ${fromHostString(JSON.stringify(expression))}`;
}
