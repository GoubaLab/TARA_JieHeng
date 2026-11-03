import test from "node:test";
import assert from "node:assert/strict";
import {
  normalizeExclusions,
  propertyHasScenario,
  activeAssetProperties,
  deletionBlockReason,
  deletionReferences,
} from "../client/src/asset-scope.js";

test("asset property edits retain reasons and remove exclusions of removed properties", () => {
  const original = [
    { property: "integrity", reason: "边界说明" },
    { property: "availability", reason: "部署范围说明" },
  ];
  assert.deepEqual(normalizeExclusions(["integrity"], original), [
    { property: "integrity", reason: "边界说明" },
  ]);
  assert.equal(original.length, 2);
  assert.deepEqual(
    normalizeExclusions(["privacy"], [{ property: "privacy" }]),
    [{ property: "privacy", reason: "" }],
  );
});
test("all linked assets and properties prevent exclusion, including legacy scenario relations", () => {
  const ws = {
    scenarios: [
      {
        asset_id: "a",
        property: "integrity",
        asset_ids: ["a", "b"],
        properties: ["integrity", "availability"],
      },
      { asset_id: "c", property: "privacy" },
    ],
  };
  assert.equal(propertyHasScenario(ws, "b", "availability"), true);
  assert.equal(propertyHasScenario(ws, "c", "privacy"), true);
  assert.equal(propertyHasScenario(ws, "b", "privacy"), false);
  assert.equal(propertyHasScenario(ws, undefined, "integrity"), false);
});
test("threat entry offers only properties in the asset analysis scope", () => {
  assert.deepEqual(
    activeAssetProperties({
      properties: ["integrity", "availability"],
      excluded_properties: [{ property: "availability", reason: "边界" }],
    }),
    ["integrity"],
  );
  assert.deepEqual(activeAssetProperties(undefined), []);
});
test("requirements with any verification history cannot be deleted through UI", () => {
  const r = { id: "r" };
  assert.match(
    deletionBlockReason({ access_role: "viewer" }, "asset", r),
    /只读/,
  );
  assert.match(
    deletionBlockReason(
      { evidence: [{ requirement_id: "r", result: "fail" }] },
      "requirement",
      r,
    ),
    /验证记录/,
  );
  assert.equal(
    deletionBlockReason(
      { evidence: [{ requirement_id: "other" }] },
      "requirement",
      r,
    ),
    "",
  );
  assert.equal(deletionBlockReason({ evidence: [] }, "asset", r), "");
});

test("goal deletion exposes unconfirmed draft and requirement dependencies for resolution", () => {
  const requirement = { id: "r", goal_ids: ["g"] },
    draft = { scenario_id: "s", goal_ids: ["g"] };
  assert.deepEqual(
    deletionReferences(
      { requirements: [requirement], treatment_drafts: [draft] },
      "goal",
      "g",
    ),
    [
      { kind: "requirement", record: requirement },
      { kind: "draft", record: draft },
    ],
  );
  assert.deepEqual(
    deletionReferences(
      { requirements: [requirement], treatment_drafts: [draft] },
      "goal",
      "other",
    ),
    [],
  );
});
