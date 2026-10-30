import test from "node:test";
import assert from "node:assert/strict";
import {
  functionCatalog,
  potentialResult,
  projectImpactCriteria,
  impactGuidance,
  pickScenario,
} from "../client/src/tara-guidance.js";
test("function ranges retain their shared meaning; explicit definitions override one code", () => {
  const ws = {
    item: {
      purpose: "FR-019..FR-024 休眠唤醒：硬线、网络、RTC；FR-025 单项：SOC估算",
      functions: [
        { id: "FR-021", name: "已确认功能", description: "实际需求内容" },
      ],
    },
  };
  const rows = functionCatalog(ws);
  assert.equal(rows.length, 7);
  assert.equal(rows.find((r) => r.id === "FR-021").description, "实际需求内容");
  assert.equal(rows.find((r) => r.id === "FR-020").group, "FR-019–FR-024");
  assert.equal(rows.find((r) => r.id === "FR-020").name, "休眠唤醒");
  assert.equal(rows.find((r) => r.id === "FR-025").group, "");
});
test("attack potential uses categorical values, explicit zero and project bands", () => {
  const p = {
    elapsed_time: 4,
    expertise: 3,
    knowledge: 3,
    opportunity: 0,
    equipment: 4,
  };
  assert.equal(potentialResult(p).level, 3);
  assert.equal(
    potentialResult(p, { attack_potential_thresholds: [10, 14, 20] }).level,
    2,
  );
  assert.equal(potentialResult({ ...p, elapsed_time: 999999 }), null);
  assert.equal(potentialResult({ ...p, expertise: null }), null);
  assert.equal(potentialResult({ ...p, elapsed_time: 2 }), null);
  assert.equal(
    potentialResult({
      elapsed_time: 17,
      expertise: 8,
      knowledge: 0,
      opportunity: 0,
      equipment: 0,
    }).level,
    1,
  );
});
test("each SFOP dimension has four distinct criteria and workbook rules retain provenance", () => {
  for (const k of ["safety", "financial", "operational", "privacy"]) {
    assert.equal(impactGuidance[k].rows.length, 4);
    assert.equal(new Set(impactGuidance[k].rows).size, 4);
  }
  const ws = {
    tara_import: {
      sheets: [
        {
          name: "附录A 影响评级对照",
          rows: [
            ["A.2 财务影响"],
            ["取值"],
            ...["0", "10", "100", "1000"].map((v, i) => [
              v,
              "级" + i,
              "项目条件" + i,
              "说明",
            ]),
          ],
        },
      ],
    },
  };
  assert.equal(
    projectImpactCriteria(ws, "financial").rows[2].description,
    "项目条件2；说明",
  );
  assert.equal(projectImpactCriteria(ws, "safety"), null);
});
test("stage changes copy scenario data and preserve untouched assessments", () => {
  const s = {
    id: "s",
    title: "t",
    paths: [{ id: "p", potential: { elapsed_time: 4 } }],
    impact: { safety: 2 },
    unknown: "internal",
  };
  const v = pickScenario(s);
  v.paths[0].potential.elapsed_time = 0;
  assert.equal(s.paths[0].potential.elapsed_time, 4);
  assert.equal(v.unknown, undefined);
  assert.equal(v.impact.safety, 2);
});
