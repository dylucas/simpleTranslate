export const MAX_INPUT_BYTES = 6000;

const encoder = new TextEncoder();

export function utf8ByteLength(value: string): number {
  return encoder.encode(value).byteLength;
}

export function truncateUtf8(value: string, maxBytes = MAX_INPUT_BYTES): string {
  if (utf8ByteLength(value) <= maxBytes) return value;
  let used = 0;
  let output = "";
  for (const character of value) {
    const bytes = utf8ByteLength(character);
    if (used + bytes > maxBytes) break;
    output += character;
    used += bytes;
  }
  return output;
}
