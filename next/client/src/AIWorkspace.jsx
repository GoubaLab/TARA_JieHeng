import React, { useEffect, useMemo, useRef, useState } from "react";
import {
  Alert,
  App,
  Button,
  Card,
  Checkbox,
  Col,
  Descriptions,
  Drawer,
  Empty,
  Form,
  Input,
  Modal,
  Row,
  Select,
  Space,
  Spin,
  Statistic,
  Table,
  Tabs,
  Tag,
  Typography,
  Upload,
} from "antd";
import {
  PlusOutlined,
  ReloadOutlined,
  UploadOutlined,
  PlayCircleOutlined,
} from "@ant-design/icons";
import { PageTitle, useWorkbench } from "./Workbench.jsx";
import { useAuth } from "./AuthGate.jsx";
import { request, date, navigate, properties, treatments } from "./api.js";
import {
  mutateResource,
  purposeLabels,
  taskStates,
  useResource,
} from "./platform-ui.js";
import { staticDiagramSvg } from "./architecture-view.js";
const { Text, Paragraph, Title } = Typography;
const groups = {
  assets: "资产",
  damage_scenarios: "损害场景",
  scenarios: "威胁与攻击路径",
  security_goals: "安全目标",
  requirements: "安全需求",
  treatments: "处置建议",
};
const fieldLabels = {
  id: "建议编号",
  name: "名称",
  title: "场景",
  description: "描述",
  consequence: "后果",
  asset_id: "资产",
  asset_ids: "关联资产",
  property: "安全属性",
  properties: "受损属性",
  damage: "损害",
  damage_ids: "关联损害",
  threat: "威胁",
  stride: "STRIDE",
  cause: "原因",
  cal: "CAL",
  impact: "影响建议",
  paths: "攻击路径",
  statement: "内容",
  scenario_id: "威胁",
  scenario_ids: "关联威胁",
  goal_ids: "关联目标",
  theme: "主题",
  allocation: "分配",
  verification: "验证方法",
  standard_refs: "法规引用",
  treatment: "处置方式",
  rationale: "依据",
  goal: "目标",
  claim: "声明",
  controls: "控制措施",
  residual_impact: "残余影响目标",
  residual_feasibility: "残余可行性目标",
  residual_rationale: "残余依据",
  node_ids: "图形关联",
  asset_type: "类型",
  group: "分组",
  function_refs: "功能",
  steps: "步骤",
  feasibility: "可行性建议",
  actor: "攻击者",
  vector: "向量",
  safety: "S",
  financial: "F",
  operational: "O",
  privacy: "P",
};
function ReadableValue({ value }) {
  const displayValue = (v) => properties[v] || treatments[v] || v;
  if (value == null) return <Text type="secondary">待确认</Text>;
  if (Array.isArray(value)) {
    if (!value.length) return <Text type="secondary">—</Text>;
    if (value.some((v) => v && typeof v === "object"))
      return (
        <Space orientation="vertical">
          {value.map((v, i) => (
            <div key={i}>
              <ReadableValue value={v} />
            </div>
          ))}
        </Space>
      );
    return value.map(displayValue).join("、");
  }
  if (typeof value === "object")
    return (
      <Descriptions
        size="small"
        column={1}
        items={Object.entries(value).map(([key, v]) => ({
          key,
          label: fieldLabels[key] || key,
          children: <ReadableValue value={v} />,
        }))}
      />
    );
  return (
    <span style={{ whiteSpace: "pre-wrap", overflowWrap: "anywhere" }}>
      {String(displayValue(value))}
    </span>
  );
}
function GraphPreview({ graph }) {
  const rendered = useMemo(() => {
    try {
      return { svg: staticDiagramSvg(graph, []) };
    } catch (e) {
      return { error: e.message };
    }
  }, [graph]);
  if (rendered.error)
    return (
      <Alert
        type="warning"
        title="连接关系已保留，布局仍需整理"
        description={rendered.error}
      />
    );
  return (
    <img
      className="ai-diagram-preview"
      alt="AI 生成的原生 DFD 草案预览"
      src={`data:image/svg+xml;charset=utf-8,${encodeURIComponent(rendered.svg)}`}
    />
  );
}
function ProposalDetails({ task }) {
  const p = task.proposal;
  return (
    <Space orientation="vertical" size={16} style={{ width: "100%" }}>
      <Descriptions
        column={2}
        items={[
          {
            key: "status",
            label: "状态",
            children: taskStates[task.status]?.[0],
          },
          {
            key: "revision",
            label: "分析依据",
            children: `项目修订 ${task.base_revision}`,
          },
          {
            key: "prompt",
            label: "提示词",
            children: task.prompt
              ? `${task.prompt.name} · v${task.prompt.revision}`
              : "—",
          },
          {
            key: "model",
            label: "实际模型",
            children: task.usage?.model || "—",
          },
          { key: "time", label: "创建时间", children: date(task.created_at) },
          {
            key: "usage",
            label: "模型用量",
            children: `输入 ${task.usage?.input_tokens || 0} / 输出 ${task.usage?.output_tokens || 0} Token`,
          },
        ]}
      />
      {task.error && <Alert type="error" showIcon title={task.error} />}{" "}
      {task.validation_error && (
        <Alert
          type="warning"
          showIcon
          title="草案未通过平台校验"
          description={task.validation_error}
        />
      )}
      {task.status === "running" && (
        <Space>
          <Spin />
          <Text>{task.step}</Text>
        </Space>
      )}
      {task.index_result && (
        <Alert
          type="success"
          title={`索引完成：${task.index_result.chunks} 个片段 · ${task.index_result.dimensions} 维 · ${task.index_result.model}`}
        />
      )}
      {p && (
        <>
          <Card size="small" title="分析摘要">
            <Paragraph style={{ whiteSpace: "pre-wrap" }}>
              {p.summary}
            </Paragraph>
            {p.assumptions?.length > 0 && (
              <>
                <Text strong>使用的假设</Text>
                <ul>
                  {p.assumptions.map((v, i) => (
                    <li key={i}>{v}</li>
                  ))}
                </ul>
              </>
            )}
            {p.questions?.length > 0 && (
              <Alert
                type="warning"
                title="需要工作人员核对"
                description={
                  <ol>
                    {p.questions.map((v, i) => (
                      <li key={i}>{v}</li>
                    ))}
                  </ol>
                }
              />
            )}
          </Card>
          {task.preview_architecture && (
            <Card
              title={`图形草案 · ${task.counts?.nodes || 0} 个节点 / ${task.counts?.flows || 0} 条连接`}
            >
              <GraphPreview graph={task.preview_architecture} />
              <Text type="secondary">
                采用后成为平台原生图形，可以继续拖动节点、调整连线、标注协议并导出静态图。保存时仍会遵循项目的资产生成设置。
              </Text>
            </Card>
          )}
          <Tabs
            items={Object.entries(groups)
              .filter(([key]) => p[key]?.length)
              .map(([key, label]) => ({
                key,
                label: `${label} (${p[key].length})`,
                children: (
                  <Space orientation="vertical" style={{ width: "100%" }}>
                    {p[key].map((row, i) => (
                      <Card
                        size="small"
                        key={row.id || i}
                        title={
                          row.title || row.name || row.id || `${label} ${i + 1}`
                        }
                      >
                        <ReadableValue value={row} />
                      </Card>
                    ))}
                  </Space>
                ),
              }))}
          />
        </>
      )}
      <Card size="small" title="引用与检索依据">
        <Space orientation="vertical" style={{ width: "100%" }}>
          <Tag>
            {task.retrieval?.mode === "hybrid"
              ? "语义与关键词混合检索"
              : "项目内容与关键词检索"}
          </Tag>
          {task.retrieval?.warnings?.map((v, i) => (
            <Alert key={i} type="warning" title={v} />
          ))}
          {task.sources?.map((source) => (
            <div key={source.id} className="ai-source">
              <Space wrap>
                <Text strong>{source.name}</Text>
                <Tag>{source.edition}</Tag>
                {p?.citations?.includes(source.id) && (
                  <Tag color="blue">已引用</Tag>
                )}
              </Space>
              <Paragraph type="secondary">
                {source.source}
                {source.chunk ? ` · 片段 ${source.chunk}` : ""}
              </Paragraph>
              {source.text && (
                <Paragraph
                  ellipsis={{
                    rows: 4,
                    expandable: true,
                    symbol: "展开引用原文",
                  }}
                >
                  {source.text}
                </Paragraph>
              )}
            </div>
          ))}
        </Space>
      </Card>
      {task.attempts?.length > 0 && (
        <Card size="small" title="调用记录">
          <Table
            size="small"
            pagination={false}
            rowKey={(_, i) => i}
            dataSource={task.attempts}
            columns={[
              {
                title: "服务",
                dataIndex: "provider_id",
                render: (v) =>
                  task.providers?.find((p) => p.id === v)?.name || v,
              },
              { title: "模型", dataIndex: "model" },
              {
                title: "结果",
                dataIndex: "status",
                render: (v) =>
                  ({
                    success: "成功",
                    failed: "失败",
                    circuit_open: "熔断跳过",
                  })[v] || v,
              },
              {
                title: "用时",
                dataIndex: "latency_ms",
                render: (v) => (v != null ? `${v} ms` : "—"),
              },
              { title: "说明", dataIndex: "error" },
            ]}
          />
        </Card>
      )}
    </Space>
  );
}
export default function AIWorkspace() {
  const { ws, reload } = useWorkbench(),
    { user } = useAuth(),
    { message } = App.useApp();
  const providers = useResource("/model-providers"),
    prompts = useResource("/prompt-templates"),
    knowledge = useResource(`/knowledge-documents?workspace_id=${ws.id}`),
    tasks = useResource(`/workspaces/${ws.id}/ai-tasks`);
  const [form] = Form.useForm(),
    [creating, setCreating] = useState(false),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [image, setImage] = useState(null),
    [selected, setSelected] = useState(null),
    [task, setTask] = useState(null),
    [owner, setOwner] = useState(""),
    [reviewed, setReviewed] = useState(false),
    [feedback, setFeedback] = useState("");
  const generation = useRef(0),
    requestMemo = useRef(null);
  const kind = Form.useWatch("kind", form) || "analysis",
    chain = Form.useWatch("provider_ids", form) || [],
    embedding = Form.useWatch("embedding_provider_id", form);
  const canRun = (ws.access_role || user.role) !== "viewer";
  const available = (providers.data?.items || []).filter((p) => p.enabled),
    external = available.filter(
      (p) =>
        (chain.includes(p.id) || p.id === embedding) &&
        p.network_scope === "external",
    );
  useEffect(() => {
    generation.current++;
    setCreating(false);
    setSelected(null);
    setTask(null);
    setError("");
    setImage(null);
  }, [ws.id]);
  useEffect(() => {
    if (!selected) {
      setTask(null);
      return;
    }
    let active = true,
      interval;
    setTask(null);
    const seq = ++generation.current;
    const load = async () => {
      try {
        const out = await request(`/workspaces/${ws.id}/ai-tasks/${selected}`);
        if (active && seq === generation.current) {
          setTask(out);
          if (!["running", "queued"].includes(out.status))
            clearInterval(interval);
        }
      } catch (e) {
        if (active) setError(e.message);
      }
    };
    interval = setInterval(() => {
      if (document.visibilityState === "visible") load();
    }, 2500);
    load();
    return () => {
      active = false;
      clearInterval(interval);
    };
  }, [selected, ws.id]);
  useEffect(() => {
    if (
      !tasks.data?.items?.some((t) => ["running", "queued"].includes(t.status))
    )
      return;
    const interval = setInterval(tasks.refresh, 3500);
    return () => clearInterval(interval);
  }, [tasks.data?.items, tasks.refresh]);
  const open = () => {
    setCreating(true);
    setImage(null);
    setError("");
    form.resetFields();
    form.setFieldsValue({
      kind: "analysis",
      provider_ids: available
        .filter((p) => p.models.analysis && p.network_scope === "internal")
        .slice(0, 1)
        .map((p) => p.id),
      prompt_id: "builtin-analysis",
      owner: ws.item.owner || user.name,
      document_ids: [],
      allow_external: false,
    });
    requestMemo.current = null;
  };
  const upload = async (file) => {
    try {
      if (file.size > 1000000) throw Error("图片不能超过 1 MB，请先裁剪或压缩");
      if (!["image/png", "image/jpeg"].includes(file.type))
        throw Error("请选择 PNG 或 JPEG 图片");
      const data = await new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(reader.result.split(",")[1]);
        reader.onerror = () => reject(Error("图片读取失败"));
        reader.readAsDataURL(file);
      });
      setImage({ name: file.name, data, mime: file.type });
      setError("");
    } catch (e) {
      setError(e.message);
    }
    return false;
  };
  const submit = async () => {
    try {
      const values = await form.validateFields();
      if (kind === "diagram" && !image) throw Error("请先选择一张参考图");
      setBusy(true);
      setError("");
      const payload = {
        ...values,
        embedding_provider_id: values.embedding_provider_id || "",
        instructions: values.instructions || "",
        ...(kind === "diagram"
          ? { image_base64: image.data, image_mime: image.mime }
          : {}),
      };
      const fp = JSON.stringify(payload);
      if (requestMemo.current?.fp !== fp)
        requestMemo.current = { fp, key: crypto.randomUUID() };
      const job = await mutateResource(
        `/workspaces/${ws.id}/ai-tasks`,
        payload,
        ws.revision,
        "POST",
        requestMemo.current.key,
      );
      setCreating(false);
      setSelected(job.id);
      setTask(job);
      setOwner(values.owner || user.name);
      setReviewed(false);
      setFeedback("");
      tasks.refresh();
      message.success("任务已创建，可离开页面后继续查看");
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  const action = async (type) => {
    if (!task) return;
    setBusy(true);
    setError("");
    try {
      let out;
      if (type === "adoptions") {
        out = await mutateResource(
          `/workspaces/${ws.id}/ai-tasks/${task.id}/adoptions`,
          { owner, reviewed },
          ws.revision,
        );
        await reload({ quiet: true });
        message.success("草案已采用，原项目已建立还原点");
      } else if (type === "retries") {
        out = await mutateResource(
          `/workspaces/${ws.id}/ai-tasks/${task.id}/retries`,
          { instructions: feedback || task.request.instructions },
          ws.revision,
        );
        setSelected(out.id);
        setTask(out);
        setReviewed(false);
        message.success("已基于当前修订创建新任务");
      } else {
        out = await mutateResource(
          `/workspaces/${ws.id}/ai-tasks/${task.id}/cancellations`,
          {},
        );
        setTask(out);
      }
      tasks.refresh();
      if (type === "adoptions")
        setTask(await request(`/workspaces/${ws.id}/ai-tasks/${task.id}`));
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <>
      <PageTitle
        title="辅助建模与分析"
        description="把图片、项目上下文和企业知识转为可审阅的工作产品草案；平台负责校验、风险计算和追溯。"
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={tasks.refresh}>
              刷新任务
            </Button>
            {canRun && (
              <Button type="primary" icon={<PlusOutlined />} onClick={open}>
                新建分析任务
              </Button>
            )}
          </Space>
        }
      />
      {(error || tasks.error || providers.error) && (
        <Alert
          className="section-card"
          type="error"
          showIcon
          title={error || tasks.error || providers.error}
          closable
          onClose={() => setError("")}
        />
      )}
      {!available.length && (
        <Alert
          className="section-card"
          type="info"
          title="尚未配置可用模型服务"
          description="管理员可在“模型服务”中接入内网模型或允许使用的外部 API，配置分析、视觉与向量模型。"
          action={
            <Button onClick={() => navigate("models", ws.id)}>前往配置</Button>
          }
        />
      )}
      <Row gutter={16} className="section-card">
        {[
          ["图片建模", "上传结构图，生成可继续编辑的 DFD。"],
          ["全流程分析", "补齐资产、损害、威胁、目标、需求和处置草案。"],
          ["质量复核", "结合现有分析与知识，发现遗漏和矛盾。"],
        ].map(([title, description]) => (
          <Col xs={24} lg={8} key={title}>
            <Card size="small" title={title}>
              <Paragraph type="secondary">{description}</Paragraph>
            </Card>
          </Col>
        ))}
      </Row>
      <Card>
        <Table
          rowKey="id"
          loading={tasks.loading}
          dataSource={tasks.data?.items || []}
          locale={{
            emptyText: (
              <Empty description="尚无辅助任务。已有项目可直接复核，也可以从一张图开始建模。" />
            ),
          }}
          columns={[
            {
              title: "任务",
              dataIndex: "kind",
              render: (v) => purposeLabels[v],
            },
            {
              title: "状态",
              dataIndex: "status",
              render: (v) => (
                <Tag color={taskStates[v]?.[1]}>{taskStates[v]?.[0] || v}</Tag>
              ),
            },
            { title: "当前阶段", dataIndex: "step" },
            {
              title: "项目修订",
              dataIndex: "base_revision",
              render: (v) => `修订 ${v}`,
            },
            { title: "发起时间", dataIndex: "created_at", render: date },
            {
              title: "操作",
              render: (_, t) => (
                <Button
                  type="link"
                  onClick={() => {
                    setSelected(t.id);
                    setOwner(t.request?.owner || ws.item.owner || user.name);
                    setReviewed(false);
                    setFeedback("");
                  }}
                >
                  查看结果
                </Button>
              ),
            },
          ]}
          pagination={{ pageSize: 12 }}
        />
      </Card>
      <Modal
        open={creating}
        title="新建辅助分析任务"
        width={820}
        onCancel={() => setCreating(false)}
        footer={
          <Space>
            <Button onClick={() => setCreating(false)}>取消</Button>
            <Button
              type="primary"
              icon={<PlayCircleOutlined />}
              loading={busy}
              onClick={submit}
            >
              生成草案
            </Button>
          </Space>
        }
        destroyOnHidden
      >
        {error && <Alert type="error" title={error} />}
        <Form form={form} layout="vertical">
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="kind"
                label="任务类型"
                rules={[{ required: true }]}
              >
                <Select
                  options={["diagram", "analysis", "review"].map((value) => ({
                    value,
                    label: purposeLabels[value],
                  }))}
                  onChange={(value) => {
                    form.setFieldsValue({
                      prompt_id: `builtin-${value}`,
                      provider_ids: [],
                    });
                    setImage(null);
                  }}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="owner"
                label="草案责任人"
                rules={[{ required: true }]}
              >
                <Input maxLength={200} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item
            name="provider_ids"
            label="模型服务顺序（首选 → 备用，最多 4 个）"
            rules={[{ required: true }]}
          >
            <Select
              mode="multiple"
              maxCount={4}
              placeholder="请选择需要的模型服务"
              options={available
                .filter(
                  (p) => p.models[kind === "diagram" ? "vision" : "analysis"],
                )
                .map((p) => ({
                  value: p.id,
                  label: `${p.name} · ${p.network_scope === "internal" ? "内网" : "外部"}`,
                }))}
            />
          </Form.Item>
          <Form.Item
            name="prompt_id"
            label="分析指引"
            rules={[{ required: true }]}
          >
            <Select
              options={(prompts.data?.items || [])
                .filter((p) => p.enabled && p.purpose === kind)
                .map((p) => ({
                  value: p.id,
                  label: `${p.name} · v${p.revision}`,
                }))}
            />
          </Form.Item>
          {kind === "diagram" && (
            <Form.Item label="参考图片" required>
              <Upload
                accept="image/png,image/jpeg"
                beforeUpload={upload}
                showUploadList={false}
              >
                <Button icon={<UploadOutlined />}>选择 PNG / JPEG</Button>
              </Upload>
              {image && (
                <>
                  <Tag color="blue">{image.name}</Tag>
                  <img
                    alt="待识别参考图"
                    style={{
                      maxWidth: "100%",
                      maxHeight: 240,
                      display: "block",
                      marginTop: 12,
                    }}
                    src={`data:${image.mime};base64,${image.data}`}
                  />
                </>
              )}
              <Text type="secondary">
                最大 1 MB；图中文字和方向不清晰时，会列入待核对项。
              </Text>
            </Form.Item>
          )}
          <Form.Item name="instructions" label="本次重点与补充要求">
            <Input.TextArea
              rows={4}
              maxLength={12000}
              placeholder="例如：重点检查升级链路、调试接口和密钥存储；保留已有分析，只补充缺项。"
            />
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="document_ids" label="知识范围">
                <Select
                  mode="multiple"
                  maxCount={100}
                  allowClear
                  placeholder="默认检索组织与本项目启用资料"
                  options={(knowledge.data?.items || [])
                    .filter((d) => d.enabled)
                    .map((d) => ({
                      value: d.id,
                      label: `${d.name} · ${d.edition}`,
                    }))}
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="embedding_provider_id" label="检索方式">
                <Select
                  allowClear
                  placeholder="关键词检索（不调用向量模型）"
                  options={available
                    .filter((p) => p.models.embedding)
                    .map((p) => ({
                      value: p.id,
                      label: `语义检索 · ${p.name}`,
                    }))}
                />
              </Form.Item>
            </Col>
          </Row>
          <Alert
            type="info"
            title={`任务依据当前修订 ${ws.revision}`}
            description="发送范围包括本次指令、项目工作产品和匹配的知识片段；图片建模还会发送所选图片。不发送账号密钥、原始工作簿文件或验证附件。结果不会自动覆盖项目，也不会生成通过记录或签署结论。"
          />
          {external.length > 0 && (
            <Alert
              type="warning"
              style={{ marginTop: 12 }}
              title={`外部目的地：${external.map((p) => p.name).join("、")}`}
              description={external.map((p) => p.base_url).join("；")}
            />
          )}
          <Form.Item
            name="allow_external"
            hidden={!external.length}
            valuePropName="checked"
          >
            <Checkbox>允许本次任务向上述所选外部服务发送已说明的数据</Checkbox>
          </Form.Item>
        </Form>
      </Modal>
      <Drawer
        open={!!selected}
        title="任务结果与采用"
        width="min(1200px, 95vw)"
        onClose={() => setSelected(null)}
        extra={
          task &&
          canRun &&
          ["queued", "running"].includes(task.status) && (
            <Button
              danger
              loading={busy}
              onClick={() => action("cancellations")}
            >
              取消任务
            </Button>
          )
        }
      >
        {error && <Alert type="error" title={error} showIcon />}
        {!task ? (
          <Spin />
        ) : (
          <>
            <ProposalDetails task={task} />
            {task.status === "ready" && task.kind !== "review" && canRun && (
              <Card className="section-card" title="采用草案">
                <Space orientation="vertical" style={{ width: "100%" }}>
                  {task.base_revision !== ws.revision && (
                    <Alert
                      type="warning"
                      title="项目已有更新，需要基于当前修订重新生成，才能采用。"
                    />
                  )}
                  <Alert
                    type="info"
                    title="采用前自动保存还原点，整批写入在同一事务内完成。处置建议仍需人工确认；残余风险目标需验证。"
                  />
                  <Input
                    value={owner}
                    onChange={(e) => setOwner(e.target.value)}
                    placeholder="采用建议的责任人"
                    maxLength={200}
                  />
                  <Checkbox
                    checked={reviewed}
                    onChange={(e) => setReviewed(e.target.checked)}
                  >
                    我已审阅草案、引用依据和待确认项，同意将其写入当前项目
                  </Checkbox>
                  <Button
                    type="primary"
                    loading={busy}
                    disabled={
                      !reviewed ||
                      !owner.trim() ||
                      task.base_revision !== ws.revision
                    }
                    onClick={() => action("adoptions")}
                  >
                    采用并保存工作产品
                  </Button>
                </Space>
              </Card>
            )}
            {task.status === "adopted" && (
              <Alert
                type="success"
                title={`已采用到修订 ${task.adopted_revision}`}
                action={
                  <Button
                    onClick={() =>
                      navigate(
                        task.kind === "diagram" ? "architecture" : "tara",
                        ws.id,
                      )
                    }
                  >
                    打开工作产品
                  </Button>
                }
              />
            )}
            {canRun && !["queued", "running"].includes(task.status) && (
              <Card title="继续完善" style={{ marginTop: 16 }}>
                <Space orientation="vertical" style={{ width: "100%" }}>
                  <Input.TextArea
                    value={feedback}
                    onChange={(e) => setFeedback(e.target.value)}
                    rows={3}
                    placeholder="补充要求或修正意见；新任务将使用当前项目修订，保留原任务结果。"
                  />
                  <Button loading={busy} onClick={() => action("retries")}>
                    基于当前项目重新生成
                  </Button>
                </Space>
              </Card>
            )}
          </>
        )}
      </Drawer>
    </>
  );
}
