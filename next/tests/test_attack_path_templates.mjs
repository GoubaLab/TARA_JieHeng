import test from "node:test";
import assert from "node:assert/strict";
import {
  templateToPath,
  appendTemplatePaths,
  existingTemplateCodes,
} from "../client/src/attack-path-templates.js";

function template(id = "PATH-UPDATE") {
  return {
    id,
    version: "1",
    source_code: `TPL:${id}@1`,
    name: "Synthetic update path",
    entry_point: "Synthetic update interface",
    actor: "Synthetic untrusted updater",
    objective: "Unauthorized update changes behavior",
    prerequisites: [
      "Update interface is reachable",
      "Authorization boundary needs review",
    ],
    steps: [
      "Reach the update interface",
      "Submit an altered update candidate",
      "Check whether installation accepts the candidate",
    ],
    controls_to_check: [
      "Content origin verification",
      "Target and version authorization",
    ],
    properties: ["integrity"],
    tags: ["synthetic"],
  };
}

test("preset creates an editable path with explicit assumptions but no invented assessment", () => {
  const t = template(),
    before = structuredClone(t),
    p = templateToPath(t, () => "path-new");
  assert.equal(p.id, "path-new");
  assert.equal(p.source_code, "TPL:PATH-UPDATE@1");
  assert.equal(p.feasibility, null);
  assert.equal(p.rationale, "");
  assert.equal(p.reference, null);
  assert.ok(!("potential" in p));
  assert.equal(p.actor, t.actor);
  assert.equal(p.vector, t.entry_point);
  assert.ok(p.steps.includes("1. Reach the update interface"));
  assert.ok(p.steps.includes("适用前提"));
  assert.ok(p.steps.includes("待核对的控制"));
  p.name = "Edited by analyst";
  p.steps += "\nProject evidence";
  assert.deepEqual(t, before);
});

test("batch append preserves manual assessment and ordering while deduplicating template provenance", () => {
  const original = [
    {
      id: "manual",
      name: "Manual path",
      steps: "Keep",
      feasibility: 3,
      rationale: "Measured",
      reference: null,
    },
    {
      ...templateToPath(template(), () => "old"),
      steps: "User refined route",
      feasibility: 2,
      rationale: "User evidence",
    },
  ];
  const before = structuredClone(original);
  let calls = 0;
  const result = appendTemplatePaths(
    original,
    [template(), template("PATH-DEBUG"), template("PATH-DEBUG")],
    () => `new-${++calls}`,
  );
  assert.equal(result.added, 1);
  assert.equal(result.skipped, 2);
  assert.equal(calls, 1);
  assert.deepEqual(original, before);
  assert.deepEqual(result.paths.slice(0, 2), before);
  assert.equal(result.paths[2].source_code, "TPL:PATH-DEBUG@1");
  assert.equal(result.paths[2].feasibility, null);
  assert.deepEqual(
    [...existingTemplateCodes(result.paths)],
    ["TPL:PATH-UPDATE@1", "TPL:PATH-DEBUG@1"],
  );
});

test("capacity applies after deduplication and rejects the whole oversized selection", () => {
  const paths = Array.from({ length: 99 }, (_, i) => ({
    id: `p${i}`,
    name: `Manual ${i}`,
  }));
  const before = structuredClone(paths);
  let calls = 0;
  assert.throws(
    () =>
      appendTemplatePaths(
        paths,
        [template("A"), template("B")],
        () => `n${++calls}`,
      ),
    /最多 100/,
  );
  assert.equal(calls, 0);
  assert.deepEqual(paths, before);
  const result = appendTemplatePaths(
    paths,
    [template("A"), template("A")],
    () => "last",
  );
  assert.equal(result.paths.length, 100);
  assert.equal(result.added, 1);
  assert.equal(result.skipped, 1);
});

test("malformed preset or repeated path IDs cannot partly modify existing paths", () => {
  const original = [{ id: "existing", name: "Preserve" }],
    before = structuredClone(original);
  const broken = template("broken");
  broken.prerequisites = [];
  assert.throws(() =>
    appendTemplatePaths(original, [template(), broken], () => "new"),
  );
  assert.deepEqual(original, before);
  assert.throws(
    () => appendTemplatePaths(original, [template()], () => "existing"),
    /编号重复/,
  );
  assert.throws(
    () =>
      templateToPath(
        { ...template(), steps: ["x".repeat(10001)] },
        () => "new",
      ),
    /长度上限/,
  );
  assert.throws(
    () =>
      templateToPath(
        { ...template(), source_code: "private:retired" },
        () => "new",
      ),
    /来源标识/,
  );
});
