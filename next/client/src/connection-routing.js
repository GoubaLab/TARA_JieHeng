import "../public/lane-router.js";
const plan = globalThis.AssuranceLaneRouter.plan;
export const connectionSides = ["top", "right", "bottom", "left"];
const delta = { top: [0, -1], right: [1, 0], bottom: [0, 1], left: [-1, 0] };
const length = (points) =>
  points
    .slice(1)
    .reduce(
      (n, p, i) =>
        n + Math.abs(p.x - points[i].x) + Math.abs(p.y - points[i].y),
      0,
    );
const port = (n, side) => ({
  x: n.x + (side === "left" ? 0 : side === "right" ? n.width : n.width / 2),
  y: n.y + (side === "top" ? 0 : side === "bottom" ? n.height : n.height / 2),
});
export const snapRoutePoint = (p) => ({
  x: Math.max(-10000, Math.min(10000, Math.round(p.x / 10) * 10)),
  y: Math.max(-10000, Math.min(10000, Math.round(p.y / 10) * 10)),
});
export const flowWaypoint = (flow) =>
  Object.hasOwn(flow, "route_layout")
    ? flow.route_layout.waypoint
    : flow.wireless_layout?.waypoint;
export const flowSide = (flow, role) =>
  Object.hasOwn(flow, role + "Side")
    ? flow[role + "Side"]
    : flow.wireless_layout?.[role + "_side"];

// Projection is always onto the original entity. It never changes a business endpoint.
export function attachmentAt(node, point, tolerance = 36) {
  const clamp = (v, a, b) => Math.max(a, Math.min(b, v));
  return (
    connectionSides
      .map((side) => {
        const p = port(node, side);
        if (side === "top" || side === "bottom")
          p.x = clamp(point.x, node.x, node.x + node.width);
        else p.y = clamp(point.y, node.y, node.y + node.height);
        return {
          side,
          point: p,
          distance: Math.hypot(p.x - point.x, p.y - point.y),
        };
      })
      .filter((p) => p.distance <= tolerance)
      .sort((a, b) => a.distance - b.distance)[0] || null
  );
}
export function editableSegments(points) {
  // A straight two-point line may also be dragged to create a detour. Endpoint
  // handles take priority and cannot be detached by dragging the line itself.
  return points.slice(1).flatMap((b, i) => {
    const a = points[i];
    if (Math.abs(a.x - b.x) + Math.abs(a.y - b.y) < 32) return [];
    return [
      {
        index: i,
        point: { x: a.x + (b.x - a.x) * 0.25, y: a.y + (b.y - a.y) * 0.25 },
        axis: a.y === b.y ? "y" : "x",
      },
    ];
  });
}
let cachedKey, cachedResult;
export function routeConnections(graph) {
  const nodes = graph.nodes.map((n) => ({
    ...n,
    boundary: n.kind === "boundary",
  }));
  const byId = new Map(nodes.map((n) => [n.id, n]));
  const grid = graph.equal_spacing === false ? 10 : 20;
  const variants = [...graph.flows]
    .sort((a, b) => a.id.localeCompare(b.id))
    .map((flow) => {
      const a = byId.get(flow.source),
        b = byId.get(flow.target),
        waypoint = flowWaypoint(flow);
      const base = {
        id: flow.id,
        source: flow.source,
        target: flow.target,
        waypoints: waypoint ? [waypoint] : [],
      };
      if (!a || !b) return [base];
      const choices = [];
      for (const sourceSide of flowSide(flow, "source")
        ? [flowSide(flow, "source")]
        : connectionSides)
        for (const targetSide of flowSide(flow, "target")
          ? [flowSide(flow, "target")]
          : connectionSides) {
          const start = port(a, sourceSide),
            end = port(b, targetSide);
          const escape = (p, side) => ({
            x: p.x + delta[side][0] * grid,
            y: p.y + delta[side][1] * grid,
          });
          choices.push({
            ...base,
            sourceSide,
            targetSide,
            estimate: length([
              start,
              escape(start, sourceSide),
              ...base.waypoints,
              escape(end, targetSide),
              end,
            ]),
          });
        }
      return choices.sort(
        (a, b) =>
          a.estimate - b.estimate ||
          a.sourceSide.localeCompare(b.sourceSide) ||
          a.targetSide.localeCompare(b.targetSide),
      );
    });
  const key = JSON.stringify([
    nodes.map(({ id, x, y, width, height, boundary }) => ({
      id,
      x,
      y,
      width,
      height,
      boundary,
    })),
    variants,
    grid,
  ]);
  if (key === cachedKey) return cachedResult;
  let choices = variants.map((v) => v[0]),
    attempts = 0;
  const run = (edges, avoidCrossings = true) => {
    attempts++;
    return plan({ nodes, edges, grid, avoidCrossings });
  };
  const score = (r) =>
    r.conflicts.length * 1e8 -
    r.routes.length * 1e9 +
    r.routes.reduce(
      (s, r) => s + length(r.points) + Math.max(0, r.points.length - 2) * 24,
      0,
    );
  let result = run(choices),
    best = score(result);
  // Evaluate nearby alternatives, including a feasible alternative to an
  // obstructed automatic port. Explicitly pinned sides are never overridden.
  for (let i = 0; i < variants.length && attempts < 20; i++)
    for (const variant of variants[i].slice(1, 9)) {
      if (attempts >= 20) break;
      const next = choices.map((c, j) => (j === i ? variant : c)),
        candidate = run(next),
        cost = score(candidate);
      if (cost < best) {
        choices = next;
        result = candidate;
        best = cost;
      }
    }
  if (result.conflicts.length) {
    // Non-planar graphs may require a crossing. Use a bridge, never hide a flow
    // or overlap entities merely to claim a crossing-free result.
    const fallback = run(choices, false);
    if (fallback.routes.length > result.routes.length) result = fallback;
  }
  cachedKey = key;
  cachedResult = result;
  return result;
}
