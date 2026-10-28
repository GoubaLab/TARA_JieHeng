// Validate the preview's renderable shape before untrusted JSON reaches React.
// The server remains authoritative for URL, version and transaction validation.
export function validateKnowledgePackPreview(value) {
  const object = (v) => v !== null && typeof v === "object" && !Array.isArray(v);
  const string = (v, max) => typeof v === "string" && v.trim().length > 0 && [...v].length <= max;
  const list = (v, max, size) => Array.isArray(v) && v.length <= max && v.every(x => string(x, size));
  if (!object(value) || value.schema !== "jieheng.knowledge-pack/1" || !string(value.pack_id, 100) || !string(value.version, 200)) throw Error("知识包格式、来源标识或版本无效");
  if (value.review_required !== undefined && (!Array.isArray(value.review_required) || value.review_required.length > 100)) throw Error("来源复核通知须为最多 100 项的列表");
  if (!Array.isArray(value.entries) || value.entries.length > 50 || (!value.entries.length && !value.review_required?.length)) throw Error("知识包须包含至多 50 条资料或来源复核通知");
  const seen = new Set();
  for (const [index, row] of value.entries.entries()) {
    if (!object(row) || !string(row.id, 100) || !string(row.name, 200) || !string(row.edition, 200) || !string(row.source, 1000) || !string(row.content, 120000) || !list(row.evidence_urls, 30, 1000) || !row.evidence_urls.length || (row.cve_ids !== undefined && !list(row.cve_ids, 50, 28)) || (row.tags !== undefined && !list(row.tags, 30, 80))) throw Error(`知识包第 ${index + 1} 条格式无效，请检查名称、正文、版本与来源列表`);
    if (seen.has(row.id)) throw Error(`知识包条目标识重复：${row.id}`);
    seen.add(row.id);
  }
  return value;
}
