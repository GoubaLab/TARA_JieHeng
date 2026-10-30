import React, { useEffect, useState } from "react";
import {
  Alert,
  Button,
  Card,
  Drawer,
  Empty,
  Form,
  Input,
  Select,
  Space,
  Steps,
  Table,
  Tag,
  Typography,
} from "antd";
import { PageTitle, useWorkbench, Risk } from "./Workbench.jsx";
import { navigate, propOptions, properties } from "./api.js";
import { effectiveMethod } from "./method-config.js";
import { recordCode as numberCode, pathNumberKey } from "./numbering.js";
import {
  recordCode,
  recordLabel,
  recordText,
  pickScenario,
  potentialResult,
} from "./tara-guidance.js";
import { ImpactFields, FeasibilityFields } from "./AssessmentGuidance.jsx";
import { FunctionReferences } from "./FunctionCatalog.jsx";
import AttackPathTemplatePicker from "./AttackPathTemplatePicker.jsx";
import { appendTemplatePaths } from "./attack-path-templates.js";
const { Text, Paragraph } = Typography;
const required = [{ required: true, whitespace: true, message: "请填写此项" }],
  selected = [{ required: true, message: "请选择此项" }];
const arrayRequired = [
  { required: true, type: "array", min: 1, message: "至少选择一项" },
];
export const workflowStages = [
  ["definition", "相关项与功能"],
  ["architecture", "系统架构"],
  ["assets", "资产识别"],
  ["damages", "损害场景"],
  ["impacts", "影响评级"],
  ["threats", "威胁场景"],
  ["paths", "攻击路径"],
  ["feasibility", "攻击可行性"],
  ["assessment", "风险确定"],
  ["treatment", "风险处置与需求"],
  ["verification", "验证与报告"],
];
export function WorkflowSteps() {
  const { ws, page } = useWorkbench(),
    i = workflowStages.findIndex(([key]) => key === page);
  if (!ws || i < 0) return null;
  const start = Math.max(0, Math.min(i - 1, workflowStages.length - 3)),
    visible = workflowStages.slice(start, start + 3);
  return (
    <Card size="small" className="section-card">
      <Steps
        size="small"
        current={i - start}
        onChange={(n) => navigate(visible[n][0], ws.id)}
        items={visible.map(([, title], n) => ({
          title,
          icon: <span>{start + n + 1}</span>,
          status: start + n === i ? "process" : "wait",
        }))}
      />
      <Space wrap style={{ marginTop: 12 }}>
        <Button
          disabled={i === 0}
          onClick={() => navigate(workflowStages[i - 1][0], ws.id)}
        >
          上一步
        </Button>
        <Select
          aria-label="分析步骤"
          value={page}
          style={{ width: 240 }}
          options={workflowStages.map(([value, title], n) => ({
            value,
            label: `${n + 1} / ${workflowStages.length} · ${title}`,
          }))}
          onChange={(key) => navigate(key, ws.id)}
        />
        <Button
          disabled={i === workflowStages.length - 1}
          onClick={() => navigate(workflowStages[i + 1][0], ws.id)}
        >
          下一步
        </Button>
      </Space>
    </Card>
  );
}
const label = (ws, kind, r, parent) => {
  const code =
      (r.source_code && !r.source_code.startsWith("TPL:")
        ? r.source_code
        : null) ||
      numberCode(
        ws,
        kind,
        kind === "path" ? pathNumberKey(parent, r.id) : r.id,
      ),
    text = recordText(r);
  return text === code || text.startsWith(`${code} · `)
    ? text
    : `${code} · ${text}`;
};
function Related({ ws, ids, collection, kind }) {
  return (
    <Space orientation="vertical" size={4}>
      {(ids || []).map((id) => {
        const r = (ws[collection] || []).find((v) => v.id === id);
        return (
          <div key={id}>{r ? label(ws, kind, r) : `${id} · 引用内容缺失`}</div>
        );
      })}
    </Space>
  );
}
export function DamageEditor({ editor, onClose }) {
  const { ws, save, message } = useWorkbench(),
    record = (ws.damage_scenarios || []).find((r) => r.id === editor.id),
    rating = editor.kind === "impact",
    [form] = Form.useForm(),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  useEffect(() => {
    form.resetFields();
    form.setFieldsValue(
      record || {
        asset_ids: editor.assetId ? [editor.assetId] : [],
        properties: editor.assetId
          ? ws.assets.find((a) => a.id === editor.assetId)?.properties || [
              "integrity",
            ]
          : ["integrity"],
        function_refs: [],
        impact: {},
        description: "",
        consequence: "",
      },
    );
  }, [editor]);
  const submit = async () => {
    try {
      const v = await form.validateFields();
      setBusy(true);
      setError("");
      const body = {
        ...record,
        ...v,
        impact: rating ? v.impact : record?.impact || {},
      };
      const keys = [
        "id",
        "source_code",
        "description",
        "consequence",
        "asset_ids",
        "function_refs",
        "properties",
        "impact",
      ];
      await save(
        "record-damage",
        Object.fromEntries(
          keys.filter((k) => body[k] !== undefined).map((k) => [k, body[k]]),
        ),
      );
      message.success(
        rating
          ? "影响评级已保存，关联风险已重新计算"
          : "损害场景已保存，请继续完成影响评级",
      );
      onClose();
    } catch (e) {
      if (!e.errorFields) setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <Drawer
      open
      title={rating ? "影响评级" : "损害场景识别"}
      size={820}
      onClose={() => !busy && onClose()}
      extra={
        <Button type="primary" loading={busy} onClick={submit}>
          {rating ? "保存影响评级" : "保存损害场景"}
        </Button>
      }
    >
      {error && <Alert type="error" title={error} className="section-card" />}
      <Form form={form} layout="vertical" requiredMark>
        {rating ? (
          <>
            <Alert
              className="section-card"
              title={label(ws, "damage", record)}
              description={record?.consequence}
            />
            <ImpactFields ws={ws} form={form} />
          </>
        ) : (
          <>
            <Form.Item name="source_code" label="业务编号">
              <Input maxLength={100} />
            </Form.Item>
            <Form.Item name="asset_ids" label="关联资产" rules={arrayRequired}>
              <Select
                mode="multiple"
                showSearch
                optionFilterProp="label"
                options={ws.assets.map((r) => ({
                  value: r.id,
                  label: label(ws, "asset", r),
                }))}
              />
            </Form.Item>
            <Form.Item name="function_refs" label="关联功能">
              <FunctionReferences ws={ws} />
            </Form.Item>
            <Form.Item name="description" label="损害场景" rules={required}>
              <Input.TextArea
                rows={4}
                maxLength={10000}
                placeholder="说明资产的安全属性受损后，在何种使用情景下可能造成损害。"
              />
            </Form.Item>
            <Form.Item
              name="consequence"
              label="受影响对象与不利后果"
              rules={required}
            >
              <Input.TextArea rows={4} maxLength={10000} />
            </Form.Item>
            <Form.Item
              name="properties"
              label="受损安全属性"
              rules={arrayRequired}
            >
              <Select mode="multiple" options={propOptions} />
            </Form.Item>
            <Alert
              type="info"
              title="下一步：影响评级"
              description="保存后在影响评级步骤逐项判断安全、财务、运行和隐私后果。"
            />
          </>
        )}
      </Form>
    </Drawer>
  );
}
export function ScenarioStageEditor({ editor, onClose }) {
  const { ws, save, message } = useWorkbench(),
    [form] = Form.useForm(),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [templates, setTemplates] = useState(false);
  const stage = editor.step || "threat",
    s = ws.scenarios.find((r) => r.id === editor.id),
    p = s?.paths?.find((r) => r.id === editor.pathId);
  const linkedIds = Form.useWatch("damage_ids", form) || [],
    assets = Form.useWatch("asset_ids", form) || [];
  const options = (rows, kind) =>
    (rows || []).map((r) => ({ value: r.id, label: label(ws, kind, r) }));
  useEffect(() => {
    form.resetFields();
    setError("");
    if (stage === "threat")
      form.setFieldsValue(
        s
          ? {
              ...s,
              asset_ids: s.asset_ids || [s.asset_id],
              properties: s.properties || [s.property],
            }
          : {
              asset_ids: editor.assetId ? [editor.assetId] : [],
              properties: [],
              damage_ids: [],
            },
      );
    else if (stage === "path") form.setFieldsValue(p || {});
    else if (stage === "feasibility")
      form.setFieldsValue({
        method:
          p?.feasibility_method ||
          (p?.potential ? "attack_potential" : undefined),
        potential: p?.potential,
        attack_vector: p?.attack_vector,
        rationale: p?.rationale || "",
      });
    else form.setFieldsValue({ impact: s?.impact });
  }, [editor]);
  const submit = async () => {
    try {
      const v = await form.validateFields();
      setBusy(true);
      setError("");
      if (stage === "feasibility") {
        await save("rate-attack-path", {
          scenario_id: s.id,
          path_id: p.id,
          method: v.method,
          rationale: v.rationale,
          ...(v.method === "attack_potential"
            ? { potential: v.potential }
            : { attack_vector: v.attack_vector }),
        });
      } else {
        let body = pickScenario(
          s || { references: [], paths: [], impact: {}, damage: "" },
        );
        if (stage === "threat") {
          body = { ...body, ...v };
          body.property = v.properties[0];
          body.asset_id =
            ws.assets.find(
              (a) =>
                v.asset_ids.includes(a.id) &&
                a.properties.includes(body.property),
            )?.id || v.asset_ids[0];
          const damages = (ws.damage_scenarios || []).filter((d) =>
            v.damage_ids?.includes(d.id),
          );
          if (damages.length)
            body.damage = damages.map((d) => d.description).join("\n\n");
        } else if (stage === "path") {
          const changed =
            !p ||
            ["name", "steps", "actor", "vector"].some(
              (k) => (p[k] || "") !== (v[k] || ""),
            );
          const path = {
            ...p,
            ...v,
            id: p?.id || "path_" + crypto.randomUUID(),
            reference: p?.reference || null,
          };
          if (changed) {
            path.feasibility = null;
            path.rationale = "";
            delete path.potential;
            delete path.feasibility_method;
            delete path.attack_vector;
          }
          body.paths = p
            ? body.paths.map((x) => (x.id === p.id ? path : x))
            : [...body.paths, path];
        } else body.impact = v.impact;
        await save("record-scenario", body);
      }
      message.success(
        stage === "path"
          ? "攻击路径已保存，请完成可行性评级"
          : "已保存，关联风险已重新计算",
      );
      onClose();
    } catch (e) {
      if (!e.errorFields) setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  const titles = {
    threat: "威胁场景识别",
    path: "攻击路径分析",
    feasibility: "攻击可行性评级",
    impact: "影响评级",
  };
  return (
    <Drawer
      open
      title={titles[stage]}
      size={820}
      onClose={() => !busy && onClose()}
      extra={
        <Button type="primary" loading={busy} onClick={submit}>
          保存{titles[stage]}
        </Button>
      }
    >
      {error && <Alert type="error" title={error} className="section-card" />}
      {s && stage !== "threat" && (
        <Card
          size="small"
          className="section-card"
          title={label(ws, "scenario", s)}
        >
          <Paragraph>{s.threat}</Paragraph>
          {p && (
            <>
              <Text strong>{label(ws, "path", p, s.id)}</Text>
              <Paragraph style={{ whiteSpace: "pre-wrap" }}>
                {p.steps}
              </Paragraph>
            </>
          )}
        </Card>
      )}
      <Form form={form} layout="vertical" requiredMark>
        {stage === "threat" && (
          <>
            <Form.Item name="source_code" label="业务编号">
              <Input maxLength={100} />
            </Form.Item>
            <Form.Item name="title" label="威胁场景名称" rules={required}>
              <Input maxLength={300} />
            </Form.Item>
            <Form.Item name="asset_ids" label="关联资产" rules={arrayRequired}>
              <Select
                mode="multiple"
                showSearch
                optionFilterProp="label"
                options={options(ws.assets, "asset")}
              />
            </Form.Item>
            <Form.Item
              name="properties"
              label="受损安全属性"
              rules={arrayRequired}
            >
              <Select
                mode="multiple"
                options={propOptions.filter((o) =>
                  ws.assets
                    .filter((a) => assets.includes(a.id))
                    .some((a) => a.properties.includes(o.value)),
                )}
              />
            </Form.Item>
            <Form.Item
              name="damage_ids"
              label="关联损害场景"
              rules={s?.damage && !s.damage_ids?.length ? [] : arrayRequired}
            >
              <Select
                mode="multiple"
                showSearch
                optionFilterProp="label"
                options={options(ws.damage_scenarios, "damage")}
              />
            </Form.Item>
            <Related
              ws={ws}
              ids={linkedIds}
              collection="damage_scenarios"
              kind="damage"
            />
            {s?.damage && !linkedIds.length && (
              <Form.Item name="damage" label="损害描述" rules={required}>
                <Input.TextArea rows={3} />
              </Form.Item>
            )}
            <Form.Item name="threat" label="威胁描述" rules={required}>
              <Input.TextArea
                rows={4}
                maxLength={10000}
                placeholder="说明攻击者怎样破坏资产安全属性，并导致关联的损害。"
              />
            </Form.Item>
            <Form.Item name="cause" label="攻击原因与前提">
              <Input.TextArea rows={3} maxLength={10000} />
            </Form.Item>
            <Form.Item name="stride" label="STRIDE 分类">
              <Select
                mode={undefined}
                allowClear
                options={[
                  "Spoofing",
                  "Tampering",
                  "Repudiation",
                  "Information Disclosure",
                  "Denial of Service",
                  "Elevation of Privilege",
                ].map((value) => ({ value, label: value }))}
              />
            </Form.Item>
            <Alert
              type="info"
              title="后续步骤"
              description="影响评级取自关联损害场景；攻击路径和可行性在各自步骤中维护。"
            />
          </>
        )}
        {stage === "path" && (
          <>
            <Form.Item name="source_code" label="业务编号">
              <Input maxLength={100} />
            </Form.Item>
            <Form.Item name="name" label="路径名称" rules={required}>
              <Input maxLength={500} />
            </Form.Item>
            <Form.Item name="actor" label="攻击者与能力假设">
              <Input maxLength={500} />
            </Form.Item>
            <Form.Item name="vector" label="入口或攻击向量">
              <Input maxLength={200} />
            </Form.Item>
            <Form.Item name="steps" label="攻击步骤与必要前提" rules={required}>
              <Input.TextArea rows={9} maxLength={10000} />
            </Form.Item>
            <Alert
              type="info"
              title="路径条件变更后须重新评级"
              description="修改步骤、攻击者或入口后，原可行性评级将清除，风险恢复待评估。"
            />
          </>
        )}
        {stage === "feasibility" && <FeasibilityFields ws={ws} form={form} />}
        {stage === "impact" && <ImpactFields ws={ws} form={form} />}
      </Form>
    </Drawer>
  );
}
export default function TaraWorkflow({ stage }) {
  const { ws, setEditor, review, save, message } = useWorkbench(),
    [editing, setEditing] = useState(null),
    [templateScenario, setTemplateScenario] = useState(null),
    [q, setQ] = useState("");
  useEffect(() => {
    setEditing(null);
    setTemplateScenario(null);
    setQ("");
  }, [stage, ws.id]);
  const readOnly = ws.access_role === "viewer",
    method = effectiveMethod(ws.method);
  const paths = ws.scenarios.flatMap((s) =>
      (s.paths || []).map((p) => ({ ...p, scenario_id: s.id, scenario: s })),
    ),
    damages = ws.damage_scenarios || [];
  const open = (r, step) =>
    setEditor({
      kind: "scenario",
      id: r.scenario_id || r.id,
      pathId: r.scenario_id ? r.id : undefined,
      step,
    });
  const textColumn = (title, key, width = 260) => ({
    title,
    dataIndex: key,
    width,
    render: (v) => <div style={{ whiteSpace: "pre-wrap" }}>{v || "—"}</div>,
  });
  let rows = [],
    columns = [],
    extra = null,
    description = "";
  if (stage === "damages" || stage === "impacts") {
    rows = damages;
    if (stage === "impacts")
      rows = [
        ...rows,
        ...ws.scenarios
          .filter((s) => !s.damage_ids?.length)
          .map((s) => ({ ...s, description: s.damage, embedded: true })),
      ];
    description =
      stage === "damages"
        ? "先识别受影响对象、使用情景和不利后果，再关联资产与功能。"
        : "按安全、财务、运行和隐私分别评定损害后果，记录判断准则与证据。";
    columns = [
      {
        title: "编号",
        width: 130,
        render: (_, r) =>
          r.source_code ||
          numberCode(ws, r.embedded ? "scenario" : "damage", r.id),
      },
      textColumn("损害场景", "description", 340),
      textColumn("不利后果", "consequence", 260),
      {
        title: "关联资产",
        width: 300,
        render: (_, r) => (
          <Related
            ws={ws}
            ids={r.asset_ids || [r.asset_id]}
            collection="assets"
            kind="asset"
          />
        ),
      },
      ...(stage === "impacts"
        ? [
            {
              title: "S / F / O / P",
              width: 260,
              render: (_, r) => (
                <Space wrap>
                  {["safety", "financial", "operational", "privacy"].map(
                    (key, i) => (
                      <Tag key={key}>
                        {"SFOP"[i]} ·{" "}
                        {method.impact_levels[r.impact?.[key]]?.label ||
                          "待评级"}
                      </Tag>
                    ),
                  )}
                </Space>
              ),
            },
            textColumn("评级依据", "impact", 280),
          ]
        : []),
      {
        title: "操作",
        width: 120,
        fixed: "right",
        render: (_, r) => (
          <Button
            type="link"
            disabled={readOnly}
            onClick={() =>
              r.embedded
                ? setEditor({ kind: "scenario", id: r.id, step: "impact" })
                : setEditing({
                    kind: stage === "impacts" ? "impact" : "damage",
                    id: r.id,
                  })
            }
          >
            {stage === "impacts" ? "影响评级" : "编辑场景"}
          </Button>
        ),
      },
    ];
    if (stage === "impacts")
      columns.find((c) => c.dataIndex === "impact").render = (v) =>
        v?.rationale || "待补充";
    if (stage === "damages")
      extra = (
        <Button
          type="primary"
          disabled={readOnly || !ws.assets.length}
          onClick={() => setEditing({ kind: "damage" })}
        >
          新增损害场景
        </Button>
      );
  } else if (stage === "threats") {
    rows = ws.scenarios;
    description = "明确资产如何受到破坏，关联已识别的损害场景。";
    columns = [
      {
        title: "威胁场景",
        width: 300,
        render: (_, r) => label(ws, "scenario", r),
      },
      textColumn("威胁描述", "threat", 380),
      {
        title: "资产",
        width: 280,
        render: (_, r) => (
          <Related
            ws={ws}
            ids={r.asset_ids || [r.asset_id]}
            collection="assets"
            kind="asset"
          />
        ),
      },
      {
        title: "损害场景",
        width: 350,
        render: (_, r) =>
          r.damage_ids?.length ? (
            <Related
              ws={ws}
              ids={r.damage_ids}
              collection="damage_scenarios"
              kind="damage"
            />
          ) : (
            r.damage
          ),
      },
      {
        title: "操作",
        fixed: "right",
        width: 120,
        render: (_, r) => (
          <Button
            type="link"
            disabled={readOnly}
            onClick={() => open(r, "threat")}
          >
            编辑威胁
          </Button>
        ),
      },
    ];
    extra = (
      <Button
        type="primary"
        disabled={readOnly || !damages.length}
        onClick={() => setEditor({ kind: "scenario", step: "threat" })}
      >
        新增威胁场景
      </Button>
    );
  } else {
    rows = paths;
    description =
      stage === "paths"
        ? "为每个威胁场景建立实现路径，明确攻击步骤、入口和前提。"
        : "逐条评定攻击路径。全部路径完成评级和依据后，才能形成风险值。";
    columns = [
      {
        title: "攻击路径",
        width: 300,
        render: (_, r) => label(ws, "path", r, r.scenario_id),
      },
      {
        title: "关联威胁",
        width: 300,
        render: (_, r) => label(ws, "scenario", r.scenario),
      },
      textColumn("攻击步骤与前提", "steps", 400),
      {
        title: "可行性",
        width: 170,
        render: (_, r) => (
          <Tag color={r.feasibility && r.rationale ? "blue" : "orange"}>
            {r.feasibility && r.rationale
              ? method.feasibility_levels[
                  (r.potential
                    ? potentialResult(r.potential, method)?.level ||
                      r.feasibility
                    : r.feasibility) - 1
                ]?.label || "待评级"
              : "待评级"}
          </Tag>
        ),
      },
      textColumn("评级依据", "rationale", 300),
      {
        title: "操作",
        width: 120,
        fixed: "right",
        render: (_, r) => (
          <Button
            type="link"
            disabled={readOnly || (stage === "feasibility" && !r.steps)}
            onClick={() => open(r, stage === "paths" ? "path" : "feasibility")}
          >
            {stage === "paths" ? "编辑路径" : "评定可行性"}
          </Button>
        ),
      },
    ];
    if (stage === "paths")
      extra = (
        <Space>
          <Select
            placeholder="选择威胁后新增路径"
            style={{ width: 300 }}
            value={null}
            showSearch
            optionFilterProp="label"
            disabled={readOnly}
            options={ws.scenarios.map((s) => ({
              value: s.id,
              label: label(ws, "scenario", s),
            }))}
            onChange={(id) => setEditor({ kind: "scenario", id, step: "path" })}
          />
          <Select
            placeholder="选择威胁并选用路径模板"
            style={{ width: 300 }}
            value={null}
            showSearch
            optionFilterProp="label"
            disabled={readOnly}
            options={ws.scenarios.map((s) => ({
              value: s.id,
              label: label(ws, "scenario", s),
            }))}
            onChange={(id) =>
              setTemplateScenario(ws.scenarios.find((s) => s.id === id))
            }
          />
        </Space>
      );
  }
  return (
    <>
      <PageTitle
        title={workflowStages.find(([key]) => key === stage)?.[1]}
        description={description}
        extra={extra}
      />
      {!rows.length && (
        <Alert
          className="section-card"
          type="info"
          showIcon
          title={
            stage === "threats"
              ? "请先完成资产与损害场景识别"
              : stage === "impacts"
                ? "请先建立损害场景"
                : stage === "paths"
                  ? "请先建立威胁场景，再添加攻击路径"
                  : stage === "feasibility"
                    ? "请先建立攻击路径"
                    : "请先识别资产，再描述损害场景"
          }
        />
      )}
      <Card>
        <Input.Search
          className="section-card"
          placeholder="按编号或内容搜索"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          allowClear
          style={{ maxWidth: 420 }}
        />
        <Table
          rowKey={(r) => (r.scenario_id ? `${r.scenario_id}/${r.id}` : r.id)}
          size="small"
          dataSource={rows.filter((r) =>
            JSON.stringify(r).toLowerCase().includes(q.toLowerCase()),
          )}
          columns={columns}
          pagination={{
            pageSize: 10,
            showSizeChanger: true,
            showTotal: (n) => `共 ${n} 条`,
          }}
          scroll={{ x: columns.reduce((s, c) => s + (c.width || 240), 0) }}
        />
      </Card>
      {editing && (
        <DamageEditor editor={editing} onClose={() => setEditing(null)} />
      )}
      {templateScenario && (
        <AttackPathTemplatePicker
          open
          workspace={ws}
          assetId={templateScenario.asset_id}
          property={templateScenario.property}
          existingPaths={templateScenario.paths}
          onClose={() => setTemplateScenario(null)}
          onSelect={async (rows) => {
            try {
              const next = appendTemplatePaths(templateScenario.paths, rows);
              await save("record-scenario", {
                ...pickScenario(templateScenario),
                paths: next.paths,
              });
              message.success("攻击路径已添加，请核对步骤并评定可行性");
              setTemplateScenario(null);
            } catch (e) {
              message.error(e.message);
            }
          }}
        />
      )}
    </>
  );
}
