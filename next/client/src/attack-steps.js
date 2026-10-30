// Numbered imported steps remain individual actions; unnumbered prose stays intact.
export function parseAttackSteps(value = "") {
  const text = value.replace(/\r\n?/g, "\n").trim();
  if (!text) return [""];
  const lines = text.split("\n"),
    result = [];
  const marker = /^\s*\d+\s*[、.．)）]\s*/;
  for (const line of lines) {
    if (marker.test(line)) result.push(line.replace(marker, ""));
    else if (result.length) result[result.length - 1] += "\n" + line;
    else result.push(line);
  }
  return result;
}
export const formatAttackSteps = (steps) =>
  steps.map((s, i) => `${i + 1}、${s}`).join("\n");
