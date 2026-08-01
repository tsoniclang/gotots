export function timeQuote(source: string): string {
  let result = '"';
  const encoder = new TextEncoder();
  for (const rune of source) {
    const codePoint = rune.codePointAt(0) ?? 0;
    if (codePoint >= 0x20 && codePoint < 0x80) {
      if (rune === '"' || rune === "\\") {
        result += "\\";
      }
      result += rune;
      continue;
    }
    for (const byte of encoder.encode(rune)) {
      result += `\\x${byte.toString(16).padStart(2, "0")}`;
    }
  }
  return `${result}"`;
}
