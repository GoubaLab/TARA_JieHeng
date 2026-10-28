export const MAX_ATTACK_PATHS = 100;

export function pathTemplateCode(template) {
  const code = template?.source_code;
  if (
    typeof code !== "string" ||
    !code.startsWith("TPL:") ||
    [...code].length > 100
  )
    throw Error("攻击路径模板缺少有效的来源标识，请刷新模板列表");
  return code;
}

export function existingTemplateCodes(paths = []) {
  return new Set(
    paths
      .map((path) => path?.source_code)
      .filter((code) => typeof code === "string" && code.startsWith("TPL:")),
  );
}

function templateText(value, label, max, optional = false) {
  if (optional && (value === undefined || value === null)) return "";
  if (
    typeof value !== "string" ||
    (!optional && !value.trim()) ||
    [...value].length > max
  )
    throw Error(`模板${label}无效，请刷新模板列表`);
  return value.trim();
}
function templateList(value, label) {
  if (
    !Array.isArray(value) ||
    !value.length ||
    !value.every((v) => typeof v === "string" && v.trim())
  )
    throw Error(`模板${label}无效，请刷新模板列表`);
  return value.map((v) => v.trim());
}

export function templateToPath(
  template,
  makeId = () => `path_${crypto.randomUUID()}`,
) {
  const source_code = pathTemplateCode(template);
  const name = templateText(template.name, "名称", 500);
  const actor = templateText(template.actor, "攻击主体", 500);
  const vector = templateText(template.entry_point, "入口", 200);
  const objective = templateText(template.objective, "目标", 2000);
  const prerequisites = templateList(template.prerequisites, "前提");
  const steps = templateList(template.steps, "步骤");
  const controls = templateList(template.controls_to_check, "控制检查项");
  const content = [
    `入口：${vector}`,
    `攻击主体：${actor}`,
    `适用前提：\n${prerequisites.map((v) => `- ${v}`).join("\n")}`,
    `攻击步骤：\n${steps.map((v, i) => `${i + 1}. ${v}`).join("\n")}`,
    `攻击目标与可能后果：${objective}`,
    `待核对的控制：\n${controls.map((v) => `- ${v}`).join("\n")}`,
    "以上为路径分析模板。入口可达性、前提及控制实施情况需要结合项目核实。",
  ].join("\n\n");
  if ([...content].length > 10000)
    throw Error("模板内容超过路径长度上限，未添加任何路径");
  const id = makeId();
  if (typeof id !== "string" || !id.trim())
    throw Error("无法生成有效的攻击路径编号");
  return {
    id,
    name,
    source_code,
    actor,
    vector,
    steps: content,
    feasibility: null,
    rationale: "",
    reference: null,
  };
}

export function appendTemplatePaths(
  existingPaths = [],
  templates = [],
  makeId,
) {
  if (!Array.isArray(existingPaths) || !Array.isArray(templates))
    throw Error("攻击路径列表格式无效");
  const used = existingTemplateCodes(existingPaths);
  const fresh = [];
  for (const template of templates) {
    const code = pathTemplateCode(template);
    if (!used.has(code)) {
      fresh.push(template);
      used.add(code);
    }
  }
  if (existingPaths.length + fresh.length > MAX_ATTACK_PATHS)
    throw Error(
      `每个风险场景最多 ${MAX_ATTACK_PATHS} 条攻击路径，请减少选择或整理已有路径`,
    );
  const ids = new Set(existingPaths.map((p) => p.id));
  const additions = fresh.map((template) => {
    const path = templateToPath(template, makeId);
    if (ids.has(path.id))
      throw Error("生成的路径编号重复，未添加任何路径，请重试");
    ids.add(path.id);
    return path;
  });
  return {
    paths: [...existingPaths, ...additions],
    added: additions.length,
    skipped: templates.length - additions.length,
  };
}
