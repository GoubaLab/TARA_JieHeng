export const numberingTypes = [
  ["entity", "实体", "EN"],
  ["boundary", "信任边界", "TB"],
  ["software", "内部软件", "SW"],
  ["store", "数据存储", "DS"],
  ["flow", "数据流", "DF"],
  ["asset", "资产", "AS"],
  ["damage", "损害场景", "DM"],
  ["goal", "安全目标", "CG"],
  ["scenario", "风险场景", "RS"],
  ["path", "攻击路径", "AP"],
  ["requirement", "安全需求", "SR"],
  ["evidence", "验证记录", "EV"],
];
export function effectiveNumbering(value = {}) {
  return {
    separator: "-",
    padding: 3,
    ...value,
    prefixes: {
      ...Object.fromEntries(
        numberingTypes.map(([key, , prefix]) => [key, prefix]),
      ),
      ...value.prefixes,
    },
  };
}
export function formatNumber(rules, kind, sequence) {
  return `${rules.prefixes[kind]}${rules.separator}${String(sequence).padStart(rules.padding, "0")}`;
}
export const pathNumberKey = (scenarioId, pathId) =>
  `${new TextEncoder().encode(scenarioId).length}:${scenarioId}${pathId}`;
export const nodeNumberKind = (node) =>
  node.kind === "boundary"
    ? "boundary"
    : ["store", "data_store"].includes(node.kind)
      ? "store"
      : node.kind === "process_step"
        ? "software"
        : "entity";
export function buildCodeIndex(ws, draftGraph) {
  const rules = effectiveNumbering(ws.numbering),
    state = new Map();
  for (const [kind] of numberingTypes)
    state.set(kind, new Map(Object.entries(ws.numbering_state?.[kind] || {})));
  const next = new Map(
    [...state].map(([kind, entries]) => [
      kind,
      [...entries.values()].reduce((max, n) => Math.max(max, n), 0),
    ]),
  );
  const add = (kind, id) => {
    const entries = state.get(kind);
    if (!entries || !id || entries.has(id)) return;
    const n = next.get(kind) + 1;
    next.set(kind, n);
    entries.set(id, n);
  };
  for (const graph of [ws.architecture, draftGraph]) {
    for (const node of graph?.nodes || []) {
      add(nodeNumberKind(node), node.id);
      for (const child of node.children || []) add(child.kind, child.id);
    }
    for (const flow of graph?.flows || []) add("flow", flow.id);
  }
  for (const [collection, kind] of [
    ["assets", "asset"],
    ["damage_scenarios", "damage"],
    ["security_goals", "goal"],
    ["scenarios", "scenario"],
    ["requirements", "requirement"],
    ["evidence", "evidence"],
  ]) {
    for (const row of ws[collection] || []) {
      add(kind, row.id);
      if (kind === "scenario")
        for (const path of row.paths || [])
          add("path", pathNumberKey(row.id, path.id));
    }
  }
  return (kind, id) =>
    state.get(kind)?.has(id)
      ? formatNumber(rules, kind, state.get(kind).get(id))
      : id;
}
const cache = new WeakMap();
export function recordCode(ws, kind, id) {
  if (!cache.has(ws)) cache.set(ws, buildCodeIndex(ws));
  return cache.get(ws)(kind, id);
}
export function numberedGraph(graph, code) {
  if (!code || graph.show_numbers === false) return graph;
  const label = (kind, value) => ({
    ...value,
    name: `${code(kind, value.id)} · ${value.name}`,
  });
  return {
    ...graph,
    nodes: graph.nodes.map((node) => ({
      ...label(nodeNumberKind(node), node),
      children: (node.children || []).map((child) => label(child.kind, child)),
    })),
    flows: graph.flows.map((flow) => label("flow", flow)),
  };
}
