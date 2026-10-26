import { routeConnections, editableSegments } from "./connection-routing.js";
import { flowLabelPosition } from "./flow-label.js";
import {
  communicationCaption,
  communicationDetails,
  communicationForDisplay,
  flowDashArray,
  wirelessBadge,
  isWirelessFlow,
  hasOperatingSystem,
  operatingSystemSummary,
  osFamilies,
} from "./communication.js";

export const entityTypes = {
  device: "设备",
  application: "应用",
  cloud: "云服务",
  actor: "参与者",
  ecu: "控制实体（ECU）",
  external: "外部实体",
  service: "服务实体",
};
export const isEntity = (node) =>
  ["entity", "process", "external"].includes(node.kind);
export const entityType = (node) =>
  node.entityType || (node.kind === "external" ? "external" : "ecu");
export const entityKindLabel = (node) =>
  node.kind === "boundary"
    ? "信任边界"
    : node.kind === "store"
      ? "待归属数据存储"
      : entityTypes[entityType(node)];
export const escapeXml = (value) =>
  String(value ?? "").replace(
    /[&<>"']/g,
    (char) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[
        char
      ],
  );
export function normalizedArchitecture(graph) {
  const copy = structuredClone(graph);
  copy.nodes = copy.nodes.map((node) =>
    isEntity(node)
      ? {
          ...node,
          kind: "entity",
          entityType: entityType(node),
          children: node.children || [],
          attributes: node.attributes || [],
        }
      : node,
  );
  return copy;
}
export function assignLegacyStore(graph, storeId, parentId) {
  const copy = normalizedArchitecture(graph),
    store = copy.nodes.find((node) => node.id === storeId),
    parent = copy.nodes.find((node) => node.id === parentId);
  if (!store || store.kind !== "store" || !parent || !isEntity(parent))
    throw new Error("请选择待归属的数据存储和目标实体。");
  for (const flow of copy.flows) {
    const source = flow.source === storeId ? parentId : flow.source,
      target = flow.target === storeId ? parentId : flow.target;
    if (source === target)
      throw new Error(
        `数据流“${flow.name}”归属后将位于同一实体内。请先明确处理该连线，再设置归属。`,
      );
    if (flow.source === storeId) {
      flow.source = parentId;
      flow.sourceSide = null;
    }
    if (flow.target === storeId) {
      flow.target = parentId;
      flow.targetSide = null;
    }
  }
  parent.children.push({
    id: store.id,
    name: store.name,
    kind: "store",
    description: store.description || "",
    reference: store.reference || null,
  });
  parent.width = Math.max(parent.width, 240);
  parent.height = Math.min(
    3000,
    Math.max(parent.height, 84 + parent.children.length * 26),
  );
  copy.nodes = copy.nodes.filter((node) => node.id !== storeId);
  return copy;
}
const truncate = (text, length = 25) =>
  Array.from(String(text)).length > length
    ? Array.from(String(text)).slice(0, length).join("") + "…"
    : String(text);
export function nodeMarkup(
  node,
  { selected = false, interactive = true } = {},
) {
  const e = escapeXml,
    boundary = node.kind === "boundary",
    legacyStore = node.kind === "store",
    type = entityType(node),
    children = node.children || [],
    attributes = node.attributes || [],
    os = isEntity(node) ? operatingSystemSummary(node.operating_system) : "";
  const fill = boundary
    ? "#f4f8ff"
    : legacyStore
      ? "#fffbe6"
      : type === "external"
        ? "#fafafa"
        : type === "service"
          ? "#f6f8ff"
          : "#ffffff";
  const stroke = selected
    ? "#1677ff"
    : legacyStore
      ? "#d4b106"
      : boundary
        ? "#91caff"
        : "#c9d2df";
  const identity = interactive ? `data-node="${e(node.id)}"` : "";
  const titleY = boundary
    ? 24
    : children.length || os
      ? 25
      : node.height / 2 - 4;
  let content = `<title>${e(node.name)} · ${e(entityKindLabel(node))}</title><rect width="${node.width}" height="${node.height}" rx="${boundary ? 12 : 8}" fill="${fill}" stroke="${stroke}" ${boundary ? 'stroke-dasharray="7 5"' : ""}/><text x="${boundary ? 15 : node.width / 2}" y="${titleY}" text-anchor="${boundary ? "start" : "middle"}" fill="#1f2329" font-size="12" font-weight="600">${e(truncate(node.name, Math.max(10, Math.floor(node.width / 13))))}</text>`;
  if (!boundary) {
    content += `<text x="${node.width / 2}" y="${titleY + 18}" text-anchor="middle" font-size="10" fill="${legacyStore ? "#ad8b00" : "#687281"}">${e(entityKindLabel(node))}${attributes.length ? " · " + attributes.length + " 项属性" : ""}</text>`;
    if (os && node.height >= 72)
      content += `<text x="${node.width / 2}" y="63" text-anchor="middle" font-size="10" fill="#536176"><title>${e(os)}</title>${e(truncate("OS · " + os, Math.max(8, Math.floor(node.width / 11))))}</text>`;
    if (children.length) {
      const offset = os ? 23 : 0;
      content += `<path d="M12 ${54 + offset}H${node.width - 12}" stroke="#e5e9f0"/>`;
      const capacity = Math.max(
        0,
        Math.floor((node.height - 70 - offset) / 26),
      );
      children.slice(0, capacity).forEach((child, index) => {
        const y = 66 + offset + index * 26;
        content += `<g class="entity-child"><title>${e(child.name)}${child.description ? " · " + e(child.description) : ""}</title><rect x="12" y="${y - 3}" width="${node.width - 24}" height="23" rx="3" fill="${child.kind === "store" ? "#f2f7ed" : "#f0f5ff"}"/><text x="20" y="${y + 12}" fill="#536176" font-size="10">${child.kind === "store" ? "存储" : "软件"}</text><text x="54" y="${y + 12}" fill="#303c4b" font-size="10">${e(truncate(child.name, Math.max(5, Math.floor((node.width - 70) / 10))))}</text></g>`;
      });
      if (capacity < children.length)
        content += `<text x="${node.width / 2}" y="${node.height - 7}" text-anchor="middle" font-size="9" fill="#687281">另有 ${children.length - capacity} 项${interactive ? "，选中实体查看" : "，详见下方明细"}</text>`;
    }
    if (interactive && isEntity(node))
      content += [
        [node.width / 2, 0],
        [node.width, node.height / 2],
        [node.width / 2, node.height],
        [0, node.height / 2],
      ]
        .map(
          ([x, y], index) =>
            `<circle class="port" data-port="${["top", "right", "bottom", "left"][index]}" cx="${x}" cy="${y}" r="5"/>`,
        )
        .join("");
  }
  return `<g ${identity} class="dfd-node ${selected ? "selected" : ""} ${boundary ? "boundary" : ""} ${legacyStore ? "legacy-store" : ""}" transform="translate(${node.x} ${node.y})">${content}</g>`;
}
export function labelMarkup(
  flow,
  route,
  { selected = false, dragging = false, interactive = true } = {},
) {
  const p = flowLabelPosition(route.points, flow.labelOffset),
    { width, caption, wireless } = labelDimensions(flow),
    title = `${flow.name}${caption ? " · " + communicationCaption(flow) : ""}`;
  return `<g ${interactive ? `data-flow="${escapeXml(flow.id)}" data-flow-label="${escapeXml(flow.id)}"` : ""} class="flow-label-handle ${selected ? "selected" : ""} ${dragging ? "dragging" : ""}" transform="translate(${p.x} ${p.y})"><title>${escapeXml(title)}${interactive ? " · 拖动标签调整位置" : ""}</title><rect class="flow-label-background" x="${-width / 2}" y="${caption ? -22 : -13}" width="${width}" height="${caption ? 44 : 26}" rx="4" fill="#fff"/><text class="flow-label" text-anchor="middle" y="${caption ? -7 : 0}" dominant-baseline="central" font-size="12" fill="#434b57">${escapeXml(flow.name)}</text>${caption ? `${wireless ? wirelessBadge(flow, -textWidth(caption, 5, 10) / 2 - 16, 10) : ""}<text x="${wireless ? 12 : 0}" y="14" text-anchor="middle" font-size="10" fill="#687281">${escapeXml(caption)}</text>` : ""}</g>`;
}
const textWidth = (value, ascii = 7, other = 12) =>
  Array.from(value).reduce(
    (sum, char) => sum + (/[\u0000-\u00ff]/.test(char) ? ascii : other),
    0,
  );
export function labelDimensions(flow) {
  const caption = truncate(communicationCaption(flow), 40),
    wireless = communicationForDisplay(flow).medium === "wireless";
  return {
    caption,
    wireless,
    width: Math.max(
      32,
      textWidth(flow.name) + 16,
      textWidth(caption, 5, 10) + (wireless ? 52 : 16),
    ),
  };
}
// Handles are view controls, never entities or business endpoints.
export function connectionHandles(flow, route) {
  if (!flow || !route?.points.length) return "";
  const e = escapeXml,
    points = route.points;
  const endpoints = ["source", "target"]
    .map((role) => {
      const p = role === "source" ? points[0] : points.at(-1);
      return `<g data-flow="${e(flow.id)}" data-flow-end="${role}" class="connection-end" role="button" tabindex="0" aria-label="${role === "source" ? "发送端" : "接收端箭头"}：拖到所属实体另一侧换边"><title>拖到所属实体的另一侧换边；不会更换实体</title><circle cx="${p.x}" cy="${p.y}" r="8" fill="#fff" stroke="#1677ff" stroke-width="2"/><circle cx="${p.x}" cy="${p.y}" r="2.5" fill="#1677ff" pointer-events="none"/></g>`;
    })
    .join("");
  const segments = editableSegments(points)
    .map(
      ({ index, point, axis }) =>
        `<g data-flow="${e(flow.id)}" data-flow-segment="${index}" class="connection-segment" style="cursor:${axis === "y" ? "ns-resize" : "ew-resize"}"><title>拖动线段调整途经位置；松手自动对齐</title><rect x="${point.x - 5}" y="${point.y - 5}" width="10" height="10" rx="2" fill="#fff" stroke="#1677ff" stroke-width="1.5"/></g>`,
    )
    .join("");
  return segments + endpoints;
}

function wrapDetail(text, units) {
  const lines = [];
  for (const paragraph of String(text).split("\n")) {
    let current = "",
      width = 0;
    for (const char of Array.from(paragraph)) {
      const step = /[\u0000-\u00ff]/.test(char) ? 1 : 2;
      if (current && width + step > units) {
        lines.push(current);
        current = "";
        width = 0;
      }
      current += char;
      width += step;
    }
    lines.push(current);
  }
  return lines;
}
export function staticDiagramSvg(graph, routes, pathForRoute) {
  const flowById = new Map(graph.flows.map((flow) => [flow.id, flow]));
  if (
    routes.length !== graph.flows.length ||
    routes.some((r) => !flowById.has(r.id))
  ) {
    const result = routeConnections(graph);
    if (result.conflicts.length)
      throw new Error("存在布线冲突，请调整实体间距或整理连线后再导出。");
    routes = result.routes;
  }
  const points = graph.nodes.flatMap((node) => [
    { x: node.x, y: node.y },
    { x: node.x + node.width, y: node.y + node.height },
  ]);
  for (const route of routes) {
    points.push(...route.points);
    const flow = graph.flows.find((flow) => flow.id === route.id);
    if (flow) {
      const p = flowLabelPosition(route.points, flow.labelOffset),
        half = labelDimensions(flow).width / 2;
      points.push(
        { x: p.x - half, y: p.y - 26 },
        { x: p.x + half, y: p.y + 26 },
      );
    }
  }
  const minX = points.length ? Math.min(...points.map((p) => p.x)) - 35 : 0,
    minY = points.length ? Math.min(...points.map((p) => p.y)) - 35 : 0;
  const detailEntities = graph.nodes.filter(
    (node) =>
      isEntity(node) &&
      ((node.children || []).length ||
        (node.attributes || []).length ||
        node.description ||
        hasOperatingSystem(node.operating_system)),
  );
  const width = points.length
    ? Math.max(
        detailEntities.length || graph.flows.length ? 640 : 320,
        Math.max(...points.map((p) => p.x)) - minX + 35,
      )
    : 640;
  const diagramHeight = points.length
    ? Math.max(200, Math.max(...points.map((p) => p.y)) - minY + 35)
    : 360;
  const detailRows = [];
  const append = (text, header = false) =>
    wrapDetail(text, Math.max(20, Math.floor((width - 70) / 6))).forEach(
      (line) => detailRows.push({ text: line, header }),
    );
  for (const entity of detailEntities) {
    append(entity.name + " · " + entityKindLabel(entity), true);
    if (entity.description) append("实体说明：" + entity.description);
    if (hasOperatingSystem(entity.operating_system)) {
      const os = entity.operating_system;
      append(
        "操作系统类别：" + (osFamilies[os.family || "unknown"] || os.family),
      );
      for (const [key, title] of [
        ["name", "操作系统名称"],
        ["version", "操作系统版本"],
        ["architecture", "处理器架构"],
        ["notes", "操作系统备注"],
      ])
        if (os[key]) append(title + "：" + os[key]);
    }
    for (const attribute of entity.attributes || [])
      append("属性 · " + attribute.name + "：" + attribute.value);
    for (const child of entity.children || []) {
      append(
        (child.kind === "store" ? "数据存储" : "软件") + " · " + child.name,
      );
      if (child.description) append("说明：" + child.description);
    }
    detailRows.push({ text: "" });
  }
  if (graph.flows.length) {
    append("数据流与通信明细", true);
    for (const flow of graph.flows) {
      const nodeName = (id) =>
        graph.nodes.find((node) => node.id === id)?.name || id;
      append(
        flow.name +
          " · " +
          nodeName(flow.source) +
          " → " +
          nodeName(flow.target),
        true,
      );
      communicationDetails(flow).forEach((text) => append(text));
      detailRows.push({ text: "" });
    }
  }
  const detailHeight = detailRows.length ? 65 + detailRows.length * 19 : 0,
    height = diagramHeight + detailHeight;
  const linePath = (route) =>
    pathForRoute
      ? pathForRoute(route, routes)
      : route.points.map((p, i) => `${i ? "L" : "M"} ${p.x} ${p.y}`).join(" ");
  const lines = routes
    .map((route) => {
      const flow = graph.flows.find((flow) => flow.id === route.id),
        dash = flow ? flowDashArray(flow) : "";
      return `<path d="${linePath(route)}" fill="none" stroke="#687281" stroke-width="1.6" ${dash ? `stroke-dasharray="${dash}"` : ""} marker-end="url(#static-arrow)"/>`;
    })
    .join("");
  const labels = routes
    .map((route) => {
      const flow = graph.flows.find((flow) => flow.id === route.id);
      return flow ? labelMarkup(flow, route, { interactive: false }) : "";
    })
    .join("");
  const details = detailRows.length
    ? `<path d="M${minX + 18} ${minY + diagramHeight}H${minX + width - 18}" stroke="#e5e8ee"/><text x="${minX + 18}" y="${minY + diagramHeight + 27}" font-size="13" font-weight="600" fill="#1f2329">实体属性与内部组成明细</text>${detailRows.map((row, index) => `<text x="${minX + 22}" y="${minY + diagramHeight + 52 + index * 19}" font-size="11" font-weight="${row.header ? "600" : "400"}" fill="${row.header ? "#1f2329" : "#536176"}">${escapeXml(row.text)}</text>`).join("")}`
    : "";
  return `<?xml version="1.0" encoding="UTF-8"?><svg xmlns="http://www.w3.org/2000/svg" width="${Math.ceil(width)}" height="${Math.ceil(height)}" viewBox="${minX} ${minY} ${width} ${height}" role="img" aria-label="实体与数据流静态图"><title>实体与数据流静态图</title><defs><marker id="static-arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse"><path d="M0 0L10 5L0 10Z" fill="#687281"/></marker></defs><rect x="${minX}" y="${minY}" width="${width}" height="${height}" fill="#ffffff"/><g font-family="Arial,Microsoft YaHei,PingFang SC,sans-serif">${graph.nodes
    .filter((n) => n.kind === "boundary")
    .map((n) => nodeMarkup(n, { interactive: false }))
    .join("")}${lines}${graph.nodes
    .filter((n) => n.kind !== "boundary")
    .map((n) => nodeMarkup(n, { interactive: false }))
    .join("")}${labels}${details}</g></svg>`;
}
