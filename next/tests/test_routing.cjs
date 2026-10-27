'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const { test } = require('node:test');
const context = { module: { exports: {} } };
vm.runInNewContext(fs.readFileSync(path.join(__dirname, '../web/lane-router.js'), 'utf8'), context);
const { plan } = context.module.exports;
const TOL = 1e-5;
const normalized = (value) => JSON.parse(JSON.stringify(value));

// These assertions inspect returned geometry independently of the planner's search code.
function checkGeometry(input, output, allowPartial = false) {
  if (!allowPartial) {
    assert.equal(output.conflicts.length, 0, JSON.stringify(output.conflicts));
    assert.equal(output.routes.length, input.edges.length, 'Every business edge has exactly one route');
  }
  const nodes = new Map(input.nodes.map((node) => [node.id, node]));
  const edges = new Map(input.edges.map((edge) => [edge.id, edge]));
  const lines = [], groups = new Map();
  for (const route of output.routes) {
    const edge = edges.get(route.id);
    assert.ok(edge, 'No invented edge');
    for (const terminal of ['source', 'target']) {
      const end = route[terminal], node = nodes.get(edge[terminal]);
      const side = end.side, p = end.point;
      if (edge[terminal + 'Side']) assert.equal(side, edge[terminal + 'Side']);
      assert.ok(end.ratio > 0 && end.ratio < 1);
      const expectedX = side === 'left' ? node.x : side === 'right' ? node.x + node.width : node.x + node.width * end.ratio;
      const expectedY = side === 'top' ? node.y : side === 'bottom' ? node.y + node.height : node.y + node.height * end.ratio;
      assert.ok(Math.abs(p.x - expectedX) < TOL && Math.abs(p.y - expectedY) < TOL, 'Port lies at its exact proportional position');
      const key = node.id + ':' + side;
      if (!groups.has(key)) groups.set(key, []);
      groups.get(key).push(end.ratio);
    }
    assert.deepEqual(normalized(route.points[0]), normalized(route.source.point));
    assert.deepEqual(normalized(route.points.at(-1)), normalized(route.target.point));
    for (let i = 1; i < route.points.length; i++) {
      const a = route.points[i - 1], b = route.points[i];
      const horizontal = Math.abs(a.y - b.y) < TOL, vertical = Math.abs(a.x - b.x) < TOL;
      assert.notEqual(horizontal, vertical, 'Each segment is nonzero and orthogonal');
      const line = { id: route.id, horizontal, axis: horizontal ? a.y : a.x, low: Math.min(horizontal ? a.x : a.y, horizontal ? b.x : b.y), high: Math.max(horizontal ? a.x : a.y, horizontal ? b.x : b.y) };
      lines.push(line);
      for (const node of input.nodes.filter((n) => !n.boundary)) {
        const collision = horizontal
          ? line.axis > node.y + TOL && line.axis < node.y + node.height - TOL && line.high > node.x + TOL && line.low < node.x + node.width - TOL
          : line.axis > node.x + TOL && line.axis < node.x + node.width - TOL && line.high > node.y + TOL && line.low < node.y + node.height - TOL;
        assert.equal(collision, false, `Route ${route.id} enters node ${node.id}`);
      }
    }
  }
  for (const ratios of allowPartial ? [] : groups.values()) {
    ratios.sort((a, b) => a - b);
    ratios.forEach((ratio, i) => assert.ok(Math.abs(ratio - (i + 1) / (ratios.length + 1)) < TOL, 'Ports use actual-degree equal division'));
  }
  for (let i = 0; i < lines.length; i++) for (let j = i + 1; j < lines.length; j++) {
    const a = lines[i], b = lines[j];
    if (a.id === b.id || a.horizontal !== b.horizontal) continue;
    const overlap = Math.min(a.high, b.high) - Math.max(a.low, b.low);
    if (overlap > TOL) assert.ok(Math.abs(a.axis - b.axis) >= (input.grid || 20) - TOL,
      `Parallel overlap ${a.id} / ${b.id} has only ${Math.abs(a.axis - b.axis)} clearance`);
  }
  return lines;
}

const pair = () => ({ nodes: [{ id: 'a', x: 0, y: 0, width: 190, height: 80 }, { id: 'b', x: 400, y: 0, width: 190, height: 80 }], edges: [{ id: 'e1', source: 'a', target: 'b' }], grid: 20 });

test('one centered connection and two / three equally distributed connections, including endpoint stubs', () => {
  for (const count of [1, 2, 3]) {
    const input = pair();
    input.edges = Array.from({ length: count }, (_, i) => ({ id: `e${i}`, source: 'a', target: 'b' }));
    const out = plan(input);
    checkGeometry(input, out);
    assert.equal(out.routes.length, count);
  }
});

test('stable paths for reordered input, with no input mutation', () => {
  const input = pair();
  input.edges.push({ id: 'e0', source: 'b', target: 'a' });
  const before = JSON.stringify(input), first = plan(input);
  assert.equal(JSON.stringify(input), before);
  const reversed = { ...input, nodes: [...input.nodes].reverse(), edges: [...input.edges].reverse() };
  assert.deepEqual(normalized(plan(reversed)), normalized(first));
  checkGeometry(input, first);
});

test('routes around intervening components and ignores boundary frames as obstacles', () => {
  const input = pair();
  input.nodes.push({ id: 'blocker', x: 250, y: -40, width: 70, height: 160 }, { id: 'boundary', x: -40, y: -80, width: 700, height: 240, boundary: true });
  const out = plan(input);
  checkGeometry(input, out);
  assert.ok(out.routes[0].points.length >= 4, 'Obstacle requires a detour');
});

test('perpendicular crossings are allowed while parallel lanes retain one grid space', () => {
  const input = { grid: 20, nodes: [
    { id: 'left', x: 0, y: 200, width: 100, height: 80 }, { id: 'right', x: 600, y: 200, width: 100, height: 80 },
    { id: 'top', x: 300, y: 0, width: 100, height: 80 }, { id: 'bottom', x: 300, y: 400, width: 100, height: 80 },
  ], edges: [{ id: 'horizontal', source: 'left', target: 'right' }, { id: 'vertical', source: 'top', target: 'bottom' }] };
  const lines = checkGeometry(input, plan(input));
  assert.ok(lines.some((a) => lines.some((b) => a.id !== b.id && a.horizontal !== b.horizontal && a.axis > b.low && a.axis < b.high && b.axis > a.low && b.axis < a.high)), 'The simple crossing need not take an unnecessary detour');
});

test('capacity failure reports the necessary side length instead of returning overlapping routes', () => {
  const input = pair();
  input.edges = Array.from({ length: 4 }, (_, i) => ({ id: `e${i}`, source: 'a', target: 'b' }));
  const out = plan(input);
  assert.equal(out.routes.length, 0);
  assert.ok(out.conflicts.some((c) => c.code === 'SIDE_CAPACITY' && c.requiredSize === 100));
});

test('blocked ports and invalid endpoints fail explicitly', () => {
  const input = pair();
  input.nodes.push({ id: 'blocker', x: 195, y: 0, width: 100, height: 80 });
  assert.ok(plan(input).conflicts.some((c) => c.code === 'PORT_BLOCKED'));
  const missing = pair(); missing.edges[0].target = 'missing';
  assert.ok(plan(missing).conflicts.some((c) => c.code === 'INVALID_INPUT'));
  const invalidSide = pair(); invalidSide.edges[0].sourceSide = 'diagonal';
  assert.ok(plan(invalidSide).conflicts.some((c) => c.code === 'INVALID_INPUT'));
});

test('unreachable endpoints do not falsely report success when all routes fail', () => {
  const input = pair();
  input.nodes = [{ id: 'a', x: 0, y: 0, width: 100, height: 80 }, input.nodes[1],
    { id: 'top', x: 90, y: -50, width: 80, height: 40 },
    { id: 'bottom', x: 90, y: 90, width: 80, height: 40 },
    { id: 'right', x: 150, y: -50, width: 20, height: 180 }];
  const out = plan(input);
  assert.equal(out.routes.length, 0);
  assert.ok(out.conflicts.length > 0, 'Failure must never look like a successful empty plan');
});

test('huge canvas extents fail promptly without allocating an unbounded grid', () => {
  const input = pair(); input.nodes[1].x = 1e12;
  assert.ok(plan(input).conflicts.some((c) => c.code === 'LAYOUT_TOO_LARGE'));
});

function branchedFixture() {
  const nodes = [
    ['TEST-BOUNDARY', 260, 50, 830, 830, true], ['E1', 30, 420, 190, 110], ['E2', 1170, 420, 190, 110],
    ['P1', 380, 140, 190, 80], ['P2', 380, 340, 190, 80], ['P3', 380, 540, 190, 80], ['P4', 380, 740, 190, 80],
    ['D1', 800, 340, 190, 80], ['D2', 800, 740, 190, 80], ['D3', 800, 540, 190, 80], ['D4', 800, 140, 190, 80],
  ].map(([id, x, y, width, height, boundary]) => ({ id, x, y, width, height, boundary }));
  const rows = [
    [1, 'E1', 'right', 'P1', 'left'], [3, 'E1', 'right', 'P4', 'left'], [5, 'E2', 'left', 'P2', 'top'], [7, 'E2', 'left', 'D1', 'right'],
    [9, 'E2', 'bottom', 'D2', 'right'], [11, 'E2', 'top', 'D4', 'right'], [13, 'P1', 'left', 'P3', 'left'], [15, 'P1', 'bottom', 'P2', 'top'],
    [17, 'P2', 'bottom', 'P3', 'top'], [19, 'P2', 'right', 'P4', 'right'], [21, 'P2', 'right', 'D1', 'left'], [23, 'D1', 'left', 'P3', 'right'],
    [24, 'P4', 'right', 'D2', 'left'], [26, 'P2', 'right', 'D3', 'left'], [27, 'P3', 'right', 'D3', 'left'], [28, 'P4', 'right', 'D3', 'left'],
    [30, 'D4', 'left', 'P2', 'top'], [31, 'D4', 'right', 'P4', 'bottom'],
  ];
  return { nodes, edges: rows.map(([id, source, sourceSide, target, targetSide]) => ({ id: 'VIEW-' + id, source, target, sourceSide, targetSide })), grid: 20 };
}

test('all 18 branched geometry-test edges fit, including non-grid ports on 190 by 80 nodes', () => {
  const input = branchedFixture(), start = performance.now(), out = plan(input);
  const elapsed = performance.now() - start;
  checkGeometry(input, out);
  assert.ok(elapsed < 3000, `Routing took ${elapsed.toFixed(0)} ms`);
  const p2Top = out.routes.flatMap((r) => input.edges.find((e) => e.id === r.id).target === 'P2' && r.target.side === 'top' ? [r.target.ratio] : []);
  assert.deepEqual(normalized(p2Top.sort((a, b) => a - b)), [0.25, 0.5, 0.75]);
});

test('moving, resizing, removing and adding edges retains geometry or returns an explicit conflict', (t) => {
  let seed = 0x51a7e;
  const random = () => ((seed = (Math.imul(seed, 1664525) + 1013904223) >>> 0) / 0x100000000);
  const stats = {}, variants = [];
  for (const kind of ['move', 'resize', 'remove', 'parallel']) for (let i = 0; i < 8; i++) {
    const input = branchedFixture();
    if (kind === 'move') for (const node of input.nodes.filter((n) => !n.boundary)) {
      node.x += Math.round((random() - 0.5) * 140); node.y += Math.round((random() - 0.5) * 100);
    }
    if (kind === 'resize') for (const node of input.nodes.filter((n) => !n.boundary)) {
      node.width += Math.round(random() * 50); node.height += Math.round(random() * 45);
    }
    if (kind === 'remove') input.edges.splice(Math.floor(random() * input.edges.length), 1);
    if (kind === 'parallel') {
      const original = input.edges[Math.floor(random() * input.edges.length)];
      input.edges.push({ ...original, id: original.id + '-parallel' });
      // Make room for four endpoints when the selected side was already full.
      for (const node of input.nodes) if (!node.boundary) node.height = Math.max(110, node.height);
    }
    variants.push({ kind, input });
  }
  for (const { kind, input } of variants) {
    const start = performance.now();
    const out = plan(input);
    const elapsed = performance.now() - start;
    assert.ok(elapsed < 3000, `${kind} routing took ${elapsed.toFixed(0)} ms`);
    if (!stats[kind]) stats[kind] = { solved: 0, conflict: 0, maxMs: 0 };
    stats[kind].maxMs = Math.max(stats[kind].maxMs, Math.round(elapsed));
    if (!out.conflicts.length) { checkGeometry(input, out); stats[kind].solved++; }
    else {
      checkGeometry(input, out, true); stats[kind].conflict++;
      assert.ok(out.routes.length < input.edges.length, 'A rejected plan identifies at least one unresolved edge');
      assert.ok(out.conflicts.every((c) => typeof c.code === 'string' && typeof c.message === 'string'));
    }
  }
  for (const kind of ['move', 'resize', 'remove', 'parallel']) assert.equal(stats[kind].solved, 8, `A previously routable ${kind} variant regressed`);
  t.diagnostic(JSON.stringify(stats));
});
