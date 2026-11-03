import React from "react";
import { Alert, Input, Select, Space, Tag, Tooltip, Typography } from "antd";
import { properties as propertyNames } from "./api.js";
import { normalizeExclusions, propertyHasScenario } from "./asset-scope.js";
export function AssetScopeField({
  value = [],
  onChange,
  selected = [],
  assetId,
  ws,
}) {
  const rows = normalizeExclusions(selected, value);
  const setScope = (property, excluded) =>
    onChange?.(
      excluded
        ? [
            ...rows.filter((r) => r.property !== property),
            { property, reason: "" },
          ]
        : rows.filter((r) => r.property !== property),
    );
  return (
    <Space orientation="vertical" style={{ width: "100%" }} size={12}>
      <Alert
        type="info"
        showIcon
        title="逐项确认分析范围"
        description="纳入分析的属性应关联威胁场景。不适用需记录具体边界与理由，并随报告保留；已有威胁场景的属性不能直接排除。"
      />
      {selected.map((property) => {
        const exclusion = rows.find((r) => r.property === property);
        const linked = propertyHasScenario(ws, assetId, property);
        return (
          <div
            key={property}
            style={{
              border: "1px solid #e5e7eb",
              padding: 12,
              borderRadius: 6,
            }}
          >
            <Space wrap style={{ marginBottom: exclusion ? 8 : 0 }}>
              <Typography.Text strong>
                {propertyNames[property] || property}
              </Typography.Text>
              <Select
                aria-label={`${propertyNames[property] || property}分析范围`}
                style={{ width: 160 }}
                value={exclusion ? "excluded" : "included"}
                options={[
                  { value: "included", label: "纳入分析" },
                  { value: "excluded", label: "不适用", disabled: linked },
                ]}
                onChange={(v) => setScope(property, v === "excluded")}
              />
              {linked && (
                <Typography.Text type="secondary">已有关联威胁</Typography.Text>
              )}
            </Space>
            {exclusion && (
              <Input.TextArea
                aria-label={`${propertyNames[property] || property}不适用理由`}
                value={exclusion.reason}
                maxLength={2000}
                showCount
                autoSize={{ minRows: 2, maxRows: 5 }}
                placeholder="说明该属性不适用的项目边界、条件与依据（必填）"
                onChange={(e) =>
                  onChange?.(
                    rows.map((r) =>
                      r.property === property
                        ? { ...r, reason: e.target.value }
                        : r,
                    ),
                  )
                }
              />
            )}
          </div>
        );
      })}
      {!selected.length && (
        <Typography.Text type="secondary">
          请先选择资产的安全属性。
        </Typography.Text>
      )}
    </Space>
  );
}
export function AssetScopeSummary({ asset }) {
  const exclusions = new Map(
    (asset.excluded_properties || []).map((r) => [r.property, r.reason]),
  );
  return (
    <Space orientation="vertical" size={4}>
      <Space size={[0, 4]} wrap>
        {(asset.properties || []).map((p) => (
          <Tooltip key={p} title={exclusions.get(p) || "纳入威胁分析"}>
            <Tag color={exclusions.has(p) ? "default" : "blue"}>
              {propertyNames[p] || p}
              {exclusions.has(p) ? " · 不适用" : ""}
            </Tag>
          </Tooltip>
        ))}
      </Space>
      {[...exclusions].map(([p, reason]) => (
        <Typography.Text
          key={p}
          type="secondary"
          style={{ whiteSpace: "pre-wrap", fontSize: 12 }}
        >
          {propertyNames[p] || p}不适用：{reason}
        </Typography.Text>
      ))}
    </Space>
  );
}
