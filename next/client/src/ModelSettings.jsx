import React, { useEffect, useRef, useState } from "react";
import {
  Alert,
  App,
  AutoComplete,
  Button,
  Card,
  Checkbox,
  Col,
  Descriptions,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  Typography,
  Upload,
} from "antd";
import {
  PlusOutlined,
  CopyOutlined,
  DownloadOutlined,
  UploadOutlined,
  ApiOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
import { PageTitle } from "./Workbench.jsx";
import { useAuth } from "./AuthGate.jsx";
import { request, date } from "./api.js";
import {
  downloadJSON,
  mutateResource,
  pickFields,
  providerProtocols,
  purposeLabels,
  useResource,
} from "./platform-ui.js";
const { Text, Paragraph } = Typography;
const providerFields = [
  "name",
  "protocol",
  "base_url",
  "network_scope",
  "models",
  "enabled",
  "priority",
  "timeout_seconds",
  "max_output_tokens",
  "retries",
  "notes",
  "api_key",
  "clear_key",
  "input_price_per_million",
  "output_price_per_million",
  "currency",
  "generation_path",
  "embedding_path",
  "models_path",
  "chat_token_parameter",
];
const presets = {
  ollama: {
    name: "企业 Ollama",
    protocol: "ollama",
    base_url: "http://127.0.0.1:11434",
    network_scope: "internal",
  },
  gateway: {
    name: "企业模型网关",
    protocol: "openai_chat",
    base_url: "",
    network_scope: "internal",
  },
  openai: {
    name: "OpenAI API",
    protocol: "openai_responses",
    base_url: "https://api.openai.com/v1",
    network_scope: "external",
  },
  anthropic: {
    name: "Anthropic API",
    protocol: "anthropic",
    base_url: "https://api.anthropic.com/v1",
    network_scope: "external",
  },
};
function ProviderEditor({ record, onClose, onSaved }) {
  const [form] = Form.useForm(),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [models, setModels] = useState([]),
    key = useRef(crypto.randomUUID());
  const { message } = App.useApp();
  useEffect(() => {
    form.resetFields();
    form.setFieldsValue(
      record || {
        ...presets.gateway,
        models: {},
        enabled: true,
        priority: 10,
        timeout_seconds: 90,
        max_output_tokens: 8192,
        chat_token_parameter: "max_completion_tokens",
        retries: 1,
        currency: "USD",
        input_price_per_million: 0,
        output_price_per_million: 0,
      },
    );
  }, [record]);
  const submit = async (test = false) => {
    try {
      const values = await form.validateFields();
      setBusy(true);
      setError("");
      const saved = await mutateResource(
        record?.id ? `/model-providers/${record.id}` : "/model-providers",
        pickFields(values, providerFields),
        record?.revision,
        record?.id ? "PUT" : "POST",
        key.current,
      );
      if (test) {
        try {
          const role = values.models.analysis
            ? "analysis"
            : values.models.vision
              ? "vision"
              : "embedding";
          await mutateResource(
            `/model-providers/${saved.id}/connection-tests`,
            { role },
          );
          message.success("配置已保存，连接测试通过");
        } catch (e) {
          message.warning("配置已保存；连接测试未通过：" + e.message);
        }
      } else message.success("模型配置已保存");
      onSaved();
      onClose();
    } catch (e) {
      setError(e.message || "请检查配置项");
    } finally {
      setBusy(false);
    }
  };
  const discover = async () => {
    setBusy(true);
    try {
      const out = await mutateResource(
        `/model-providers/${record.id}/model-discoveries`,
        {},
      );
      setModels(out.models);
      message.success(`获取 ${out.models.length} 个模型`);
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal
      open
      title={record?.id ? "编辑模型服务" : "添加模型服务"}
      width={780}
      onCancel={onClose}
      footer={
        <Space>
          <Button onClick={onClose}>取消</Button>
          <Button loading={busy} onClick={() => submit(false)}>
            保存
          </Button>
          <Button type="primary" loading={busy} onClick={() => submit(true)}>
            保存并测试
          </Button>
        </Space>
      }
      destroyOnHidden
    >
      {error && <Alert type="error" title={error} showIcon />}
      <Form form={form} layout="vertical">
        {!record?.id && (
          <Form.Item label="快速配置">
            <Space wrap>
              {Object.entries(presets).map(([key, p]) => (
                <Button
                  key={key}
                  onClick={() =>
                    form.setFieldsValue({
                      ...p,
                      models: {},
                      api_key: undefined,
                    })
                  }
                >
                  {p.name}
                </Button>
              ))}
            </Space>
          </Form.Item>
        )}
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item
              name="name"
              label="服务名称"
              rules={[{ required: true }]}
            >
              <Input maxLength={120} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="protocol"
              label="接口协议"
              rules={[{ required: true }]}
            >
              <Select
                options={Object.entries(providerProtocols).map(
                  ([value, label]) => ({ value, label }),
                )}
              />
            </Form.Item>
          </Col>
        </Row>
        <Form.Item
          name="base_url"
          label="API 根地址"
          extra="OpenAI / Anthropic 通常包含 /v1；Ollama 填写主机地址。路径可在高级设置调整。"
          rules={[{ required: true }]}
        >
          <Input
            placeholder="https://models.company.example/v1"
            autoComplete="off"
          />
        </Form.Item>
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item
              name="network_scope"
              label="数据目的地"
              rules={[{ required: true }]}
            >
              <Select
                options={[
                  { value: "internal", label: "本机／企业内网" },
                  { value: "external", label: "外部 HTTPS 服务" },
                ]}
              />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="api_key"
              label={
                record?.has_key
                  ? "替换凭据（留空保留原凭据）"
                  : "API Key（无认证内网服务可留空）"
              }
            >
              <Input.Password autoComplete="new-password" />
            </Form.Item>
          </Col>
        </Row>
        {record?.has_key && (
          <Form.Item name="clear_key" valuePropName="checked">
            <Checkbox>清除当前保存的凭据</Checkbox>
          </Form.Item>
        )}
        <Alert
          type="info"
          title="凭据加密保存在服务端，配置导出和浏览器读取不会返回密钥。外部服务只有在具体任务明确允许时才能接收项目内容。"
        />
        <Space style={{ marginTop: 16 }}>
          <Text strong>按用途分配模型</Text>
          {record?.id && (
            <Button size="small" loading={busy} onClick={discover}>
              获取模型列表
            </Button>
          )}
        </Space>
        <Row gutter={16}>
          {[
            ["analysis", "分析模型"],
            ["vision", "视觉模型"],
            ["embedding", "向量模型"],
          ].map(([key, label]) => (
            <Col span={8} key={key}>
              <Form.Item name={["models", key]} label={label}>
                <AutoComplete
                  allowClear
                  options={models.map((value) => ({ value, label: value }))}
                  placeholder="输入模型 ID"
                />
              </Form.Item>
            </Col>
          ))}
        </Row>
        <Paragraph type="secondary">
          填写服务商实际提供的模型
          ID；至少配置一种用途。未配置视觉模型时，不能执行图片建模。
        </Paragraph>
        <Tabs
          items={[
            {
              key: "reliability",
              label: "可靠性与限额",
              children: (
                <>
                  <Row gutter={16}>
                    <Col span={6}>
                      <Form.Item
                        name="enabled"
                        label="启用服务"
                        valuePropName="checked"
                      >
                        <Switch />
                      </Form.Item>
                    </Col>
                    <Col span={6}>
                      <Form.Item name="priority" label="显示优先级">
                        <InputNumber min={0} max={1000} />
                      </Form.Item>
                    </Col>
                    <Col span={6}>
                      <Form.Item name="timeout_seconds" label="超时（秒）">
                        <InputNumber min={5} max={180} />
                      </Form.Item>
                    </Col>
                    <Col span={6}>
                      <Form.Item name="retries" label="瞬时故障重试">
                        <InputNumber min={0} max={2} />
                      </Form.Item>
                    </Col>
                  </Row>
                  <Form.Item
                    name="chat_token_parameter"
                    label="Chat 接口输出上限参数"
                  >
                    <Select
                      options={[
                        {
                          value: "max_completion_tokens",
                          label: "max_completion_tokens（新版）",
                        },
                        {
                          value: "max_tokens",
                          label: "max_tokens（兼容网关）",
                        },
                      ]}
                    />
                  </Form.Item>
                  <Form.Item name="max_output_tokens" label="单次输出上限">
                    <InputNumber min={256} max={32768} />
                  </Form.Item>
                  <Paragraph type="secondary">
                    限流、服务不可用和网络故障可有限重试；凭据错误直接停止。连续失败会进入短暂熔断。备用服务由每次任务明确选择，不会自动转发到未选服务。
                  </Paragraph>
                </>
              ),
            },
            {
              key: "usage",
              label: "用量估算",
              children: (
                <Row gutter={16}>
                  <Col span={8}>
                    <Form.Item name="currency" label="币种">
                      <Select
                        options={[
                          { value: "USD", label: "USD" },
                          { value: "CNY", label: "CNY" },
                        ]}
                      />
                    </Form.Item>
                  </Col>
                  <Col span={8}>
                    <Form.Item
                      name="input_price_per_million"
                      label="每百万输入 Token 单价"
                    >
                      <InputNumber min={0} />
                    </Form.Item>
                  </Col>
                  <Col span={8}>
                    <Form.Item
                      name="output_price_per_million"
                      label="每百万输出 Token 单价"
                    >
                      <InputNumber min={0} />
                    </Form.Item>
                  </Col>
                  <Col span={24}>
                    <Text type="secondary">
                      单价由管理员维护；页面按当前单价估算，不代表供应商账单。
                    </Text>
                  </Col>
                </Row>
              ),
            },
            {
              key: "advanced",
              label: "高级路径",
              children: (
                <>
                  {[
                    ["generation_path", "生成接口"],
                    ["models_path", "模型列表接口"],
                    ["embedding_path", "向量接口"],
                  ].map(([key, label]) => (
                    <Form.Item key={key} name={key} label={label}>
                      <Input placeholder="留空使用协议默认相对路径" />
                    </Form.Item>
                  ))}
                  <Form.Item name="notes" label="备注">
                    <Input.TextArea rows={3} />
                  </Form.Item>
                </>
              ),
            },
          ]}
        />
      </Form>
    </Modal>
  );
}
function Models() {
  const resource = useResource("/model-providers"),
    { user } = useAuth(),
    { message } = App.useApp();
  const [editing, setEditing] = useState(null),
    [testing, setTesting] = useState(""),
    [importing, setImporting] = useState(null),
    [error, setError] = useState("");
  const canManage = user.role === "admin";
  const test = async (p) => {
    setTesting(p.id);
    try {
      const role = p.models.analysis
        ? "analysis"
        : p.models.vision
          ? "vision"
          : "embedding";
      const out = await mutateResource(
        `/model-providers/${p.id}/connection-tests`,
        { role },
      );
      message.success(`连接成功 · ${out.latency_ms} ms`);
    } catch (e) {
      message.error(e.message);
    } finally {
      setTesting("");
      resource.refresh();
    }
  };
  const exportConfig = async () => {
    try {
      downloadJSON(
        await request("/model-providers/export"),
        "界衡-模型服务配置.json",
      );
    } catch (e) {
      setError(e.message);
    }
  };
  const prepareImport = async (file) => {
    try {
      if (file.size > 200000) throw Error("配置文件不能超过 200 KB");
      const data = JSON.parse(await file.text());
      if (
        data.schema !== "jieheng.providers/1" ||
        !Array.isArray(data.providers)
      )
        throw Error("请选择界衡导出的配置文件");
      setImporting(data);
    } catch (e) {
      setError(e.message);
    }
    return false;
  };
  return (
    <Space orientation="vertical" size={16} style={{ width: "100%" }}>
      <Space wrap>
        <Button icon={<ReloadOutlined />} onClick={resource.refresh}>
          刷新状态
        </Button>
        {canManage && (
          <>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setEditing({})}
            >
              添加服务
            </Button>
            <Button icon={<DownloadOutlined />} onClick={exportConfig}>
              导出配置
            </Button>
            <Upload
              accept=".json"
              showUploadList={false}
              beforeUpload={prepareImport}
            >
              <Button icon={<UploadOutlined />}>导入配置</Button>
            </Upload>
          </>
        )}
      </Space>
      {(resource.error || error) && (
        <Alert type="error" title={resource.error || error} showIcon />
      )}
      {!resource.loading && !resource.data?.items.length && (
        <Empty description="配置企业模型服务后，即可使用图片建模、分析草案和语义检索。平台不会自动连接任何模型服务。" />
      )}
      <Row gutter={[16, 16]}>
        {resource.data?.items.map((p) => (
          <Col xs={24} xl={12} key={p.id}>
            <Card
              title={
                <Space>
                  <ApiOutlined />
                  {p.name}
                  <Tag color={p.enabled ? "blue" : "default"}>
                    {p.enabled ? "已启用" : "已停用"}
                  </Tag>
                </Space>
              }
              extra={
                canManage && (
                  <Button type="link" onClick={() => setEditing(p)}>
                    编辑
                  </Button>
                )
              }
            >
              <Descriptions
                size="small"
                column={1}
                items={[
                  {
                    key: "endpoint",
                    label: "地址",
                    children: (
                      <Text style={{ wordBreak: "break-all" }}>
                        {p.base_url}
                      </Text>
                    ),
                  },
                  {
                    key: "protocol",
                    label: "协议",
                    children: providerProtocols[p.protocol],
                  },
                  {
                    key: "scope",
                    label: "范围",
                    children:
                      p.network_scope === "internal"
                        ? "企业内网"
                        : "外部 HTTPS",
                  },
                  {
                    key: "models",
                    label: "模型",
                    children: (
                      <Space wrap>
                        {Object.entries(p.models)
                          .filter(([, v]) => v)
                          .map(([role, id]) => (
                            <Tag key={role}>
                              {
                                {
                                  analysis: "分析",
                                  vision: "视觉",
                                  embedding: "向量",
                                }[role]
                              }{" "}
                              · {id}
                            </Tag>
                          ))}
                      </Space>
                    ),
                  },
                  {
                    key: "health",
                    label: "健康",
                    children: (
                      <Space>
                        <Tag
                          color={
                            p.health.state === "healthy"
                              ? "success"
                              : p.health.state === "untested"
                                ? "default"
                                : "warning"
                          }
                        >
                          {
                            {
                              healthy: "正常",
                              untested: "未测试",
                              degraded: "异常待恢复",
                              open: "熔断冷却",
                            }[p.health.state]
                          }
                        </Tag>
                        {p.health.latency_ms != null &&
                          `${p.health.latency_ms} ms`}
                      </Space>
                    ),
                  },
                  {
                    key: "usage",
                    label: "用量",
                    children: `${p.health.requests || 0} 次请求 · 输入 ${p.health.input_tokens || 0} / 输出 ${p.health.output_tokens || 0} Token`,
                  },
                  {
                    key: "cost",
                    label: "估算",
                    children: `${p.health.currency || p.currency} ${Number(p.health.estimated_cost || 0).toFixed(4)}`,
                  },
                ]}
              />
              {p.health.last_error && (
                <Alert type="warning" title={p.health.last_error} />
              )}
              <Space style={{ marginTop: 12 }}>
                {canManage && (
                  <>
                    <Button loading={testing === p.id} onClick={() => test(p)}>
                      测试连接
                    </Button>
                    <Button
                      icon={<CopyOutlined />}
                      onClick={() =>
                        setEditing({
                          ...p,
                          id: undefined,
                          revision: undefined,
                          has_key: false,
                          name: p.name + " · 副本",
                        })
                      }
                    >
                      复制配置
                    </Button>
                  </>
                )}
                <Text type="secondary">
                  凭据：{p.has_key ? "已配置" : "未配置"}
                </Text>
              </Space>
            </Card>
          </Col>
        ))}
      </Row>
      {editing && (
        <ProviderEditor
          record={Object.keys(editing).length ? editing : null}
          onClose={() => setEditing(null)}
          onSaved={resource.refresh}
        />
      )}
      <Modal
        open={!!importing}
        title="导入模型服务配置"
        onCancel={() => setImporting(null)}
        onOk={async () => {
          try {
            await mutateResource("/model-provider-imports", importing);
            setImporting(null);
            resource.refresh();
            message.success("配置已导入，请补充凭据并测试");
          } catch (e) {
            setError(e.message);
          }
        }}
        okText="确认导入"
      >
        <Paragraph>
          将添加 {importing?.providers.length || 0}{" "}
          项配置，不覆盖已有服务。导入的配置默认停用且不含凭据。
        </Paragraph>
        {importing?.providers.map((p, i) => (
          <Tag key={i}>{p.name}</Tag>
        ))}
      </Modal>
    </Space>
  );
}
function Prompts() {
  const resource = useResource("/prompt-templates"),
    { user } = useAuth(),
    { message } = App.useApp();
  const [editing, setEditing] = useState(null),
    [form] = Form.useForm(),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const open = (p) => {
    setEditing(p);
    form.resetFields();
    form.setFieldsValue(p);
    setError("");
  };
  const save = async () => {
    try {
      const values = await form.validateFields();
      setBusy(true);
      const fields = pickFields(values, [
        "name",
        "purpose",
        "instructions",
        "enabled",
      ]);
      await mutateResource(
        editing.id ? `/prompt-templates/${editing.id}` : "/prompt-templates",
        fields,
        editing.revision,
        editing.id ? "PUT" : "POST",
      );
      setEditing(null);
      resource.refresh();
      message.success("提示词版本已保存；已执行任务保留原版本");
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <>
      <Alert
        type="info"
        title="组织提示词可补充领域经验、分析重点和输出要求。平台的引用校验、人工审阅、禁止伪造验证等规则始终生效。"
        style={{ marginBottom: 16 }}
      />
      {resource.error && <Alert type="error" title={resource.error} />}
      <Table
        rowKey="id"
        dataSource={resource.data?.items || []}
        loading={resource.loading}
        columns={[
          { title: "模板", dataIndex: "name" },
          {
            title: "用途",
            dataIndex: "purpose",
            render: (v) => purposeLabels[v],
          },
          { title: "版本", dataIndex: "revision", render: (v) => `v${v}` },
          {
            title: "类型",
            dataIndex: "builtin",
            render: (v) => (v ? "平台内置" : "组织模板"),
          },
          {
            title: "状态",
            dataIndex: "enabled",
            render: (v) => (v ? "启用" : "停用"),
          },
          {
            title: "操作",
            render: (_, p) => (
              <Space>
                <Button
                  type="link"
                  onClick={() =>
                    open(
                      user.role === "admin"
                        ? p.builtin
                          ? {
                              ...p,
                              id: undefined,
                              revision: undefined,
                              name: p.name + " · 组织版",
                            }
                          : p
                        : { ...p, readOnly: true },
                    )
                  }
                >
                  {user.role === "admin"
                    ? p.builtin
                      ? "创建组织版本"
                      : "编辑新版本"
                    : "查看"}
                </Button>
              </Space>
            ),
          },
        ]}
        pagination={false}
      />
      <Modal
        open={!!editing}
        title={editing?.readOnly ? "提示词内容" : "组织提示词"}
        width={760}
        onCancel={() => setEditing(null)}
        onOk={save}
        confirmLoading={busy}
        footer={editing?.readOnly ? null : undefined}
      >
        {error && <Alert type="error" title={error} />}
        <Form form={form} layout="vertical" disabled={editing?.readOnly}>
          <Form.Item name="name" label="模板名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item
            name="purpose"
            label="任务用途"
            rules={[{ required: true }]}
          >
            <Select
              options={["diagram", "analysis", "review"].map((value) => ({
                value,
                label: purposeLabels[value],
              }))}
            />
          </Form.Item>
          <Form.Item
            name="instructions"
            label="分析指引"
            rules={[{ required: true }]}
          >
            <Input.TextArea rows={14} maxLength={16000} showCount />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
function Audit() {
  const r = useResource("/platform-events");
  return (
    <>
      <Alert
        type="info"
        title="记录配置变更、任务发起、账号与项目成员调整，不记录密钥或完整项目正文。"
      />
      {r.error && <Alert type="error" title={r.error} />}
      <Table
        rowKey="id"
        dataSource={r.data?.items || []}
        loading={r.loading}
        columns={[
          { title: "时间", dataIndex: "at", render: date },
          { title: "操作", dataIndex: "action" },
          { title: "操作者", dataIndex: "actor" },
          { title: "对象", dataIndex: "resource_id" },
        ]}
        pagination={{ pageSize: 15 }}
      />
    </>
  );
}
export default function ModelSettings() {
  const { user } = useAuth();
  return (
    <>
      <PageTitle
        title="模型服务与分析规则"
        description="统一管理模型连接、用途分配、健康状态和组织提示词。"
      />
      <Card>
        <Tabs
          items={[
            { key: "providers", label: "模型服务", children: <Models /> },
            { key: "prompts", label: "提示词模板", children: <Prompts /> },
            ...(user.role === "admin"
              ? [{ key: "audit", label: "管理记录", children: <Audit /> }]
              : []),
          ]}
        />
      </Card>
    </>
  );
}
