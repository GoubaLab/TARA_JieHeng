import { test } from "node:test";
import assert from "node:assert/strict";
import {
  effectiveMethod,
  previewRisk,
  methodErrors,
} from "../client/src/method-config.js";
const matrix = [
  [1, 1, 1, 1],
  [1, 2, 2, 3],
  [2, 3, 3, 4],
  [3, 4, 5, 5],
];
test("legacy method defaults are displayed without mutating persisted method", () => {
  const old = { name: "Synthetic", rationale: "Original example", matrix };
  const before = structuredClone(old),
    view = effectiveMethod(old);
  assert.deepEqual(old, before);
  assert.equal(view.impact_aggregation, "maximum");
  assert.equal(view.risk_calculation, "matrix");
  assert.deepEqual(view.impact_weights, {
    safety: 25,
    financial: 25,
    operational: 25,
    privacy: 25,
  });
});
test("matrix and product preview use distinct scales and closed upper bounds", () => {
  const m = effectiveMethod({ matrix });
  assert.equal(previewRisk(m, 3, 4), 5);
  assert.equal(previewRisk(m, 0, 4), 1);
  m.risk_calculation = "product";
  const expected = [
    [1, 1, 1, 2],
    [1, 2, 2, 3],
    [1, 2, 3, 4],
    [2, 3, 4, 5],
  ];
  assert.deepEqual(
    [0, 1, 2, 3].map((i) => [1, 2, 3, 4].map((f) => previewRisk(m, i, f))),
    expected,
  );
  for (const [i, f] of [
    [null, 4],
    [0, null],
    [-1, 2],
    [3, 0],
    [4, 1],
  ])
    assert.equal(previewRisk(m, i, f), null);
});
test("settings reject inverted matrix, zero total weights and equal thresholds", () => {
  const m = effectiveMethod({
    matrix: structuredClone(matrix),
    impact_aggregation: "weighted_mean",
  });
  assert.deepEqual(methodErrors(m), []);
  m.impact_weights = { safety: 0, financial: 0, operational: 0, privacy: 0 };
  assert.equal(methodErrors(m).length, 1);
  m.product_thresholds = [3, 3, 9, 12];
  m.matrix[2][1] = 1;
  assert.equal(methodErrors(m).length, 3);
});

test("coefficient sums remain raw decimals and use the raw product without an added unit", async () => {
  const { coefficientImpact, coefficientMaximum, previewScore } = await import(
    "../client/src/method-config.js"
  );
  const method = effectiveMethod({
    matrix,
    impact_aggregation: "coefficient_sum",
    risk_calculation: "product",
  });
  const impact = coefficientImpact(
    { safety: 3, financial: 2, operational: 1, privacy: 1 },
    method,
  );
  assert.equal(impact, 5.2);
  assert.equal(coefficientMaximum(method), 6.6);
  assert.equal(previewScore(method, impact, 2), 10.4);
  assert.equal(previewRisk(method, impact, 2), 4);
  assert.equal(previewScore(method, 2.5, 2), 5);
  assert.equal(previewRisk(method, 2.5, 2), 2);
  method.product_thresholds = [0.3, 0.6, 0.9, 1.2];
  assert.equal(
    previewRisk(method, 0.3, 3),
    3,
    "0.3 × 3 is on the inclusive 0.9 threshold",
  );
  assert.equal(previewRisk(method, 0, 4), 1);
  method.impact_coefficients = {
    safety: 1,
    financial: 0,
    operational: 0,
    privacy: 0,
  };
  assert.equal(
    coefficientImpact(
      { safety: 2, financial: null, operational: null, privacy: null },
      method,
    ),
    2,
  );
  method.risk_calculation = "matrix";
  assert.ok(methodErrors(method).length > 0);
});
