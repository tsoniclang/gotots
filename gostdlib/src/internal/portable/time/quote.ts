export function timeQuote(source: string): string {
  let result = '"';
  for (let index = 0; index < source.length; index++) {
    const byte = source.charCodeAt(index);
    const character = source.charAt(index);
    if (byte >= 0x20 && byte < 0x80) {
      if (character === '"' || character === "\\") {
        result += "\\";
      }
      result += character;
      continue;
    }
    result += `\\x${byte.toString(16).padStart(2, "0")}`;
  }
  return `${result}"`;
}
