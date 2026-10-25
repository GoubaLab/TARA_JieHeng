import test from 'node:test';
import assert from 'node:assert/strict';
import {
  routeMidpoint,
  flowLabelPosition,
  labelOffsetForDrag,
} from '../client/src/flow-label.js';

const freezePoints = (points) => Object.freeze(points.map((point) => Object.freeze(point)));

test('default label is centered on a straight route rather than at a route vertex', () => {
  const points = freezePoints([{x: 20, y: 30}, {x: 270, y: 30}]);
  assert.deepEqual(routeMidpoint(points), {x: 145, y: 30});
  assert.deepEqual(flowLabelPosition(points), {x: 145, y: 30});
  assert.deepEqual(flowLabelPosition(points, null), {x: 145, y: 30});
});

test('polyline center follows cumulative length, even when it is inside the shortest segment', () => {
  const points = freezePoints([
    {x: 0, y: 0}, {x: 80, y: 0}, {x: 80, y: 10}, {x: 155, y: 10},
  ]);
  assert.deepEqual(routeMidpoint(points), {x: 80, y: 2.5});
  assert.deepEqual(routeMidpoint([...points].reverse()), {x: 80, y: 2.5});
});

test('vertical, multi-turn, and diagonal routes interpolate along their actual length', () => {
  assert.deepEqual(routeMidpoint([{x: -5, y: 50}, {x: -5, y: -10}]), {x: -5, y: 20});
  assert.deepEqual(routeMidpoint([
    {x: 0, y: 0}, {x: 10, y: 0}, {x: 10, y: 60}, {x: 50, y: 60},
  ]), {x: 10, y: 45});
  assert.deepEqual(routeMidpoint([{x: 0, y: 0}, {x: 3, y: 4}, {x: 9, y: 4}]), {x: 3.5, y: 4});
});

test('empty, singleton, and zero-length segments have stable finite centers and independent outputs', () => {
  assert.deepEqual(routeMidpoint(), {x: 0, y: 0});
  assert.deepEqual(routeMidpoint([]), {x: 0, y: 0});
  const point = Object.freeze({x: -17, y: 42});
  const singleton = routeMidpoint([point]);
  assert.deepEqual(singleton, point);
  assert.notEqual(singleton, point);
  assert.deepEqual(routeMidpoint([point, point, point]), point);
  assert.deepEqual(routeMidpoint([
    {x: 0, y: 0}, {x: 0, y: 0}, {x: 30, y: 0}, {x: 30, y: 0},
  ]), {x: 15, y: 0});
});

test('saved label offsets remain relative to the route after routing changes', () => {
  const offset = Object.freeze({x: 12.4, y: -30.2});
  assert.deepEqual(flowLabelPosition([{x: 0, y: 0}, {x: 100, y: 0}], offset), {x: 62.4, y: -30.2});
  assert.deepEqual(flowLabelPosition([{x: 50, y: 50}, {x: 50, y: 150}], offset), {x: 62.4, y: 69.8});
  assert.deepEqual(flowLabelPosition([{x: 0, y: 0}, {x: 100, y: 0}], {x: 0, y: 0}), {x: 50, y: 0});
});

test('drag uses canvas deltas and keeps the original grab offset without mutating inputs', () => {
  const before = Object.freeze({x: 20, y: -10});
  const start = Object.freeze({x: 75.25, y: 40.5});
  const current = Object.freeze({x: 80.5, y: 51.76});
  // Screen zoom has already been removed by the caller's inverse SVG transform.
  assert.deepEqual(labelOffsetForDrag(before, start, current), {x: 25.3, y: 1.3});
  assert.deepEqual(labelOffsetForDrag(before, start, start), before);
  assert.deepEqual(before, {x: 20, y: -10});
  assert.deepEqual(start, {x: 75.25, y: 40.5});
  assert.deepEqual(current, {x: 80.5, y: 51.76});
  assert.deepEqual(labelOffsetForDrag(undefined, {x: 0, y: 0}, {x: 12.35, y: -8.26}), {x: 12.4, y: -8.3});
});

test('drag offsets respect persistence bounds and normalize negative zero', () => {
  assert.deepEqual(labelOffsetForDrag({x: 9999, y: -9999}, {x: 1, y: 1}, {x: 101, y: -99}), {x: 10000, y: -10000});
  assert.deepEqual(labelOffsetForDrag({x: 0, y: 0}, {x: 0, y: 0}, {x: -0.01, y: 0.01}), {x: 0, y: 0});
});
