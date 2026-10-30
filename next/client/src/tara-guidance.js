// Application summaries of ISO/SAE 21434:2021 Annexes F and G.
export const impactGuidance = {
  safety: {
    source: "附录 F 表 F.1；15.5，ISO 26262-3:2018 §6.4.3",
    rows: [
      "S0：不会造成人员伤害。",
      "S1：可能造成轻度或中度伤害。",
      "S2：可能造成严重伤害，或危及生命但预计能够存活。",
      "S3：可能造成致命伤害，或危及生命且存活不确定。",
    ],
  },
  financial: {
    source: "附录 F 表 F.2",
    rows: [
      "对道路使用者没有财务影响，或损失轻微到可以忽略。",
      "损失造成不便，受影响者利用有限资源即可承担并恢复。",
      "损失明显、后果较大，但受影响者仍有能力承担和恢复。",
      "经济损失具有灾难性，可能超出受影响者能够承担或恢复的能力。",
    ],
  },
  operational: {
    source: "附录 F 表 F.3",
    rows: [
      "车辆功能没有受损，或变化无法被使用者察觉。",
      "部分功能性能下降，降低使用体验，但重要功能保持可用。",
      "重要功能丧失或受损，显著妨碍车辆使用。",
      "核心功能丧失或异常，例如车辆无法正常工作、进入降级行驶或出现非预期运动。",
    ],
  },
  privacy: {
    source: "附录 F 表 F.4",
    rows: [
      "没有可感知的个人隐私后果；信息不敏感且难以关联到特定个人。",
      "造成隐私方面的不便；涉及敏感但难以关联个人的信息，或不敏感但容易关联个人的信息。",
      "造成严重的隐私后果；涉及高度敏感但难以关联个人的信息，或敏感且容易关联个人的信息。",
      "造成重大乃至不可逆的隐私后果；涉及高度敏感且容易关联到个人的信息。",
    ],
  },
};
export const potentialFactors = [
  {
    key: "elapsed_time",
    label: "ET · 所需时间",
    source: "G.1、G.6",
    description: "识别弱点、开发方法并成功实施攻击所需的累计时间。",
    options: [
      [0, "不超过 1 天"],
      [1, "超过 1 天，不超过 1 周"],
      [4, "超过 1 周，不超过 1 个月"],
      [17, "超过 1 个月，不超过 6 个月"],
      [19, "超过 6 个月"],
    ],
  },
  {
    key: "expertise",
    label: "Ex · 专业知识",
    source: "G.2、G.6",
    description: "按实施整条路径所需的知识和经验判断。",
    options: [
      [0, "普通人员：可按公开步骤操作"],
      [3, "熟练人员：熟悉该类产品及常见攻击"],
      [6, "专家：掌握底层协议、硬件或安全机制"],
      [8, "多领域专家：不同步骤需要多类专业能力"],
    ],
  },
  {
    key: "knowledge",
    label: "KoI · 相关项知识",
    source: "G.3、G.6",
    description: "判断攻击所需资料的实际可获得性。",
    options: [
      [0, "公开资料：无需授权即可获得"],
      [3, "受限资料：在组织或合作方中受控共享"],
      [7, "机密资料：仅限特定团队获得"],
      [11, "严格保密资料：极少数人员按需访问"],
    ],
  },
  {
    key: "opportunity",
    label: "WoO · 机会窗口",
    source: "G.4、G.6",
    description: "结合访问方式、目标可达性和可持续访问的时间判断。",
    options: [
      [0, "不受限：持续可达，无特别时间条件"],
      [1, "容易：可达性高，但需等待有限时段"],
      [4, "中等：目标可达性低，物理或逻辑访问受限"],
      [10, "困难：极短时窗或极低可达性，实施条件苛刻"],
    ],
  },
  {
    key: "equipment",
    label: "Eq · 所需设备",
    source: "G.5、G.6",
    description: "按整条路径所需工具的获取、研发与组合难度判断。",
    options: [
      [0, "通用设备：常见电脑、公开工具或通用接口适配器"],
      [4, "专业设备：专用测试、调试或分析设备"],
      [7, "定制设备：需专门研制，或昂贵且获取受限"],
      [9, "多套定制设备：不同步骤需要不同定制工具"],
    ],
  },
];
export const attackVectors = [
  {
    value: "network",
    label: "网络 · 高",
    level: 4,
    description: "通过网络远程到达目标，不要求处于特定邻接网络。",
  },
  {
    value: "adjacent",
    label: "邻接 · 中",
    level: 3,
    description: "需要物理或逻辑邻接网络，例如蓝牙覆盖范围或受限局域网。",
  },
  {
    value: "local",
    label: "本地 · 低",
    level: 2,
    description: "需要本地访问能力，例如已有本地执行权限或访问本地存储接口。",
  },
  {
    value: "physical",
    label: "物理 · 很低",
    level: 1,
    description: "需要直接物理接触或操纵目标硬件。",
  },
];
export function potentialResult(potential, method = {}) {
  if (
    !potentialFactors.every((f) =>
      f.options.some(([v]) => v === potential?.[f.key]),
    )
  )
    return null;
  const total = potentialFactors.reduce((s, f) => s + potential[f.key], 0);
  const thresholds = method.attack_potential_thresholds || [14, 20, 25];
  return {
    total,
    level: 4 - thresholds.filter((t) => total >= t).length,
    thresholds,
  };
}
export function functionCatalog(ws) {
  const records = new Map();
  const text = ws.item?.purpose || "";
  for (const match of text.matchAll(
    /(FR-\d+)(?:\.\.(FR-\d+))?\s+([^；;\n]+)/g,
  )) {
    const first = Number(match[1].slice(3)),
      last = match[2] ? Number(match[2].slice(3)) : first;
    if (last < first || last - first > 500) continue;
    const width = match[1].slice(3).length;
    for (let n = first; n <= last; n++) {
      const id = `FR-${String(n).padStart(width, "0")}`;
      records.set(id, {
        id,
        name: match[3].split(/[：:]/)[0],
        description: match[3],
        group: first !== last ? `${match[1]}–${match[2]}` : "",
        source: "相关项定义 · 功能与预期用途",
        derived: true,
      });
    }
  }
  for (const f of ws.item?.functions || [])
    records.set(f.id, { ...f, source: "项目功能目录", derived: false });
  return [...records.values()];
}
export function projectImpactCriteria(ws, key) {
  if (ws.method?.impact_criteria?.[key])
    return { source: "项目配置", rows: ws.method.impact_criteria[key] };
  const sheet = ws.tara_import?.sheets?.find((s) =>
    /影响评级对照/.test(s.name),
  );
  if (!sheet) return null;
  const index = ["safety", "financial", "operational", "privacy"].indexOf(key);
  const start = sheet.rows.findIndex((r) =>
    new RegExp(`^A\\.${index + 1}\\s`).test(r[0] || ""),
  );
  if (start < 0) return null;
  const rows = sheet.rows
    .slice(start + 1)
    .filter((r) => /^(0|10|100|1000)$/.test(r[0] || ""))
    .slice(0, 4);
  if (rows.length !== 4) return null;
  return {
    source: `来源工作簿 · ${sheet.name} · A.${index + 1}`,
    rows: rows.map((r) => ({
      label: r[1],
      description: [r[2], r[3]].filter(Boolean).join("；"),
    })),
  };
}
export const scenarioKeys = [
  "id",
  "title",
  "asset_id",
  "property",
  "damage",
  "threat",
  "impact",
  "paths",
  "references",
  "source_code",
  "asset_ids",
  "damage_ids",
  "properties",
  "stride",
  "cause",
  "cal",
];
export const pickScenario = (s) =>
  Object.fromEntries(
    scenarioKeys
      .filter((k) => s[k] !== undefined)
      .map((k) => [k, structuredClone(s[k])]),
  );
export const recordCode = (r) =>
  r?.source_code || r?.display_code || r?.id || "";
export const recordText = (r) =>
  r?.name || r?.title || r?.description || r?.statement || "内容待补充";
export const recordLabel = (r) => `${recordCode(r)} · ${recordText(r)}`;
