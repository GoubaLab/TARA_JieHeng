import { routeConnections, editableSegments } from "./connection-routing.js";
import { flowLabelPosition } from "./flow-label.js";
import {
  communicationCaption,
  communicationForDisplay,
  flowDashArray,
  flowColor,
  wirelessBadge,
  operatingSystemSummary,
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
export const isDiagramEndpoint = (node, graph) =>
  !!node &&
  (isEntity(node) ||
    (graph?.view_mode === "dfd" &&
      ["process_step", "data_store"].includes(node.kind)));
export const entityType = (node) =>
  node.entityType || (node.kind === "external" ? "external" : "ecu");
export const entityKindLabel = (node) =>
  node.kind === "process_step"
    ? "处理过程"
    : node.kind === "data_store"
      ? "数据存储"
      : node.kind === "boundary"
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
// Entity view, expanded DFD and exports share the platform's symbol palette.
export function nodeAppearance(node, selected = false) {
  const boundary = node.kind === "boundary";
  return {
    fill: boundary
      ? "#f4f8ff"
      : node.kind === "store"
        ? "#fffbe6"
        : node.kind === "data_store"
          ? "#f2f7ed"
          : node.kind === "process_step"
            ? "#f0f5ff"
            : entityType(node) === "external"
              ? "#fafafa"
              : entityType(node) === "service"
                ? "#f6f8ff"
                : "#ffffff",
    stroke: selected
      ? "#1677ff"
      : node.kind === "store"
        ? "#d4b106"
        : boundary
          ? "#91caff"
          : "#c9d2df",
    radius: boundary ? 12 : 8,
    dash: boundary ? "7 5" : "",
  };
}
export function nodeMarkup(
  node,
  { selected = false, interactive = true, dfd = false } = {},
) {
  if (dfd) return dfdNodeMarkup(node, { selected, interactive });
  const e = escapeXml,
    boundary = node.kind === "boundary",
    legacyStore = node.kind === "store",
    children = node.children || [],
    attributes = node.attributes || [],
    os = isEntity(node) ? operatingSystemSummary(node.operating_system) : "";
  const { fill, stroke } = nodeAppearance(node, selected);
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
        content += `<text x="${node.width / 2}" y="${node.height - 7}" text-anchor="middle" font-size="9" fill="#687281">另有 ${children.length - capacity} 项${interactive ? "，选中实体查看" : ""}</text>`;
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
  { selected = false, dragging = false, interactive = true, dfd = false } = {},
) {
  if (flow.show_label === false && !selected) return "";
  const p = flowLabelPosition(route.points, flow.labelOffset),
    { width, caption, wireless } = labelDimensions(flow, dfd),
    title = `${flow.name}${caption ? " · " + communicationCaption(flow) : ""}`;
  return `<g ${interactive ? `data-flow="${escapeXml(flow.id)}" data-flow-label="${escapeXml(flow.id)}"` : ""} class="flow-label-handle ${selected ? "selected" : ""} ${dragging ? "dragging" : ""}" transform="translate(${p.x} ${p.y})"><title>${escapeXml(title)}${interactive ? " · 拖动标签调整位置" : ""}</title><g transform="scale(${dfd ? 1.5 : 1})"><rect class="flow-label-background" x="${-width / 2}" y="${caption ? -22 : -13}" width="${width}" height="${caption ? 44 : 26}" rx="4" fill="#fff"/><text class="flow-label" text-anchor="middle" y="${caption ? -7 : 0}" dominant-baseline="central" font-size="12" style="fill:${flowColor(flow)}">${escapeXml(flow.name)}</text>${caption ? `${wireless ? wirelessBadge(flow, -textWidth(caption, 5, 10) / 2 - 16, 10) : ""}<text x="${wireless ? 12 : 0}" y="14" text-anchor="middle" font-size="10" fill="#687281">${escapeXml(caption)}</text>` : ""}</g></g>`;
}
const textWidth = (value, ascii = 7, other = 12) =>
  Array.from(value).reduce(
    (sum, char) => sum + (/[\u0000-\u00ff]/.test(char) ? ascii : other),
    0,
  );
export function labelDimensions(flow, compact = false) {
  const wireless = communicationForDisplay(flow).medium === "wireless",
    caption =
      compact && !wireless ? "" : truncate(communicationCaption(flow), 40);
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
    if (flow && flow.show_label !== false) {
      const p = flowLabelPosition(route.points, flow.labelOffset),
        dimensions = labelDimensions(flow, graph.view_mode === "dfd"),
        scale = graph.view_mode === "dfd" ? 1.5 : 1,
        half = (dimensions.width / 2) * scale,
        halfHeight = (dimensions.caption ? 22 : 13) * scale;
      points.push(
        { x: p.x - half, y: p.y - halfHeight },
        { x: p.x + half, y: p.y + halfHeight },
      );
    }
  }
  const minX = points.length ? Math.min(...points.map((p) => p.x)) - 35 : 0,
    minY = points.length ? Math.min(...points.map((p) => p.y)) - 35 : 0;
  const width = points.length
    ? Math.max(...points.map((p) => p.x)) - minX + 35
    : 640;
  const height = points.length
    ? Math.max(...points.map((p) => p.y)) - minY + 35
    : 360;
  const linePath = (route) =>
    pathForRoute
      ? pathForRoute(route, routes)
      : route.points.map((p, i) => `${i ? "L" : "M"} ${p.x} ${p.y}`).join(" ");
  const lines = routes
    .map((route) => {
      const flow = graph.flows.find((flow) => flow.id === route.id),
        dash = flow ? flowDashArray(flow) : "";
      return `<path d="${linePath(route)}" fill="none" stroke="${flowColor(flow)}" stroke-width="${graph.view_mode === "dfd" ? 2.5 : 1.6}" ${dash ? `stroke-dasharray="${dash}"` : ""} marker-end="url(#static-arrow${flow?.channel_kind === "energy" ? "-energy" : flow?.emphasis ? "-" + flow.emphasis : ""})"/>`;
    })
    .join("");
  const labels = routes
    .map((route) => {
      const flow = graph.flows.find((flow) => flow.id === route.id);
      return flow
        ? labelMarkup(flow, route, {
            interactive: false,
            dfd: graph.view_mode === "dfd",
          })
        : "";
    })
    .join("");
  return `<?xml version="1.0" encoding="UTF-8"?><svg xmlns="http://www.w3.org/2000/svg" width="${Math.ceil(width)}" height="${Math.ceil(height)}" viewBox="${minX} ${minY} ${width} ${height}" role="img" aria-label="实体与数据流静态图"><title>实体与数据流静态图</title><defs><marker id="static-arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse"><path d="M0 0L10 5L0 10Z" fill="#687281"/></marker>${["normal", "critical", "internal", "energy"].map((kind) => `<marker id="static-arrow-${kind}" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse"><path d="M0 0L10 5L0 10Z" fill="${flowColor(kind === "energy" ? { channel_kind: kind } : { emphasis: kind })}"/></marker>`).join("")}</defs><rect x="${minX}" y="${minY}" width="${width}" height="${height}" fill="#ffffff"/><g font-family="Arial,Microsoft YaHei,PingFang SC,sans-serif">${graph.nodes
    .filter((n) => n.kind === "boundary")
    .map((n) =>
      nodeMarkup(n, { interactive: false, dfd: graph.view_mode === "dfd" }),
    )
    .join("")}${lines}${graph.nodes
    .filter((n) => n.kind !== "boundary")
    .map((n) =>
      nodeMarkup(n, { interactive: false, dfd: graph.view_mode === "dfd" }),
    )
    .join("")}${labels}</g></svg>`;
}

function dfdNodeMarkup(node, { selected, interactive }) {
  const e = escapeXml,
    boundary = node.kind === "boundary",
    store = node.kind === "data_store";
  const { stroke, fill, radius, dash } = nodeAppearance(node, selected);
  const titleSize = Math.max(13, Math.min(24, node.width / 17)),
    detailSize = titleSize * 0.84,
    lineHeight = titleSize + 5;
  const title = wrapDetail(
    node.name,
    Math.max(10, (node.width - 24) / (titleSize / 2)),
  );
  const detail = boundary
    ? []
    : String(node.description || "")
        .split("\n")
        .slice(0, 2)
        .flatMap((line) =>
          wrapDetail(line, Math.max(10, (node.width - 24) / (detailSize / 2))),
        );
  const capacity = Math.max(
    1,
    Math.floor((node.height - 14 - title.length * lineHeight) / lineHeight),
  );
  const lines = [
    ...title.map((text) => ({ text, bold: true })),
    ...detail.slice(0, capacity).map((text) => ({ text, bold: false })),
  ];
  const y = boundary
    ? lineHeight + 8
    : Math.max(
        lineHeight,
        (node.height - lines.length * lineHeight) / 2 + titleSize,
      );
  const ports =
    interactive && !boundary
      ? [
          [node.width / 2, 0],
          [node.width, node.height / 2],
          [node.width / 2, node.height],
          [0, node.height / 2],
        ]
          .map(
            ([x, y], i) =>
              `<circle class="port" data-port="${["top", "right", "bottom", "left"][i]}" cx="${x}" cy="${y}" r="5"/>`,
          )
          .join("")
      : "";
  return `<g ${interactive ? `data-node="${e(node.id)}"` : ""} class="dfd-node ${selected ? "selected" : ""} ${boundary ? "boundary" : ""}" transform="translate(${node.x} ${node.y})"><title>${e(node.name)} · ${e(entityKindLabel(node))} · ${e(node.description)}</title><rect x="${store ? 6 : 0}" width="${node.width - (store ? 6 : 0)}" height="${node.height}" rx="${radius}" stroke="${stroke}" fill="${fill}" stroke-width="2.5" ${dash ? `stroke-dasharray="${dash}"` : ""}/>${store ? `<path d="M0 0V${node.height}" stroke="${stroke}" stroke-width="2"/>` : ""}${lines.map((line, i) => `<text x="${boundary ? 14 : node.width / 2}" y="${y + i * lineHeight}" text-anchor="${boundary ? "start" : "middle"}" font-size="${line.bold ? titleSize : detailSize}" font-weight="${line.bold ? 600 : 400}" fill="#253044">${e(line.text)}</text>`).join("")}${ports}</g>`;
}
