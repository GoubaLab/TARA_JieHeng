import React, { useEffect, useState } from "react";
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
  Switch,
  Table,
  Tag,
  Typography,
  Upload,
} from "antd";
import {
  PlusOutlined,
  UploadOutlined,
  SearchOutlined,
} from "@ant-design/icons";
import { validateKnowledgePackPreview } from "./knowledge-pack.js";
import { PageTitle, useWorkbench } from "./Workbench.jsx";
import { useAuth } from "./AuthGate.jsx";
import { request, date, navigate } from "./api.js";
import { mutateResource, pickFields, useResource } from "./platform-ui.js";
const { Text, Paragraph } = Typography;
export default function KnowledgeCenter() {
  const { ws, projects } = useWorkbench(),
    { user } = useAuth(),
    { message } = App.useApp();
  const scope = ws?.id || "",
    r = useResource(
      `/knowledge-documents?workspace_id=${encodeURIComponent(scope)}`,
    ),
    providers = useResource("/model-providers");
  const [editing, setEditing] = useState(null),
    [viewing, setViewing] = useState(null),
    [form] = Form.useForm(),
    [indexForm] = Form.useForm(),
    [indexing, setIndexing] = useState(null),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [query, setQuery] = useState(""),
    [embed, setEmbed] = useState(""),
    [allowExternal, setAllowExternal] = useState(false),
    [results, setResults] = useState(null),
    [pack, setPack] = useState(null),
    [importNotices, setImportNotices] = useState([]);
  const canEdit =
    (ws?.access_role || user.role) !== "viewer" &&
    (user.role === "admin" || !!scope);
  const embeddingProviders = (providers.data?.items || []).filter(
    (p) => p.enabled && p.models.embedding,
  );
  useEffect(() => {
    setEditing(null);
    setViewing(null);
    setResults(null);
    setError("");
  }, [scope]);
  const open = async (doc, readOnly = false) => {
    try {
      const full = doc.id
        ? await request(`/knowledge-documents/${doc.id}`)
        : doc;
      if (readOnly) {
        setViewing(full);
        return;
      }
      setEditing(full);
      form.resetFields();
      form.setFieldsValue({...full, reviewed: false});
    } catch (e) {
      setError(e.message);
    }
  };
  const create = () =>
    open({
      name: "",
      source: "企业自有资料",
      edition: "v1",
      content: "",
      tags: [],
      workspace_id: scope,
      enabled: true,
    });
  const save = async () => {
    try {
      const value = await form.validateFields();
      setBusy(true);
      await mutateResource(
        editing.id
          ? `/knowledge-documents/${editing.id}`
          : "/knowledge-documents",
        pickFields(value, [
          "name",
          "source",
          "edition",
          "content",
          "workspace_id",
          "tags",
          "enabled",
          "reviewed",
        ]),
        editing.revision,
        editing.id ? "PUT" : "POST",
      );
      setEditing(null);
      r.refresh();
      message.success("资料已保存；如正文或版本变化，请重新建立语义索引");
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  const previewPack = async (file) => {
    try {
      if (file.size > 2000000) throw Error("知识包不能超过 2 MB");
      const value = JSON.parse(await file.text());
      setPack(validateKnowledgePackPreview(value));
    } catch (e) { setError(e.message); }
    return false;
  };
  const importPack = async () => {
    setBusy(true);
    try {
      const result = await mutateResource("/knowledge-imports", pack);
      setPack(null); setImportNotices(result.review_required || []); r.refresh();
      message.success(`新增 ${result.created_count} 条待复核资料，已存在 ${result.existing_count} 条；另有 ${result.review_required?.length || 0} 项来源复核通知`);
    } catch(e) { setError(e.message); } finally { setBusy(false); }
  };
  const upload = async (file) => {
    try {
      if (file.size > 500000) throw Error("文本文件不能超过 500 KB");
      const text = await file.text();
      if (text.length > 120000) throw Error("正文不能超过 120,000 字符");
      form.setFieldsValue({
        content: text,
        name: form.getFieldValue("name") || file.name,
        source: form.getFieldValue("source") || file.name,
      });
    } catch (e) {
      setError(e.message);
    }
    return false;
  };
  const startIndex = async () => {
    try {
      const values = await indexForm.validateFields();
      setBusy(true);
      const project = await request(`/workspaces/${values.workspace_id}`);
      const job = await mutateResource(
        `/workspaces/${project.id}/ai-tasks`,
        {
          kind: "knowledge_index",
          provider_ids: [values.provider_id],
          document_ids: [indexing.id],
          allow_external: !!values.allow_external,
        },
        project.revision,
      );
      setIndexing(null);
      message.success("已创建索引任务，可在辅助分析中查看进度");
      navigate("assistant", job.workspace_id);
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  const search = async () => {
    if (!query.trim()) return;
    setBusy(true);
    setError("");
    try {
      setResults(
        await mutateResource("/knowledge-searches", {
          workspace_id: scope,
          query,
          embedding_provider_id: embed,
          allow_external: allowExternal,
        }),
      );
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <>
      <PageTitle
        title="企业知识资料"
        description={
          ws
            ? `当前范围：组织共享资料与「${ws.title}」的项目资料。引用保留来源、版本和片段。`
            : "管理组织共享的分析经验、技术规范和验证方法。项目资料在对应工作区维护。"
        }
        extra={
          <Space wrap>
            {user.role === "admin" && <>
              <Button onClick={async () => { try { setPack(validateKnowledgePackPreview(await request("/knowledge-packs/public-cases"))); } catch(e) { setError(e.message); } }}>公开案例种子</Button>
              <Upload accept=".json" beforeUpload={previewPack} showUploadList={false}><Button icon={<UploadOutlined />}>导入知识包</Button></Upload>
            </>}
            {canEdit && <Button type="primary" icon={<PlusOutlined />} onClick={create}>添加资料</Button>}
          </Space>
        }
      />
      {(error || r.error) && (
        <Alert
          className="section-card"
          type="error"
          title={error || r.error}
          closable
          onClose={() => setError("")}
        />
      )}
      {!!importNotices.length && <Card title="本次来源复核通知" className="section-card"><Paragraph>请核对官方来源后处理受影响资料。通知已保存至管理记录，本次未自动停用资料。</Paragraph><pre style={{whiteSpace:"pre-wrap",maxHeight:320,overflow:"auto"}}>{JSON.stringify(importNotices,null,2)}</pre></Card>}
      <Card className="section-card">
        <Alert
          type="info"
          title="知识包导入后先复核事实、适用范围与验证方法，再启用检索。公开漏洞案例不等于当前产品存在漏洞。项目法规引用来自已勾选条款。"
        />
        <Table
          rowKey="id"
          loading={r.loading}
          dataSource={r.data?.items || []}
          scroll={{ x: 940 }}
          tableLayout="fixed"
          columns={[
            {
              title: "资料",
              width: 210,
              dataIndex: "name",
              render: (v, d) => (
                <Button type="link" style={{whiteSpace:"normal",height:"auto",textAlign:"left",padding:0,display:"block",width:"100%"}} onClick={() => open(d, true)}>
                  {v}
                </Button>
              ),
            },
            {
              title: "来源 / 版本",
              width: 180,
              render: (_, d) => (
                <>
                  <Text ellipsis={{tooltip:d.source}} style={{maxWidth:"100%"}}>{d.source}</Text>
                  <br />
                  <Text type="secondary">{d.edition}</Text>
                </>
              ),
            },
            {
              title: "范围",
              width: 95,
              dataIndex: "workspace_id",
              render: (v) => (v ? "当前项目" : "组织共享"),
            },
            {
              title: "正文",
              width: 90,
              render: (_, d) => `${d.characters} 字符 · ${d.chunks} 片段`,
            },
            {
              title: "语义索引",
              width: 130,
              render: (_, d) => (
                <Space orientation="vertical">
                  <Tag
                    color={
                      d.index_state === "indexed"
                        ? "success"
                        : d.index_state === "stale"
                          ? "warning"
                          : "default"
                    }
                  >
                    {
                      {
                        indexed: "已建立",
                        stale: "版本已变，待重建",
                        not_indexed: "未建立",
                      }[d.index_state]
                    }
                  </Tag>
                  <Text type="secondary">
                    {d.index_model || (d.enabled ? "关键词检索可用" : "启用后可检索")}
                  </Text>
                </Space>
              ),
            },
            {
              title: "状态",
              width: 85,
              dataIndex: "enabled",
              render: (v, d) => <Tag color={v ? "success" : d.review_status === "pending" ? "warning" : "default"}>{v ? "已启用" : d.review_status === "pending" ? "待复核" : d.review_status === "superseded" ? "已有新版" : "停用"}</Tag>,
            },
            {
              title: "操作",
              fixed: "right",
              width: 130,
              render: (_, d) =>
                canEdit && (user.role === "admin" || d.workspace_id) ? (
                  <Space>
                    <Button type="link" onClick={() => open(d)}>
                      编辑
                    </Button>
                    <Button
                      type="link"
                      disabled={!d.enabled}
                      onClick={() => {
                        setIndexing(d);
                        indexForm.resetFields();
                        indexForm.setFieldsValue({
                          workspace_id: scope || undefined,
                          allow_external: false,
                        });
                      }}
                    >
                      建索引
                    </Button>
                  </Space>
                ) : null,
            },
          ]}
          pagination={{ pageSize: 10 }}
        />
      </Card>
      <Card title="检索验证">
        <Space orientation="vertical" style={{ width: "100%" }}>
          <Space wrap>
            <Select
              style={{ width: 260 }}
              value={embed}
              onChange={setEmbed}
              options={[
                { value: "", label: "仅关键词 · 不调用模型" },
                ...embeddingProviders.map((p) => ({
                  value: p.id,
                  label: `${p.name} · ${p.models.embedding}`,
                })),
              ]}
            />
            {embeddingProviders.find((p) => p.id === embed)?.network_scope ===
              "external" && (
              <Checkbox
                checked={allowExternal}
                onChange={(e) => setAllowExternal(e.target.checked)}
              >
                允许将本次查询发送到所选外部向量服务
              </Checkbox>
            )}
          </Space>
          <Input.Search
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            maxLength={4000}
            placeholder="输入需要查找的威胁、控制措施或验证方法"
            enterButton="检索"
            onSearch={search}
            loading={busy}
          />
          {results && (
            <>
              <Space>
                <Tag color="blue">
                  {results.mode === "hybrid"
                    ? "语义与关键词混合检索"
                    : "关键词检索"}
                </Tag>
                <Text type="secondary">
                  搜索范围：{results.candidate_chunks} 个片段
                </Text>
              </Space>
              {results.warnings?.map((w, i) => (
                <Alert key={i} type="warning" title={w} />
              ))}
              {!results.items.length && (
                <Empty description="没有匹配资料，请调整关键词或接入相关资料。" />
              )}
              {results.items.map((c) => (
                <Card
                  size="small"
                  key={c.id}
                  title={
                    <Space>
                      {c.name}
                      <Tag>{c.edition}</Tag>
                      <Tag>片段 {c.chunk}</Tag>
                    </Space>
                  }
                >
                  <Paragraph style={{ whiteSpace: "pre-wrap" }}>
                    {c.text}
                  </Paragraph>
                  <Text type="secondary">来源：{c.source}</Text>
                </Card>
              ))}
            </>
          )}
        </Space>
      </Card>
      <Modal
        open={!!editing}
        title={editing?.id ? "更新知识资料" : "添加知识资料"}
        width={850}
        onCancel={() => setEditing(null)}
        onOk={save}
        confirmLoading={busy}
        okText="保存资料"
        destroyOnHidden
      >
        {error && <Alert type="error" title={error} />}
        <Form form={form} layout="vertical">
          <Row gutter={16}>
            <Col span={16}>
              <Form.Item
                name="name"
                label="资料名称"
                rules={[{ required: true }]}
              >
                <Input maxLength={200} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="edition"
                label="版本"
                rules={[{ required: true }]}
              >
                <Input maxLength={200} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item
            name="source"
            label="来源说明"
            rules={[{ required: true }]}
          >
            <Input placeholder="文件名、发布机构或内部规范编号" />
          </Form.Item>
          <Form.Item name="workspace_id" label="使用范围">
            <Select
              disabled={!!editing?.id || user.role !== "admin"}
              options={[
                ...(user.role === "admin"
                  ? [{ value: "", label: "组织共享" }]
                  : []),
                ...(ws
                  ? [{ value: ws.id, label: `当前项目 · ${ws.title}` }]
                  : []),
              ]}
            />
          </Form.Item>
          <Upload
            accept=".txt,.md"
            beforeUpload={upload}
            showUploadList={false}
          >
            <Button icon={<UploadOutlined />}>
              从文本 / Markdown 读取正文
            </Button>
          </Upload>
          <Form.Item name="content" label="正文" rules={[{ required: true }]}>
            <Input.TextArea rows={14} maxLength={120000} />
          </Form.Item>
          <Form.Item name="tags" label="标签">
            <Select mode="tags" />
          </Form.Item>
          {editing?.source_record && <Form.Item name="reviewed" valuePropName="checked"><Checkbox>已复核事实、适用条件、不适用情形与证据来源；启用本版会停用同来源的旧版</Checkbox></Form.Item>}
          <Form.Item name="enabled" label="纳入检索" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
      <Drawer
        open={!!viewing}
        title={viewing?.name}
        width={800}
        onClose={() => setViewing(null)}
      >
        <Descriptions
          column={1}
          items={[
            { key: "source", label: "来源", children: viewing?.source },
            { key: "edition", label: "版本", children: viewing?.edition },
            { key: "cves", label: "CVE", children: viewing?.source_record?.cve_ids?.join("、") || "—" },
            { key: "review", label: "资料复核", children: viewing?.reviewed_at ? `${viewing.reviewed_by === "local" ? "本地管理员" : viewing.reviewed_by} · ${date(viewing.reviewed_at)}` : "尚无复核记录" },
            { key: "evidence", label: "证据来源", children: <Space orientation="vertical">{viewing?.source_record?.evidence_urls?.map(url => <a key={url} href={url} target="_blank" rel="noreferrer">{url}</a>)}</Space> },
            {
              key: "updated",
              label: "更新时间",
              children: date(viewing?.updated_at),
            },
          ]}
        />
        <Paragraph style={{ whiteSpace: "pre-wrap" }}>
          {viewing?.content}
        </Paragraph>
      </Drawer>
      <Modal open={!!pack} title="知识包导入预览" width={900} onCancel={() => setPack(null)} onOk={importPack} confirmLoading={busy} okText="导入为待复核资料">
        {error && <Alert type="error" title={error} />}
        <Paragraph>来源包：{pack?.pack_id} · 版本：{pack?.version}。本次不会启用资料，也不会覆盖已整理的内容。</Paragraph>
        {!!pack?.review_required?.length && <Alert type="warning" title={`来源清单另有 ${pack.review_required.length} 项需要复核。`} description={<pre style={{whiteSpace:"pre-wrap",maxHeight:220,overflow:"auto"}}>{JSON.stringify(pack.review_required,null,2)}</pre>} />}
        <Table rowKey="id" size="small" dataSource={pack?.entries || []} pagination={{pageSize:5}} columns={[{title:"案例",dataIndex:"name"},{title:"版本",dataIndex:"edition"},{title:"CVE",render:(_,d)=>(d.cve_ids||[]).join("、")}]} expandable={{expandedRowRender:d=><Paragraph style={{whiteSpace:"pre-wrap"}}>{d.content}</Paragraph>}} />
      </Modal>
      <Modal
        open={!!indexing}
        title="建立语义索引"
        onCancel={() => setIndexing(null)}
        onOk={startIndex}
        confirmLoading={busy}
        okText="创建索引任务"
      >
        <Paragraph>
          将此资料分段后交给所选向量服务计算语义向量。原文和索引保存在企业数据库，任务可取消。
        </Paragraph>
        {error && <Alert type="error" title={error} />}
        <Form form={indexForm} layout="vertical">
          <Form.Item
            name="workspace_id"
            label="任务所属项目"
            rules={[{ required: true }]}
          >
            <Select
              options={projects
                .filter(
                  (p) =>
                    !indexing?.workspace_id || p.id === indexing.workspace_id,
                )
                .map((p) => ({ value: p.id, label: p.title }))}
            />
          </Form.Item>
          <Form.Item
            name="provider_id"
            label="向量服务"
            rules={[{ required: true }]}
          >
            <Select
              options={embeddingProviders.map((p) => ({
                value: p.id,
                label: `${p.name} · ${p.network_scope === "internal" ? "内网" : "外部"}`,
              }))}
            />
          </Form.Item>
          <Form.Item name="allow_external" valuePropName="checked">
            <Checkbox>若选择外部服务，允许发送此资料正文用于建立索引</Checkbox>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
