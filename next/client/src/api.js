export const properties = {
  confidentiality: "机密性",
  integrity: "完整性",
  availability: "可用性",
  authenticity: "真实性",
  authorization: "授权",
  non_repudiation: "不可抵赖性",
};
export const impacts = ["可忽略", "中等", "重大", "严重"];
export const feasibility = ["很低", "低", "中", "高"];
export const treatments = {
  reduce: "降低风险",
  avoid: "规避风险",
  share: "分担风险",
  retain: "保留风险",
};
export const pages = {
  overview: "项目总览",
  definition: "相关项定义",
  architecture: "架构与资产",
  assessment: "风险分析",
  treatment: "风险处置与需求",
  verification: "验证与报告",
  library: "参考资料",
  settings: "评估设置",
  regulations: "法规数据库",
};
export const actions = {
  "open-workspace": "创建项目",
  "define-item": "更新相关项",
  "replace-architecture": "保存架构",
  "record-asset": "更新资产",
  "record-scenario": "更新风险场景",
  "decide-treatment": "记录处置决策",
  "specify-requirement": "更新安全需求",
  "record-evidence": "新增验证记录",
  "adopt-method": "确认评估方法",
  "seal-baseline": "建立评审基线",
  "apply-reference-selection": "批量选用参考资料",
  "select-regulations": "更新项目法规引用",
};
export const apiRoot = "/api/v1";
export async function request(path, options = {}) {
  const r = await fetch(apiRoot + path, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      "X-Workspace-Client": "jieheng",
      ...options.headers,
    },
  });
  let body;
  try {
    body = await r.json();
  } catch {
    throw new Error("服务未返回有效数据，请检查本地服务。");
  }
  if (!r.ok) {
    const e = new Error(body.detail || body.title || "操作未完成");
    e.status = r.status;
    e.requestId = body.request_id || r.headers.get("X-Request-ID");
    throw e;
  }
  return body;
}
export function command(ws, action, payload, requestId = crypto.randomUUID()) {
  const key = (value) => encodeURIComponent(value);
  const routes = {
    "define-item": ["PUT", "/item"],
    "replace-architecture": ["PUT", "/architecture"],
    "record-asset": [payload.id ? "PUT" : "POST", `/assets${payload.id ? "/" + key(payload.id) : ""}`],
    "record-scenario": [payload.id ? "PUT" : "POST", `/scenarios${payload.id ? "/" + key(payload.id) : ""}`],
    "decide-treatment": ["PUT", `/scenarios/${key(payload.scenario_id)}/treatment`],
    "specify-requirement": [payload.id ? "PUT" : "POST", `/requirements${payload.id ? "/" + key(payload.id) : ""}`],
    "record-evidence": ["POST", "/evidence"],
    "adopt-method": ["PUT", "/assessment-method"],
    "seal-baseline": ["POST", "/baselines"],
    "apply-reference-selection": ["POST", "/reference-imports"],
    "select-regulations": ["PUT", "/regulation-references"],
  };
  const route = routes[action];
  if (!route) throw new Error("未知操作，请刷新页面后重试。");
  const body = { ...payload };
  if (["record-asset", "record-scenario", "specify-requirement", "record-evidence"].includes(action)) delete body.id;
  if (action === "decide-treatment") delete body.scenario_id;
  return request(`/workspaces/${key(ws.id)}${route[1]}`, {
    method: route[0],
    headers: { "If-Match": `"${ws.revision}"`, "Idempotency-Key": requestId },
    body: JSON.stringify(body),
  });
}
export async function allWorkspaces() {
  const items = [];
  let offset = 0;
  do {
    const page = await request(`/workspaces?offset=${offset}&limit=100`);
    items.push(...page.items);
    offset += page.items.length;
    if (offset >= page.total || !page.items.length) break;
  } while (true);
  return { items };
}
export async function workspaceEvents(id) {
  const entries = [];
  let offset = 0;
  do {
    const page = await request(`/workspaces/${encodeURIComponent(id)}/events?offset=${offset}&limit=100`);
    entries.push(...page.entries);
    offset += page.entries.length;
    if (offset >= page.total || !page.entries.length) break;
  } while (true);
  return { entries };
}
export function navigate(page, id) {
  location.hash = id ? `/workspace/${id}/${page}` : `/${page}`;
}
export const match = (row, q) =>
  !q ||
  JSON.stringify(row)
    .toLocaleLowerCase()
    .includes(q.trim().toLocaleLowerCase());
export const date = (v) =>
  v ? new Date(v).toLocaleString("zh-CN", { hour12: false }) : "—";
export const propOptions = Object.entries(properties).map(([value, label]) => ({
  value,
  label,
}));
export const gradeOptions = impacts.map((label, value) => ({ label, value }));
export const feasibilityOptions = feasibility.map((label, i) => ({
  label,
  value: i + 1,
}));
export function evidenceStatus(req, ws, review) {
  const supplied = review.requirements?.find(
    (r) => r.requirement_id === req.id,
  );
  if (supplied) return supplied.status;
  const rows = ws.evidence.filter((e) => e.requirement_id === req.id),
    latest = rows.at(-1);
  if (!latest) return "missing";
  if (
    review.issues.some(
      (i) => i.stage === "verification" && i.reference === req.id,
    ) &&
    latest.result === "pass"
  )
    return "stale";
  return latest.result;
}
export async function exportFile(ws, format, baseline) {
  const r = await fetch(
    `${apiRoot}/workspaces/${encodeURIComponent(ws.id)}/reports?format=${encodeURIComponent(format)}${baseline ? "&baseline_id=" + encodeURIComponent(baseline) : ""}`,
  );
  if (!r.ok) {
    const problem = await r.json();
    throw new Error(problem.detail || "生成失败，请重试");
  }
  const blob = await r.blob();
  if (blob.size < 100) throw new Error("生成文件内容不完整，请重试");
  if (format === "xlsx") {
    const head = new Uint8Array(await blob.slice(0, 4).arrayBuffer());
    if (head[0] !== 80 || head[1] !== 75)
      throw new Error("生成文件不是有效的 Excel 文件");
  }
  const url = URL.createObjectURL(blob),
    a = document.createElement("a");
  a.href = url;
  a.download = `${ws.title.replace(/[<>:"/\\|?*]/g, "_")}-威胁建模-${baseline ? "基线" : "R" + ws.revision}.${format}`;
  a.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
  return {
    name: a.download,
    size: blob.size,
    at: new Date().toISOString(),
    revision: ws.revision,
  };
}
