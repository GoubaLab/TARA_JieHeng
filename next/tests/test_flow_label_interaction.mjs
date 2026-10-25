import test from 'node:test';
import assert from 'node:assert/strict';
import { Diagram } from '../client/src/diagram.js';

class Surface {
  constructor() {
    this.listeners = new Map();
    this.captures = new Set();
    this.clientWidth = 850;
    this.clientHeight = 540;
    this.classList = { toggle() {} };
  }
  addEventListener(name, handler) { this.listeners.set(name, handler); }
  removeEventListener(name, handler) {
    if (this.listeners.get(name) === handler) this.listeners.delete(name);
  }
  emit(name, event) { return this.listeners.get(name)?.(event); }
  setPointerCapture(id) { this.captures.add(id); }
  hasPointerCapture(id) { return this.captures.has(id); }
  releasePointerCapture(id) { this.captures.delete(id); }
  focus() {}
}

const graph = (offset) => ({
  nodes: [
    { id: 'controller', name: 'Controller', kind: 'process', x: 0, y: 0, width: 180, height: 80 },
    { id: 'gateway', name: 'Gateway', kind: 'process', x: 400, y: 0, width: 180, height: 80 },
  ],
  flows: [{
    id: 'command', name: 'Control command', source: 'controller', target: 'gateway',
    sourceSide: 'right', targetSide: 'left', protocol: 'CAN', description: 'Control request',
    ...(offset === undefined ? {} : { labelOffset: structuredClone(offset) }),
  }],
  equal_spacing: true,
});

function harness(t, initial = graph({ x: 0, y: 0 })) {
  const root = new Surface(), svg = new Surface(), inspector = {}, hint = {}, hintBar = new Surface();
  const surfaces = { svg, '.canvas-inspector': inspector, '.hint-text': hint, '.canvas-hint': hintBar };
  root.querySelector = (selector) => surfaces[selector];
  const windowSurface = new Surface();
  const storage = new Map();
  let plannerCalls = 0;
  windowSurface.AssuranceLaneRouter = { plan(input) {
    plannerCalls += 1;
    return { conflicts: [], routes: input.edges.map(edge => ({
      id: edge.id, points: [{ x: 180, y: 40 }, { x: 400, y: 40 }],
    })) };
  }};
  const restore = [];
  for (const [name, value] of Object.entries({
    window: windowSurface,
    sessionStorage: {
      getItem: key => storage.get(key) ?? null,
      setItem: (key, value) => storage.set(key, value),
      removeItem: key => storage.delete(key),
    },
  })) {
    const descriptor = Object.getOwnPropertyDescriptor(globalThis, name);
    Object.defineProperty(globalThis, name, { configurable: true, writable: true, value });
    restore.push(() => descriptor ? Object.defineProperty(globalThis, name, descriptor) : delete globalThis[name]);
  }
  class FixtureDiagram extends Diagram {
    render() { this.renderCount = (this.renderCount || 0) + 1; }
    inspect() { this.inspectCount = (this.inspectCount || 0) + 1; }
    point(event) { return { x: event.clientX, y: event.clientY }; }
  }
  const diagram = new FixtureDiagram(root, initial, {
    escape: value => String(value), draftKey: 'diagram-label-test',
    onSave: async () => {}, onFlow: () => assert.fail('label interactions must not open a new flow'),
  });
  t.after(() => { diagram.destroy(); restore.reverse().forEach(fn => fn()); });
  const target = { closest: selector => ({
    '[data-flow]': { dataset: { flow: 'command' } },
    '[data-flow-label]': { dataset: { flowLabel: 'command' } },
  })[selector] ?? null };
  const pointer = (x, y, pointerId = 1) => ({ button: 0, clientX: x, clientY: y, pointerId, target, preventDefault() {} });
  const down = (x = 290, y = 40, id = 1) => svg.emit('pointerdown', pointer(x, y, id));
  const move = (x, y, id = 1) => windowSurface.emit('pointermove', pointer(x, y, id));
  const up = (x, y, id = 1) => windowSurface.emit('pointerup', pointer(x, y, id));
  const click = action => root.emit('click', { target: { closest: selector =>
    selector === '[data-dfd]' ? { dataset: { dfd: action } } : null,
  }});
  return { diagram, svg, windowSurface, storage, down, move, up, click, plannerCalls: () => plannerCalls };
}

const semanticFlow = ({ labelOffset, ...flow }) => flow;

test('label pointer chain changes only offset, commits once, and never reroutes the drag', async t => {
  const h = harness(t), before = structuredClone(h.diagram.graph), initialCalls = h.plannerCalls();
  h.down();
  assert.equal(h.diagram.selected, 'command');
  assert.equal(h.diagram.drag.type, 'label');
  assert.equal(h.svg.hasPointerCapture(1), true);
  h.move(310, 60);
  h.move(330, 75);
  h.move(345, 84);
  assert.equal(h.diagram.undo.length, 0, 'moving must not create intermediate history entries');
  h.up(345, 84);
  assert.deepEqual(h.diagram.graph.flows[0].labelOffset, { x: 55, y: 44 });
  assert.deepEqual(semanticFlow(h.diagram.graph.flows[0]), semanticFlow(before.flows[0]));
  assert.deepEqual(h.diagram.graph.nodes, before.nodes);
  assert.equal(h.plannerCalls(), initialCalls);
  assert.equal(h.diagram.undo.length, 1);
  assert.deepEqual(h.diagram.undo[0], before);
  assert.equal(h.diagram.pointerId, null);
  assert.equal(h.diagram.drag, null);
  assert.equal(h.svg.hasPointerCapture(1), false);
  assert.deepEqual(JSON.parse(h.storage.get('diagram-label-test')).graph, h.diagram.graph);
  await h.click('undo');
  assert.deepEqual(h.diagram.graph, before);
  assert.equal(h.diagram.redo.length, 1);
  assert.equal(h.storage.has('diagram-label-test'), false);
  await h.click('redo');
  assert.deepEqual(h.diagram.graph.flows[0].labelOffset, { x: 55, y: 44 });
});

test('pointercancel restores exact old graph including absent offsets without adding history', t => {
  const h = harness(t, graph()), before = structuredClone(h.diagram.graph), initialCalls = h.plannerCalls();
  h.down(); h.move(350, 90);
  assert.deepEqual(h.diagram.graph.flows[0].labelOffset, { x: 60, y: 50 });
  h.windowSurface.emit('pointercancel', { pointerId: 1 });
  assert.deepEqual(h.diagram.graph, before);
  assert.equal(h.diagram.undo.length, 0);
  assert.equal(h.diagram.dirty, false);
  assert.equal(h.svg.captures.size, 0);
  assert.equal(h.plannerCalls(), initialCalls);
  assert.equal(h.storage.size, 0);
});

test('Escape cancels a label drag through the registered keyboard handler', t => {
  const h = harness(t, graph({ x: 20, y: -30 })), before = structuredClone(h.diagram.graph);
  h.down(); h.move(420, 120);
  let prevented = false;
  h.svg.emit('keydown', { key: 'Escape', preventDefault() { prevented = true; } });
  assert.equal(prevented, true);
  assert.deepEqual(h.diagram.graph, before);
  assert.equal(h.diagram.undo.length, 0);
  assert.equal(h.diagram.drag, null);
  assert.equal(h.svg.captures.size, 0);
  h.up(420, 120);
  assert.equal(h.diagram.undo.length, 0, 'late pointerup after Escape must not commit');
});

test('clicking without dragging selects the label without changing legacy data or history', t => {
  const h = harness(t, graph()), before = structuredClone(h.diagram.graph), initialCalls = h.plannerCalls();
  h.down(); h.up(290, 40);
  assert.equal(h.diagram.selected, 'command');
  assert.deepEqual(h.diagram.graph, before);
  assert.equal(h.diagram.undo.length, 0);
  assert.equal(h.diagram.dirty, false);
  assert.equal(h.plannerCalls(), initialCalls);
});

test('other pointers cannot move, cancel, or commit an active label drag', t => {
  const h = harness(t), before = structuredClone(h.diagram.graph);
  h.down();
  h.move(600, 500, 2);
  h.windowSurface.emit('pointercancel', { pointerId: 2 });
  h.up(600, 500, 2);
  assert.deepEqual(h.diagram.graph, before);
  assert.equal(h.diagram.pointerId, 1);
  assert.equal(h.svg.hasPointerCapture(1), true);
  h.move(310, 45); h.up(310, 45);
  assert.deepEqual(h.diagram.graph.flows[0].labelOffset, { x: 20, y: 5 });
  assert.equal(h.diagram.undo.length, 1);
});

test('restore center resets the position without rerouting and remains undoable', async t => {
  const h = harness(t, graph({ x: 35, y: -55 }));
  h.down(); h.up(290, 40);
  const before = structuredClone(h.diagram.graph), initialCalls = h.plannerCalls();
  await h.click('center-label');
  assert.deepEqual(h.diagram.graph.flows[0].labelOffset, { x: 0, y: 0 });
  assert.equal(h.plannerCalls(), initialCalls);
  assert.equal(h.diagram.undo.length, 1);
  await h.click('undo');
  assert.deepEqual(h.diagram.graph, before);
  assert.equal(h.diagram.dirty, false);
  await h.click('redo');
  assert.deepEqual(h.diagram.graph.flows[0].labelOffset, { x: 0, y: 0 });
});

test('destroying a diagram during label movement rolls back the draft and releases listeners', t => {
  const h = harness(t), before = structuredClone(h.diagram.graph);
  h.down(); h.move(340, 70);
  h.diagram.destroy();
  assert.deepEqual(h.diagram.graph, before);
  assert.equal(h.diagram.undo.length, 0);
  assert.equal(h.svg.captures.size, 0);
  assert.equal(h.svg.listeners.size, 0);
  assert.equal(h.windowSurface.listeners.size, 0);
});


test('no-op pointer movement and dragging back preserve legacy absence without creating history', t => {
  const h=harness(t,graph()),before=structuredClone(h.diagram.graph);
  h.down();h.move(290,40);h.move(350,90);h.move(290,40);h.up(290,40);
  assert.deepEqual(h.diagram.graph,before);
  assert.equal(h.diagram.undo.length,0);
  assert.equal(h.diagram.dirty,false);
});

test('returning to the grab point preserves an existing sub-tenth offset exactly', t => {
  const h=harness(t,graph({x:0.04,y:0.03})),before=structuredClone(h.diagram.graph);
  h.down();h.move(330,90);h.move(290,40);h.up(290,40);
  assert.deepEqual(h.diagram.graph,before);
  assert.equal(h.diagram.undo.length,0);
});
