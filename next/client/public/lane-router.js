/* Geometry-only DFD routing. No graph, persistence, or business records are mutated. */
(function (root, factory) {
  'use strict';
  const api = factory();
  if (typeof module === 'object' && module.exports) module.exports = api;
  else root.AssuranceLaneRouter = api;
})(typeof window !== 'undefined' ? window : globalThis, function () {
  'use strict';
  const EPS = 1e-6;
  const SIDES = ['top', 'right', 'bottom', 'left'];
  const DELTA = { top: [0, -1], right: [1, 0], bottom: [0, 1], left: [-1, 0] };
  const cmp = (a, b) => String(a) < String(b) ? -1 : String(a) > String(b) ? 1 : 0;
  const round = (n) => Math.round(n * 1e6) / 1e6;
  const center = (n) => ({ x: n.x + n.width / 2, y: n.y + n.height / 2 });

  function sideToward(node, other) {
    const a = center(node), b = center(other);
    const dx = b.x - a.x, dy = b.y - a.y;
    return Math.abs(dx) / ((node.width + other.width) / 2) >= Math.abs(dy) / ((node.height + other.height) / 2)
      ? (dx >= 0 ? 'right' : 'left') : (dy >= 0 ? 'bottom' : 'top');
  }

  function pointOn(node, side, ratio) {
    if (side === 'left' || side === 'right') return { x: round(node.x + (side === 'right' ? node.width : 0)), y: round(node.y + node.height * ratio) };
    return { x: round(node.x + node.width * ratio), y: round(node.y + (side === 'bottom' ? node.height : 0)) };
  }

  function segment(a, b, edgeId) {
    const horizontal = Math.abs(a.y - b.y) < EPS;
    return { horizontal, position: horizontal ? a.y : a.x,
      low: Math.min(horizontal ? a.x : a.y, horizontal ? b.x : b.y),
      high: Math.max(horizontal ? a.x : a.y, horizontal ? b.x : b.y), edgeId };
  }
  function overlap(a, b) { return Math.min(a.high, b.high) - Math.max(a.low, b.low) > EPS; }
  function parallelConflict(a, b, grid) {
    return a.edgeId !== b.edgeId && a.horizontal === b.horizontal && overlap(a, b) && Math.abs(a.position - b.position) < grid - EPS;
  }
  function crosses(a, b) {
    return a.horizontal !== b.horizontal && a.position > b.low + EPS && a.position < b.high - EPS && b.position > a.low + EPS && b.position < a.high - EPS;
  }
  function blocks(segment, rect) {
    return segment.horizontal
      ? segment.position > rect.top + EPS && segment.position < rect.bottom - EPS && segment.high > rect.left + EPS && segment.low < rect.right - EPS
      : segment.position > rect.left + EPS && segment.position < rect.right - EPS && segment.high > rect.top + EPS && segment.low < rect.bottom - EPS;
  }
  function simplify(points) {
    const result = [];
    for (const point of points) {
      const p = { x: round(point.x), y: round(point.y) };
      const last = result[result.length - 1];
      if (last && Math.abs(last.x - p.x) < EPS && Math.abs(last.y - p.y) < EPS) continue;
      if (result.length >= 2) {
        const first = result[result.length - 2];
        if ((Math.abs(first.x - last.x) < EPS && Math.abs(last.x - p.x) < EPS) ||
            (Math.abs(first.y - last.y) < EPS && Math.abs(last.y - p.y) < EPS)) result.pop();
      }
      result.push(p);
    }
    return result;
  }
  function segments(points, edgeId) {
    return points.slice(1).map((p, i) => segment(points[i], p, edgeId)).filter((s) => s.high - s.low > EPS);
  }

  class Heap {
    constructor() { this.items = []; }
    push(value) {
      const a = this.items; a.push(value); let i = a.length - 1;
      while (i > 0) { const p = (i - 1) >> 1; if (a[p].f <= value.f) break; a[i] = a[p]; i = p; }
      a[i] = value;
    }
    pop() {
      const a = this.items, answer = a[0], tail = a.pop();
      if (a.length) {
        let i = 0;
        while (i * 2 + 1 < a.length) {
          let c = i * 2 + 1;
          if (c + 1 < a.length && a[c + 1].f < a[c].f) c++;
          if (tail.f <= a[c].f) break;
          a[i] = a[c]; i = c;
        }
        a[i] = tail;
      }
      return answer;
    }
    get length() { return this.items.length; }
  }

  function buildGrid(nodes, edges, rects, spacing) {
    const margin = spacing * Math.max(5, Math.ceil(edges.length / 2) + 2);
    const xs = new Set(), ys = new Set();
    const left = Math.floor((Math.min(...nodes.map((n) => n.x)) - margin) / spacing) * spacing;
    const top = Math.floor((Math.min(...nodes.map((n) => n.y)) - margin) / spacing) * spacing;
    const right = Math.ceil((Math.max(...nodes.map((n) => n.x + n.width)) + margin) / spacing) * spacing;
    const bottom = Math.ceil((Math.max(...nodes.map((n) => n.y + n.height)) + margin) / spacing) * spacing;
    if ((1 + (right - left) / spacing) * (1 + (bottom - top) / spacing) > 100000) return { count: Infinity };
    for (let x = left; x <= right + EPS; x += spacing) xs.add(round(x));
    for (let y = top; y <= bottom + EPS; y += spacing) ys.add(round(y));
    for (const rect of rects) { xs.add(rect.left); xs.add(rect.right); ys.add(rect.top); ys.add(rect.bottom); }
    for (const edge of edges) for (const end of [edge.source, edge.target]) {
      xs.add(end.point.x); ys.add(end.point.y); xs.add(end.escape.x); ys.add(end.escape.y);
    }
    const x = [...xs].sort((a, b) => a - b), y = [...ys].sort((a, b) => a - b);
    const xIndex = new Map(x.map((v, i) => [v, i])), yIndex = new Map(y.map((v, i) => [v, i]));
    const width = x.length, count = width * y.length;
    if (count > 100000) return { count };
    // Cache obstacle checks once. The per-edge cache below adds lane reservations.
    const base = new Uint8Array(count * 4);
    function point(index) { return { x: x[index % width], y: y[Math.floor(index / width)] }; }
    function adjacent(index, direction) {
      const ix = index % width, iy = Math.floor(index / width);
      if (direction === 0) return ix + 1 < width ? index + 1 : -1;
      if (direction === 1) return iy + 1 < y.length ? index + width : -1;
      if (direction === 2) return ix > 0 ? index - 1 : -1;
      return iy > 0 ? index - width : -1;
    }
    function canMove(index, direction, next, line) {
      const key = index * 4 + direction;
      if (!base[key]) {
        const ok = next >= 0 && !rects.some((rect) => blocks(line, rect));
        base[key] = ok ? 1 : 2;
        if (next >= 0) base[next * 4 + ((direction + 2) % 4)] = base[key];
      }
      return base[key] === 1;
    }
    return { x, y, width, count, point, adjacent, canMove,
      index: (p) => yIndex.get(p.y) * width + xIndex.get(p.x) };
  }

  function findPath(edge, graph, reservations, spacing) {
    const start = graph.index(edge.source.escape), goal = graph.index(edge.target.escape);
    const initialAxis = edge.source.side === 'left' || edge.source.side === 'right' ? 0 : 1;
    const startState = start * 2 + initialAxis;
    const costs = new Float64Array(graph.count * 2); costs.fill(Infinity); costs[startState] = 0;
    const parent = new Int32Array(graph.count * 2); parent.fill(-1);
    const moveCache = new Int8Array(graph.count * 4);
    const crossingCost = new Float64Array(graph.count * 4);
    const heap = new Heap(), target = edge.target.escape;
    const heuristic = (p) => Math.abs(target.x - p.x) + Math.abs(target.y - p.y);
    heap.push({ state: startState, g: 0, f: heuristic(edge.source.escape) });
    const limit = Math.max(10000, graph.count * 2);
    let expanded = 0;
    while (heap.length && expanded++ < limit) {
      const item = heap.pop(), old = item.state;
      if (item.g > costs[old] + EPS) continue;
      const index = old >> 1, axis = old & 1;
      if (index === goal) {
        const path = []; let s = old;
        while (s !== -1) { path.push(graph.point(s >> 1)); s = parent[s]; }
        path.reverse();
        return simplify([edge.source.point, ...path, edge.target.point]);
      }
      const a = graph.point(index);
      for (let direction = 0; direction < 4; direction++) {
        const next = graph.adjacent(index, direction);
        if (next < 0) continue;
        const key = index * 4 + direction, b = graph.point(next);
        const line = segment(a, b, edge.id);
        if (!graph.canMove(index, direction, next, line)) continue;
        if (!moveCache[key]) {
          let valid = true, crossed = 0;
          for (const reserved of reservations) {
            if (reserved.edgeId === edge.id) continue;
            if (parallelConflict(line, reserved, spacing)) { valid = false; break; }
            if (crosses(line, reserved)) crossed++;
          }
          moveCache[key] = valid ? 1 : -1; crossingCost[key] = crossed * spacing * 0.4;
          const reverse = next * 4 + ((direction + 2) % 4);
          moveCache[reverse] = moveCache[key]; crossingCost[reverse] = crossingCost[key];
        }
        if (moveCache[key] === -1) continue;
        const nextAxis = direction % 2, nextState = next * 2 + nextAxis;
        const cost = item.g + line.high - line.low + (axis === nextAxis ? 0 : spacing * 0.8) + crossingCost[key];
        if (cost >= costs[nextState] - EPS) continue;
        costs[nextState] = cost; parent[nextState] = old;
        heap.push({ state: nextState, g: cost, f: cost + heuristic(b) });
      }
    }
    return null;
  }

  function plan(input = {}) {
    const conflicts = [], warnings = [];
    const grid = Number(input.grid === undefined ? 20 : input.grid);
    const failure = (message, extra = {}) => conflicts.push({ code: 'INVALID_INPUT', message, ...extra });
    if (!Number.isFinite(grid) || grid <= 0) return { routes: [], conflicts: [{ code: 'INVALID_INPUT', message: '网格间距必须为正数。' }], warnings };
    const nodes = (input.nodes || []).map((n) => ({ ...n, id: String(n.id) }));
    const byId = new Map();
    for (const node of nodes) {
      if (byId.has(node.id)) failure('组件编号重复。', { nodeId: node.id });
      if (![node.x, node.y, node.width, node.height].every(Number.isFinite) || node.width <= 0 || node.height <= 0) failure('组件位置或尺寸无效。', { nodeId: node.id });
      byId.set(node.id, node);
    }
    const edges = [], edgeIds = new Set(), groups = new Map();
    for (const original of input.edges || []) {
      const id = String(original.id), sourceId = String(original.source), targetId = String(original.target);
      if (edgeIds.has(id)) { failure('连线编号重复。', { edgeId: id }); continue; }
      edgeIds.add(id);
      const a = byId.get(sourceId), b = byId.get(targetId);
      if (!a || !b || a.boundary || b.boundary || sourceId === targetId) { failure('连线必须连接两个不同的有效组件。', { edgeId: id }); continue; }
      if (['sourceSide', 'targetSide'].some((key) => original[key] != null && !SIDES.includes(original[key]))) {
        failure('连接侧必须为 top、right、bottom 或 left。', { edgeId: id }); continue;
      }
      const edge = { id, source: { node: a, side: SIDES.includes(original.sourceSide) ? original.sourceSide : sideToward(a, b) },
        target: { node: b, side: SIDES.includes(original.targetSide) ? original.targetSide : sideToward(b, a) } };
      edges.push(edge);
      for (const terminal of ['source', 'target']) {
        const end = edge[terminal], other = terminal === 'source' ? edge.target : edge.source;
        const key = JSON.stringify([end.node.id, end.side]);
        if (!groups.has(key)) groups.set(key, []);
        groups.get(key).push({ edge, terminal, other: other.node });
      }
    }
    if (conflicts.length || !edges.length) return { routes: [], conflicts, warnings };
    for (const attachments of groups.values()) {
      const end = attachments[0].edge[attachments[0].terminal];
      const vertical = end.side === 'left' || end.side === 'right';
      attachments.sort((a, b) => {
        const ca = center(a.other), cb = center(b.other);
        return (vertical ? ca.y - cb.y : ca.x - cb.x) || cmp(a.other.id, b.other.id) || cmp(a.edge.id, b.edge.id);
      });
      const size = vertical ? end.node.height : end.node.width;
      if (attachments.length > 1 && size / (attachments.length + 1) < grid - EPS) {
        conflicts.push({ code: 'SIDE_CAPACITY', nodeId: end.node.id, side: end.side, count: attachments.length,
          availableSize: size, requiredSize: grid * (attachments.length + 1),
          message: `${end.node.id} 的 ${end.side} 侧需至少 ${grid * (attachments.length + 1)} 的边长，才能等分并保留一格间距。` });
      }
      attachments.forEach(({ edge, terminal }, index) => {
        const e = edge[terminal]; e.ratio = (index + 1) / (attachments.length + 1); e.point = pointOn(e.node, e.side, e.ratio);
        const d = DELTA[e.side]; e.escape = { x: round(e.point.x + d[0] * grid), y: round(e.point.y + d[1] * grid) };
      });
    }
    if (conflicts.length) return { routes: [], conflicts, warnings };
    const obstacles = nodes.filter((n) => !n.boundary);
    const rects = obstacles.map((n) => ({ id: n.id, left: round(n.x - grid / 2), right: round(n.x + n.width + grid / 2), top: round(n.y - grid / 2), bottom: round(n.y + n.height + grid / 2) }));
    const stubs = edges.flatMap((edge) => [segment(edge.source.point, edge.source.escape, edge.id), segment(edge.target.escape, edge.target.point, edge.id)]);
    for (const edge of edges) for (const end of [edge.source, edge.target]) {
      const stub = segment(end.point, end.escape, edge.id);
      if (rects.some((rect) => rect.id !== end.node.id && blocks(stub, rect))) conflicts.push({ code: 'PORT_BLOCKED', edgeId: edge.id, nodeId: end.node.id, side: end.side, message: '组件过近，连接点的引出通道被其他组件挡住。' });
    }
    for (let i = 0; i < stubs.length; i++) for (let j = i + 1; j < stubs.length; j++) {
      if (parallelConflict(stubs[i], stubs[j], grid)) conflicts.push({ code: 'PORT_CLEARANCE', edgeIds: [stubs[i].edgeId, stubs[j].edgeId], message: '不同组件的端口引出段不足一格间距，请拉开组件或调整连接侧。' });
    }
    if (conflicts.length) return { routes: [], conflicts, warnings };
    const graph = buildGrid(obstacles, edges, rects, grid);
    if (graph.count > 100000) return { routes: [], conflicts: [{ code: 'LAYOUT_TOO_LARGE', message: '画布跨度过大，请缩小组件间距离后重试。' }], warnings };
    const length = (e) => Math.abs(e.source.escape.x - e.target.escape.x) + Math.abs(e.source.escape.y - e.target.escape.y);
    let order = [...edges].sort((a, b) => length(a) - length(b) || cmp(a.id, b.id));
    let best = [], failures = [];
    for (let attempt = 0; attempt < 3; attempt++) {
      const reserved = [...stubs], result = [], failed = [];
      for (const edge of order) {
        const points = findPath(edge, graph, reserved, grid);
        if (!points) { failed.push({ code: 'NO_CLEAR_ROUTE', edgeId: edge.id, message: '未找到同时满足避障和一格线距的路线，请拉开组件或调整连接侧。' }); continue; }
        const terminal = (e) => ({ side: e.side, ratio: e.ratio, point: { ...e.point } });
        result.push({ id: edge.id, source: terminal(edge.source), target: terminal(edge.target), points });
        reserved.push(...segments(points, edge.id));
      }
      if (!failed.length) return { routes: result.sort((a, b) => cmp(a.id, b.id)), conflicts: [], warnings };
      if (!failures.length || result.length > best.length) { best = result; failures = failed; }
      // Retry constrained routes first rather than repeatedly occupying their only
      // escape corridor with an easier route. Stable sorting preserves determinism.
      const failedIds = new Set(failed.map((item) => item.edgeId));
      order = [...order].sort((a, b) => Number(failedIds.has(b.id)) - Number(failedIds.has(a.id)));
    }
    return { routes: best.sort((a, b) => cmp(a.id, b.id)), conflicts: failures, warnings };
  }
  return Object.freeze({ plan });
});
