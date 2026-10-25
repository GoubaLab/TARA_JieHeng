export const domains = {
  general: "通用产品",
  automotive: "汽车与车联网",
  robotics: "机器人与具身智能",
  iot: "物联网设备",
  charging: "充电设施",
  internet: "互联网与云服务",
};
export const dimensionKeys = ["safety", "financial", "operational", "privacy"];
export const dimensionCodes = {
  safety: "S",
  financial: "F",
  operational: "O",
  privacy: "P",
};
const defaults = {
  impact_aggregation: "maximum",
  impact_weights: { safety: 25, financial: 25, operational: 25, privacy: 25 },
  impact_coefficients: {
    safety: 1,
    financial: 1,
    operational: 0.1,
    privacy: 0.1,
  },
  impact_dimensions: {
    safety: "安全",
    financial: "财务",
    operational: "运行",
    privacy: "隐私",
  },
  impact_levels: ["可忽略", "中等", "重大", "严重"].map((label) => ({
    label,
    description: "",
  })),
  feasibility_levels: ["很低", "低", "中", "高"].map((label) => ({
    label,
    description: "",
  })),
  risk_levels: ["很低", "低", "中", "高", "很高"].map((label) => ({
    label,
    description: "",
  })),
  risk_calculation: "matrix",
  product_thresholds: [3, 6, 9, 12],
};
export function effectiveMethod(method = {}) {
  const value = { ...structuredClone(defaults), ...method };
  value.impact_dimensions = {
    ...defaults.impact_dimensions,
    ...method.impact_dimensions,
  };
  value.impact_weights = {
    ...defaults.impact_weights,
    ...method.impact_weights,
  };
  value.impact_coefficients = {
    ...defaults.impact_coefficients,
    ...method.impact_coefficients,
  };
  return value;
}
// Compare decimal inputs without binary floating point changing a threshold boundary.
function decimal(value) {
  const [mantissa, exponent = "0"] = String(value).toLowerCase().split("e");
  const fraction = (mantissa.split(".")[1] || "").length;
  let n = BigInt(mantissa.replace(".", "")),
    scale = fraction - Number(exponent);
  if (scale < 0) {
    n *= 10n ** BigInt(-scale);
    scale = 0;
  }
  return { n, scale };
}
function addDecimal(a, b) {
  const scale = Math.max(a.scale, b.scale);
  return {
    n:
      a.n * 10n ** BigInt(scale - a.scale) +
      b.n * 10n ** BigInt(scale - b.scale),
    scale,
  };
}
function multiplyDecimal(a, b) {
  return { n: a.n * b.n, scale: a.scale + b.scale };
}
function asNumber(a) {
  return Number(`${a.n}e-${a.scale}`);
}
function greaterDecimal(a, b) {
  const s = Math.max(a.scale, b.scale);
  return a.n * 10n ** BigInt(s - a.scale) > b.n * 10n ** BigInt(s - b.scale);
}
export function coefficientImpact(values, method) {
  let sum = { n: 0n, scale: 0 };
  for (const key of dimensionKeys) {
    const c = method.impact_coefficients[key];
    if (!Number.isFinite(c)) return null;
    if (c === 0) continue;
    if (!Number.isInteger(values[key]) || values[key] < 0 || values[key] > 3)
      return null;
    sum = addDecimal(sum, multiplyDecimal(decimal(values[key]), decimal(c)));
  }
  return asNumber(sum);
}
export function coefficientMaximum(method) {
  return coefficientImpact(
    Object.fromEntries(dimensionKeys.map((k) => [k, 3])),
    method,
  );
}
export function previewScore(method, impact, feasibility) {
  if (
    !Number.isFinite(impact) ||
    !Number.isInteger(feasibility) ||
    feasibility < 1 ||
    feasibility > 4
  )
    return null;
  return asNumber(
    multiplyDecimal(
      decimal(
        method.impact_aggregation === "coefficient_sum" ? impact : impact + 1,
      ),
      decimal(feasibility),
    ),
  );
}
export function previewRisk(method, impact, feasibility) {
  const coefficient = method.impact_aggregation === "coefficient_sum";
  if (
    !Number.isFinite(impact) ||
    impact < 0 ||
    impact > (coefficient ? coefficientMaximum(method) : 3) ||
    (!coefficient && !Number.isInteger(impact)) ||
    !Number.isInteger(feasibility) ||
    feasibility < 1 ||
    feasibility > 4
  )
    return null;
  if (method.risk_calculation === "product") {
    const score = multiplyDecimal(
      decimal(coefficient ? impact : impact + 1),
      decimal(feasibility),
    );
    return (
      1 +
      method.product_thresholds.filter(
        (t) => Number.isFinite(t) && greaterDecimal(score, decimal(t)),
      ).length
    );
  }
  return coefficient
    ? null
    : (method.matrix?.[impact]?.[feasibility - 1] ?? null);
}
export function methodErrors(method) {
  const errors = [];
  const coefficient = method.impact_aggregation === "coefficient_sum";
  if (
    coefficient &&
    (!dimensionKeys.some((k) => method.impact_coefficients[k] > 0) ||
      dimensionKeys.some(
        (k) =>
          !Number.isFinite(method.impact_coefficients[k]) ||
          method.impact_coefficients[k] < 0 ||
          method.impact_coefficients[k] > 100,
      ))
  )
    errors.push("系数须为 0–100，且至少一项大于 0。");
  if (coefficient && method.risk_calculation !== "product")
    errors.push("系数求和直接参与乘积计算，请选择阈值分级。");
  if (
    method.impact_aggregation === "weighted_mean" &&
    !dimensionKeys.some((k) => method.impact_weights[k] > 0)
  )
    errors.push("至少一个影响维度的权重必须大于 0。");
  if (
    method.product_thresholds.some(
      (v, i, all) =>
        !Number.isFinite(v) ||
        (!coefficient && !Number.isInteger(v)) ||
        v < (coefficient ? 0 : 1) ||
        v > (coefficient ? 4800 : 15) ||
        (i > 0 && v <= all[i - 1]),
    )
  )
    errors.push(
      coefficient
        ? "系数法的 4 个风险阈值须为 0–4800 的递增数值。"
        : "乘积法的 4 个阈值必须为 1–15 的递增整数。",
    );
  const m = method.matrix;
  if (
    !Array.isArray(m) ||
    m.length !== 4 ||
    m.some((row) => !Array.isArray(row) || row.length !== 4)
  )
    errors.push("风险矩阵必须为 4 × 4。");
  else if (
    m.some((row, i) =>
      row.some(
        (v, j) =>
          !Number.isInteger(v) ||
          v < 1 ||
          v > 5 ||
          (i > 0 && v < m[i - 1][j]) ||
          (j > 0 && v < row[j - 1]),
      ),
    )
  )
    errors.push("风险矩阵值应为 1–5；影响或可行性增加时，风险不得降低。");
  return errors;
}
export function levelOptions(levels, start = 0) {
  return levels.map(({ label, description }, i) => ({
    value: i + start,
    label: `${i + start} · ${label}`,
    title: description || label,
  }));
}
