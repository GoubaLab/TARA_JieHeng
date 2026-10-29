import React, { useEffect, useState, useRef } from "react";
import {
  Alert,
  Button,
  Card,
  Col,
  Descriptions,
  Drawer,
  Empty,
  Form,
  Image,
  Input,
  InputNumber,
  Row,
  Select,
  Space,
  Statistic,
  Table,
  Tabs,
  Tag,
  Typography,
  Upload,
} from "antd";
import {
  UploadOutlined,
  DownloadOutlined,
  PlusOutlined,
} from "@ant-design/icons";
import { PageTitle, useWorkbench } from "./Workbench.jsx";
import {
  apiRoot,
  properties,
  propOptions,
  request,
  treatments,
} from "./api.js";
import { effectiveMethod } from "./method-config.js";
const { Text } = Typography;
const pagination = {
  defaultPageSize: 10,
  showSizeChanger: true,
  showTotal: (n) => `共 ${n} 条`,
};
const code = (r) => r.source_code || r.id;
const joined = (values) => (values || []).join("、") || "—";
const propsText = (values) =>
  joined((values || []).map((v) => properties[v] || v));
const factors = [
  ["elapsed_time", "ET · 所需时间"],
  ["expertise", "Ex · 专业知识"],
  ["knowledge", "KoI · 对相关项的了解"],
  ["opportunity", "WoO · 机会窗口"],
  ["equipment", "Eq · 设备"],
];
const col = (title, key, width = 240, render) => ({
  title,
  dataIndex: key,
  key: `${key}-${title}`,
  width,
  render: render || ((v) => v || "—"),
});
function Records({ rows, columns, onEdit }) {
  return (
    <Table
      rowKey={(r) => r.id || r.scenario_id}
      dataSource={rows || []}
      columns={[
        ...columns,
        ...(onEdit
          ? [
              {
                title: "操作",
                key: "edit",
                width: 80,
                fixed: "right",
                render: (_, r) => (
                  <Button type="link" onClick={() => onEdit(r)}>
                    编辑
                  </Button>
                ),
              },
            ]
          : []),
      ]}
      pagination={pagination}
      scroll={{
        x:
          columns.reduce((s, c) => s + (c.width || 240), 0) + (onEdit ? 80 : 0),
      }}
      size="small"
    />
  );
}
function ProductEditor({ kind, record, onClose }) {
  const { ws, save, message } = useWorkbench();
  const [form] = Form.useForm();
  const [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const damage = kind === "damage";
  const method = effectiveMethod(ws.method);
  const required = [{ required: true, message: "请填写此项" }];
  useEffect(() => {
    form.resetFields();
    form.setFieldsValue(
      record ||
        (damage
          ? {
              properties: ["integrity"],
              asset_ids: [],
              function_refs: [],
              impact: {},
            }
          : { scenario_ids: [], cal: 1 }),
    );
  }, [record, kind]);
  const submit = async () => {
    try {
      await form.validateFields();
      setBusy(true);
      setError("");
      const values = form.getFieldsValue(true);
      const keys = damage
        ? [
            "source_code",
            "description",
            "consequence",
            "asset_ids",
            "function_refs",
            "properties",
            "impact",
          ]
        : ["source_code", "statement", "scenario_ids", "cal", "theme"];
      const payload = Object.fromEntries(
        keys.filter((k) => values[k] !== undefined).map((k) => [k, values[k]]),
      );
      if (record?.id) payload.id = record.id;
      await save(damage ? "record-damage" : "record-goal", payload);
      message.success("工作产品已保存，关联风险已重新计算");
      onClose();
    } catch (e) {
      if (!e.errorFields) setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <Drawer
      title={damage ? "损害场景" : "网络安全目标"}
      open
      onClose={() => !busy && onClose()}
      size={720}
      extra={
        <Button type="primary" loading={busy} onClick={submit}>
          保存
        </Button>
      }
    >
      {error && <Alert type="error" title={error} showIcon />}
      <Form form={form} layout="vertical">
        <Form.Item name="source_code" label="业务编号">
          <Input maxLength={100} />
        </Form.Item>
        {damage ? (
          <>
            <Form.Item name="description" label="损害场景" rules={required}>
              <Input.TextArea rows={4} />
            </Form.Item>
            <Form.Item name="consequence" label="道路使用者及其他相关方后果">
              <Input.TextArea rows={3} />
            </Form.Item>
            <Form.Item name="asset_ids" label="关联资产" rules={required}>
              <Select
                mode="multiple"
                options={ws.assets.map((a) => ({
                  value: a.id,
                  label: `${code(a)} · ${a.name}`,
                }))}
                optionFilterProp="label"
              />
            </Form.Item>
            <Form.Item name="function_refs" label="关联功能编号">
              <Select mode="tags" />
            </Form.Item>
            <Form.Item name="properties" label="受损安全属性" rules={required}>
              <Select mode="multiple" options={propOptions} />
            </Form.Item>
            <Alert
              type="info"
              title="修改影响评级后，所有引用此损害场景的威胁将重新计算；已有处置和验证需重新评审。"
            />
            <Row gutter={16}>
              {["safety", "financial", "operational", "privacy"].map((k, i) => (
                <Col span={12} key={k}>
                  <Form.Item
                    name={["impact", k]}
                    label={`${"SFOP"[i]} · ${method.impact_dimensions[k]}`}
                    rules={required}
                  >
                    <Select
                      options={method.impact_levels.map((v, value) => ({
                        value,
                        label: `${v.label}${method.impact_aggregation === "coefficient_bands" && method.impact_scale ? ` · ${method.impact_scale[value]}` : ""}`,
                      }))}
                    />
                  </Form.Item>
                </Col>
              ))}
            </Row>
            <Form.Item
              name={["impact", "rationale"]}
              label="评级依据"
              rules={required}
            >
              <Input.TextArea rows={3} />
            </Form.Item>
          </>
        ) : (
          <>
            <Form.Item name="statement" label="安全目标" rules={required}>
              <Input.TextArea rows={5} />
            </Form.Item>
            <Form.Item
              name="scenario_ids"
              label="关联威胁场景"
              rules={required}
            >
              <Select
                mode="multiple"
                options={ws.scenarios.map((s) => ({
                  value: s.id,
                  label: `${code(s)} · ${s.title}`,
                }))}
                optionFilterProp="label"
              />
            </Form.Item>
            <Form.Item name="cal" label="CAL" rules={required}>
              <InputNumber min={1} max={4} precision={0} />
            </Form.Item>
            <Form.Item name="theme" label="控制主题">
              <Input />
            </Form.Item>
          </>
        )}
      </Form>
    </Drawer>
  );
}
function OriginalSheets({ source }) {
  const [selected, setSelected] = useState(source.sheets[0]?.name);
  const sheet =
    source.sheets.find((s) => s.name === selected) || source.sheets[0];
  if (!sheet) return <Empty />;
  const width = Math.max(...sheet.rows.map((r) => r.length), 1);
  const letter = (i) => {
    let out = "";
    for (i++; i; i = Math.floor((i - 1) / 26))
      out = String.fromCharCode(65 + ((i - 1) % 26)) + out;
    return out;
  };
  return (
    <Space orientation="vertical" size="middle" style={{ width: "100%" }}>
      <Space wrap>
        <Select
          aria-label="选择原始表页"
          value={sheet.name}
          onChange={setSelected}
          style={{ minWidth: 280 }}
          options={source.sheets.map((s, i) => ({
            value: s.name,
            label: `${i + 1}. ${s.name}`,
          }))}
        />
        <Text type="secondary">
          原表只读 · {source.sheets.length} 个表页 ·
          保留合并范围和公式记录；原始排版见源文件。
        </Text>
      </Space>
      <Table
        size="small"
        rowKey="row"
        pagination={false}
        scroll={{ x: 60 + width * 230, y: 620 }}
        dataSource={sheet.rows.map((values, i) => ({ row: i + 1, values }))}
        columns={[
          { title: "行", dataIndex: "row", width: 60, fixed: "left" },
          ...Array.from({ length: width }, (_, i) => ({
            title: letter(i),
            key: i,
            width: 230,
            render: (_, r) => (
              <div style={{ whiteSpace: "pre-wrap", overflowWrap: "anywhere" }}>
                {r.values[i] || ""}
              </div>
            ),
          })),
        ]}
      />
      {sheet.name === "数据流图与数据流" &&
        source.images?.map((img, i) => (
          <Card key={i} title="源工作簿中的数据流图">
            <Image
              alt="源工作簿数据流图"
              src={`data:${img.mime};base64,${img.content_base64}`}
            />
          </Card>
        ))}
    </Space>
  );
}
export default function TaraProducts() {
  const { ws, review, save, setEditor, message } = useWorkbench();
  const previewAttempt = useRef(0);
  const [payload, setPayload] = useState(null),
    [preview, setPreview] = useState(null),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [editing, setEditing] = useState(null);
  useEffect(() => {
    previewAttempt.current++;
    setPayload(null);
    setPreview(null);
    setError("");
    setBusy(false);
    setEditing(null);
  }, [ws.id]);
  const source = ws.tara_import;
  const canImport =
    !source &&
    ![
      ws.assets,
      ws.scenarios,
      ws.requirements,
      ws.evidence,
      ws.baselines,
      ws.damage_scenarios,
      ws.security_goals,
      ws.decisions,
    ].some((a) => a?.length);
  const inspect = async (file) => {
    const attempt = ++previewAttempt.current;
    setBusy(true);
    setError("");
    setPreview(null);
    setPayload(null);
    try {
      if (file.size > 1400000) throw new Error("当前导入上限为 1.4 MB");
      const content = await new Promise((resolve, reject) => {
        const r = new FileReader();
        r.onload = () => resolve(r.result.split(",")[1]);
        r.onerror = reject;
        r.readAsDataURL(file);
      });
      const p = { filename: file.name, content_base64: content };
      const result = await request(
        `/workspaces/${encodeURIComponent(ws.id)}/tara-import-previews`,
        { method: "POST", body: JSON.stringify(p) },
      );
      if (attempt !== previewAttempt.current) return false;
      setPayload(p);
      setPreview(result);
    } catch (e) {
      if (attempt === previewAttempt.current)
        setError(e.message || "文件读取失败");
    } finally {
      if (attempt === previewAttempt.current) setBusy(false);
    }
    return false;
  };
  const apply = async () => {
    setBusy(true);
    try {
      await save("import-tara", payload);
      setPreview(null);
      setPayload(null);
      message.success("完整 TARA 已导入，原表已归档");
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  const edit = (kind) => (row) =>
    setEditor({ kind, id: row.id || row.scenario_id });
  const pathRows = ws.scenarios.flatMap((s) =>
    (s.paths || []).map((p) => ({ ...p, scenario_id: s.id })),
  );
  const method = effectiveMethod(ws.method);
  const baseCode = col("编号", "id", 110, (_, r) => code(r));
  const metrics = [
    ["资产", ws.assets.length],
    ["损害场景", ws.damage_scenarios?.length || 0],
    ["威胁场景", ws.scenarios.length],
    ["攻击路径", pathRows.length],
    ["安全目标", ws.security_goals?.length || 0],
    ["安全需求", ws.requirements.length],
  ];
  return (
    <>
      <PageTitle
        title="TARA 工作产品"
        description="从资产、损害场景追溯到威胁、攻击路径、安全目标与需求；保留来源，持续维护。"
        extra={
          source && (
            <Button
              icon={<DownloadOutlined />}
              href={`${apiRoot}/workspaces/${encodeURIComponent(ws.id)}/tara-imports/${encodeURIComponent(source.id)}/source`}
            >
              下载原始 Excel
            </Button>
          )
        }
      />
      {error && (
        <Alert type="error" title={error} showIcon className="section-card" />
      )}
      {!source && (
        <Card className="section-card" title="完整导入 TARA 工作簿">
          <Space orientation="vertical">
            <Text>
              支持约定的中文 TARA
              表结构：核对关联关系和算法后，将工作产品写入平台，并保留全部原表与原始文件。适配版本：tara-chinese-v1。
            </Text>
            {!canImport && (
              <Alert
                type="info"
                title="请在空白项目导入，以保留当前项目的已有分析。"
              />
            )}
            <Upload
              accept=".xlsx"
              beforeUpload={inspect}
              showUploadList={false}
              disabled={!canImport || busy}
            >
              <Button
                icon={<UploadOutlined />}
                loading={busy}
                disabled={!canImport}
              >
                选择 Excel 并核对
              </Button>
            </Upload>
            {preview && (
              <>
                <Descriptions
                  size="small"
                  column={3}
                  items={[
                    ["表页", preview.sheets],
                    ["资产", preview.assets],
                    ["损害", preview.damages],
                    ["威胁", preview.scenarios],
                    ["路径", preview.paths],
                    ["目标", preview.goals],
                    ["需求", preview.requirements],
                  ].map(([label, children]) => ({
                    key: label,
                    label,
                    children,
                  }))}
                />
                <Alert
                  type="success"
                  title="字段、关联关系及计算结果核对通过"
                  description="导入采用工作簿中的评估方法。残余风险记为目标值，验证与签署待完成。"
                />
                <Button type="primary" loading={busy} onClick={apply}>
                  导入到当前项目
                </Button>
              </>
            )}
          </Space>
        </Card>
      )}
      <Row gutter={[12, 12]} className="section-card">
        {metrics.map(([title, value]) => (
          <Col xs={12} xl={4} key={title}>
            <Card size="small">
              <Statistic title={title} value={value} />
            </Card>
          </Col>
        ))}
      </Row>
      {source && (
        <Alert
          className="section-card"
          type="info"
          showIcon
          title={`${source.filename} · ${source.sheets.length} 个原始表页已保留`}
          description="工作产品支持编辑，来源表页保留为只读快照。残余风险为待验证目标，负责人和签署信息按实际情况补充。资产沿用导入清单。"
        />
      )}
      <Card>
        <Tabs
          items={[
            {
              key: "assets",
              label: "资产",
              children: (
                <Records
                  rows={ws.assets}
                  onEdit={edit("asset")}
                  columns={[
                    baseCode,
                    col("资产", "name"),
                    col("类型", "asset_type", 120),
                    col("分组", "group", 120),
                    col("关联功能", "function_refs", 220, joined),
                    col("安全属性", "properties", 240, propsText),
                    col("DFD 关联", "node_ids", 240, joined),
                  ]}
                />
              ),
            },
            {
              key: "damages",
              label: "损害场景",
              children: (
                <>
                  <Button
                    icon={<PlusOutlined />}
                    onClick={() => setEditing({ kind: "damage" })}
                  >
                    新增损害场景
                  </Button>
                  <Records
                    rows={ws.damage_scenarios}
                    onEdit={(record) => setEditing({ kind: "damage", record })}
                    columns={[
                      baseCode,
                      col("损害场景", "description", 300),
                      col("后果", "consequence", 300),
                      col("关联资产", "asset_ids", 220, joined),
                      col("功能", "function_refs", 180, joined),
                      col("受损属性", "properties", 200, propsText),
                      col("SFOP 原始取值", "impact", 180, (v) =>
                        ["safety", "financial", "operational", "privacy"]
                          .map(
                            (k) =>
                              (method.impact_aggregation === "coefficient_bands"
                                ? method.impact_scale?.[v[k]]
                                : v[k]) ?? "待评定",
                          )
                          .join(" / "),
                      ),
                      col("评级依据", "impact", 280, (v) => v.rationale),
                    ]}
                  />
                </>
              ),
            },
            {
              key: "threats",
              label: "威胁场景",
              children: (
                <Records
                  rows={ws.scenarios}
                  onEdit={edit("scenario")}
                  columns={[
                    baseCode,
                    col("威胁描述", "threat", 360),
                    col("攻击原因", "cause", 260),
                    col("STRIDE", "stride", 140),
                    col("受损属性", "properties", 240, (v, r) =>
                      propsText(v || [r.property]),
                    ),
                    col("资产", "asset_ids", 220, (v, r) =>
                      joined(v || [r.asset_id]),
                    ),
                    col("损害场景", "damage_ids", 200, joined),
                    col("攻击路径", "paths", 220, (v) => joined(v.map(code))),
                  ]}
                />
              ),
            },
            {
              key: "paths",
              label: "攻击路径",
              children: (
                <Records
                  rows={pathRows}
                  onEdit={(p) =>
                    setEditor({ kind: "scenario", id: p.scenario_id })
                  }
                  columns={[
                    baseCode,
                    col("威胁", "scenario_id", 110),
                    col("攻击者", "actor", 180),
                    col("向量", "vector", 180),
                    col("攻击步骤", "steps", 420),
                    ...factors.map(([key, label]) =>
                      col(
                        label,
                        `factor_${key}`,
                        140,
                        (_, r) => r.potential?.[key] ?? "—",
                      ),
                    ),
                    col("AP 总值", "potential", 100, (v) =>
                      v ? Object.values(v).reduce((a, b) => a + b, 0) : "—",
                    ),
                    col(
                      "可行性",
                      "feasibility",
                      110,
                      (v) =>
                        method.feasibility_levels[v - 1]?.label || "待评级",
                    ),
                  ]}
                />
              ),
            },
            {
              key: "decisions",
              label: "处置与接受条件",
              children: (
                <Records
                  rows={ws.decisions.map((d) => {
                    const ev = review.evaluations.find(
                      (e) => e.scenario_id === d.scenario_id,
                    );
                    return {
                      ...d,
                      residual_status: ev?.decision_stale
                        ? "stale"
                        : ev?.residual_status || d.residual_status,
                    };
                  })}
                  onEdit={edit("decision")}
                  columns={[
                    col("威胁", "scenario_id", 110),
                    col("处置", "treatment", 110, (v) => treatments[v]),
                    col("控制措施", "controls", 380),
                    col("目标", "goal_ids", 180, joined),
                    col("CAL", "cal", 70),
                    col("残余依据", "residual_rationale", 380),
                    col("残余状态", "residual_status", 150, (v) => (
                      <Tag
                        color={
                          v === "target" || v === "stale" ? "orange" : undefined
                        }
                      >
                        {v === "target"
                          ? "设计目标 · 待验证"
                          : v === "stale"
                            ? "依据变化 · 待复核"
                            : v === "validated"
                              ? "已确认"
                              : "待评审"}
                      </Tag>
                    )),
                    col("接受条件", "acceptance", 220),
                    col("客户确认", "customer_confirmation", 180),
                    col("确认日期", "confirmed_at", 130),
                    col("声明人", "declarant", 140),
                  ]}
                />
              ),
            },
            {
              key: "goals",
              label: "安全目标",
              children: (
                <>
                  <Button
                    icon={<PlusOutlined />}
                    onClick={() => setEditing({ kind: "goal" })}
                  >
                    新增安全目标
                  </Button>
                  <Records
                    rows={ws.security_goals}
                    onEdit={(record) => setEditing({ kind: "goal", record })}
                    columns={[
                      baseCode,
                      col("目标内容", "statement", 480),
                      col("关联威胁", "scenario_ids", 300, joined),
                      col("CAL", "cal", 80),
                      col("控制主题", "theme", 240),
                    ]}
                  />
                </>
              ),
            },
            {
              key: "requirements",
              label: "安全需求",
              children: (
                <Records
                  rows={ws.requirements}
                  onEdit={edit("requirement")}
                  columns={[
                    baseCode,
                    col("所属目标", "goal_ids", 160, joined),
                    col("需求内容", "statement", 420),
                    col("验证方法", "verification", 340),
                    col("专业分配", "allocation", 150),
                    col("CAL", "cal", 80),
                    col("客户需求编号", "customer_refs", 200, joined),
                    col("关联威胁", "scenario_ids", 260, (v, r) =>
                      joined(v || [r.scenario_id]),
                    ),
                  ]}
                />
              ),
            },
            ...(source
              ? [
                  {
                    key: "source",
                    label: `全部原表 (${source.sheets.length})`,
                    children: <OriginalSheets source={source} />,
                  },
                  {
                    key: "checks",
                    label: "导入核对",
                    children: (
                      <>
                        <Descriptions
                          column={1}
                          items={[
                            {
                              key: "hash",
                              label: "来源 SHA-256",
                              children: <Text copyable>{source.sha256}</Text>,
                            },
                            {
                              key: "profile",
                              label: "适配版本",
                              children: source.profile,
                            },
                          ]}
                        />
                        <Alert
                          type="success"
                          title="SFOP、攻击潜力、初始风险与残余目标风险已逐条核对一致"
                          className="section-card"
                        />
                        <Table
                          rowKey={(_, i) => i}
                          size="small"
                          pagination={pagination}
                          dataSource={source.diagnostics || []}
                          columns={Object.keys(
                            source.diagnostics?.[0] || {},
                          ).map((k) =>
                            col(k, k, 180, (v) =>
                              typeof v === "object"
                                ? JSON.stringify(v)
                                : String(v ?? ""),
                            ),
                          )}
                        />
                      </>
                    ),
                  },
                ]
              : []),
          ]}
        />
      </Card>
      {editing && (
        <ProductEditor {...editing} onClose={() => setEditing(null)} />
      )}
    </>
  );
}
