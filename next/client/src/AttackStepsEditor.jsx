import React, { useEffect, useState } from "react";
import {
  Alert,
  Button,
  Card,
  Checkbox,
  Input,
  Select,
  Space,
  Tag,
  Typography,
} from "antd";
import { request } from "./api.js";
import { parseAttackSteps, formatAttackSteps } from "./attack-steps.js";
export function AttackStepsEditor({ value = "", onChange }) {
  const steps = parseAttackSteps(value);
  const update = (next) => onChange?.(formatAttackSteps(next));
  return (
    <Space orientation="vertical" style={{ width: "100%" }}>
      {steps.map((step, i) => (
        <div
          key={i}
          style={{
            display: "flex",
            gap: 8,
            width: "100%",
            alignItems: "start",
          }}
        >
          <Typography.Text style={{ paddingTop: 5 }}>{i + 1}、</Typography.Text>
          <Input.TextArea
            aria-label={`攻击步骤 ${i + 1}`}
            autoSize={{ minRows: 2, maxRows: 6 }}
            value={step}
            maxLength={10000}
            onChange={(e) =>
              update(steps.map((s, n) => (n === i ? e.target.value : s)))
            }
          />
          <Space orientation="vertical" size={0}>
            <Button
              size="small"
              aria-label={`上移步骤 ${i + 1}`}
              disabled={i === 0}
              onClick={() => {
                const a = [...steps];
                [a[i - 1], a[i]] = [a[i], a[i - 1]];
                update(a);
              }}
            >
              上移
            </Button>
            <Button
              size="small"
              aria-label={`下移步骤 ${i + 1}`}
              disabled={i === steps.length - 1}
              onClick={() => {
                const a = [...steps];
                [a[i + 1], a[i]] = [a[i], a[i + 1]];
                update(a);
              }}
            >
              下移
            </Button>
            <Button
              size="small"
              aria-label={`移除步骤 ${i + 1}`}
              disabled={steps.length === 1}
              onClick={() => update(steps.filter((_, n) => n !== i))}
            >
              移除
            </Button>
          </Space>
        </div>
      ))}
      <Button onClick={() => update([...steps, ""])}>添加步骤</Button>
      <Typography.Text type="secondary">
        可粘贴按 1、2、3、分行的步骤。每一步说明攻击动作、输入条件和预期结果。
      </Typography.Text>
    </Space>
  );
}
export function AttackRoutePicker({ workspace, scenario, form }) {
  const [catalog, setCatalog] = useState(null),
    [entry, setEntry] = useState(""),
    [target, setTarget] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  useEffect(() => {
    setCatalog(null);
    setEntry("");
    setTarget("");
    setError("");
  }, [workspace.id, scenario?.id]);
  const search = async () => {
    setBusy(true);
    setError("");
    try {
      const q = new URLSearchParams({ scenario_id: scenario.id });
      if (entry) q.set("entry_id", entry);
      if (target) q.set("target_id", target);
      const result = await request(
        `/workspaces/${workspace.id}/attack-route-candidates?${q}`,
      );
      if (result.base_revision !== workspace.revision)
        throw Error("项目已变化，请刷新项目后查找");
      setCatalog(result);
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <Card size="small" title="从架构查找候选路径" className="section-card">
      <Space wrap>
        <Select
          aria-label="路径入口"
          value={entry}
          style={{ width: 230 }}
          options={[
            { value: "", label: "全部外部实体" },
            ...(catalog?.nodes || []).map((n) => ({
              value: n.id,
              label: n.name,
            })),
          ]}
          onChange={setEntry}
        />
        <Select
          aria-label="路径目标"
          value={target}
          style={{ width: 230 }}
          options={[
            { value: "", label: "全部关联资产实体" },
            ...(catalog?.targets || []).map((n) => ({
              value: n.id,
              label: n.name,
            })),
          ]}
          onChange={setTarget}
        />
        <Button loading={busy} onClick={search}>
          查找候选路径
        </Button>
      </Space>
      {error && <Alert type="error" title={error} />}
      {catalog && (
        <>
          <Typography.Paragraph type="secondary" style={{ marginTop: 12 }}>
            {catalog.notice}
          </Typography.Paragraph>
          {catalog.truncated && (
            <Alert
              type="warning"
              title={`搜索达到范围限制（最多 ${catalog.max_hops} 段、${catalog.max_candidates} 个候选），可限定入口和目标继续查找。`}
            />
          )}
          {!catalog.items.length && (
            <Alert
              type="info"
              title="未找到连通候选"
              description="检查外部实体类型、数据流方向和资产所在实体；本地接触、供应链等路径可手工录入。"
            />
          )}
          {catalog.items.map((r) => (
            <Card
              key={r.id}
              size="small"
              style={{ marginTop: 8 }}
              title={r.name}
              extra={
                <Button
                  onClick={() =>
                    form.setFieldsValue({
                      architecture_route: r.architecture_route,
                      name: r.name,
                      vector: r.name.split(" → ")[0],
                      steps: r.steps,
                    })
                  }
                >
                  载入候选
                </Button>
              }
            >
              <Space>
                <Tag>{r.shortest ? "最短连通候选" : "协议替代候选"}</Tag>
                <Tag>{r.hops} 段</Tag>
                <Tag>{r.protocol_changes} 次协议切换</Tag>
              </Space>
              <Typography.Paragraph
                style={{ whiteSpace: "pre-wrap", marginTop: 8 }}
              >
                {r.steps}
              </Typography.Paragraph>
            </Card>
          ))}
        </>
      )}
    </Card>
  );
}
export function RouteConfirmation({ value, onChange }) {
  if (!value) return null;
  return (
    <Card size="small" className="section-card" title="架构路径核实">
      <Checkbox
        checked={value.reviewed}
        onChange={(e) => onChange({ ...value, reviewed: e.target.checked })}
      >
        已逐步补充攻击动作，并核实访问权限、前置条件及协议转换条件
      </Checkbox>
      <Typography.Paragraph type="secondary">
        架构连通不代表攻击可行；未核实的候选不能评级。相关节点或数据流变更后需重新核实。
      </Typography.Paragraph>
      <Button onClick={() => onChange(null)}>改为手工路径</Button>
    </Card>
  );
}
