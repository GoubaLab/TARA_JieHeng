export function normalizeExclusions(properties = [], exclusions = []) {
  const selected = new Set(properties),
    seen = new Set();
  return exclusions
    .filter(
      (row) =>
        selected.has(row.property) &&
        !seen.has(row.property) &&
        seen.add(row.property),
    )
    .map(({ property, reason }) => ({ property, reason: reason || "" }));
}
export function propertyHasScenario(ws, assetId, property) {
  return (
    !!assetId &&
    (ws.scenarios || []).some(
      (s) =>
        (s.asset_ids?.length ? s.asset_ids : [s.asset_id]).includes(assetId) &&
        (s.properties?.length ? s.properties : [s.property]).includes(property),
    )
  );
}
export function activeAssetProperties(asset) {
  const excluded = new Set(
    (asset?.excluded_properties || []).map((e) => e.property),
  );
  return (asset?.properties || []).filter((p) => !excluded.has(p));
}
export function deletionBlockReason(ws, kind, record) {
  if (ws.access_role === "viewer") return "当前项目为只读权限";
  if (
    kind === "requirement" &&
    (ws.evidence || []).some((e) => e.requirement_id === record.id)
  )
    return "该需求已有验证记录，不能删除；请通过修订需求保留验证追溯。";
  return "";
}
export function deletionReferences(ws, kind, id) {
  const refs = [];
  const add = (kind, record) => refs.push({ kind, record });
  for (const s of ws.scenarios || []) {
    if (
      kind === "asset" &&
      (s.asset_ids?.length ? s.asset_ids : [s.asset_id]).includes(id)
    )
      add("scenario", s);
    if (kind === "damage" && (s.damage_ids || []).includes(id))
      add("scenario", s);
  }
  if (kind === "asset")
    for (const d of ws.damage_scenarios || [])
      if ((d.asset_ids || []).includes(id)) add("damage", d);
  for (const r of ws.requirements || []) {
    if (
      (kind === "scenario" &&
        (r.scenario_ids?.length ? r.scenario_ids : [r.scenario_id]).includes(
          id,
        )) ||
      (kind === "goal" && (r.goal_ids || []).includes(id))
    )
      add("requirement", r);
  }
  if (kind === "scenario")
    for (const g of ws.security_goals || [])
      if ((g.scenario_ids || []).includes(id)) add("goal", g);
  if (kind === "goal") {
    for (const d of ws.decisions || [])
      if ((d.goal_ids || []).includes(id)) add("decision", d);
    for (const d of ws.treatment_drafts || [])
      if ((d.goal_ids || []).includes(id)) add("draft", d);
  }
  return refs;
}
