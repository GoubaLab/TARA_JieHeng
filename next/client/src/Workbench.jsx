import { recordCode, pathNumberKey, buildCodeIndex } from "./numbering.js";
import React, {
  createContext,
  useContext,
  useEffect,
  useState,
  useCallback,
  useRef,
} from "react";
import {
  Layout,
  Menu,
  Breadcrumb,
  Select,
  Button,
  Space,
  Typography,
  Tag,
  Card,
  Table,
  Input,
  Row,
  Col,
  Statistic,
  Tabs,
  Alert,
  Empty,
  Progress,
  Steps,
  Descriptions,
  Tooltip,
  Spin,
  App as AntApp,
  Badge,
  Segmented,
  Result,
} from "antd";
import {
  AppstoreOutlined,
  ProjectOutlined,
  ApartmentOutlined,
  FileTextOutlined,
  AlertOutlined,
  SafetyCertificateOutlined,
  AuditOutlined,
  DatabaseOutlined,
  PlusOutlined,
  ReloadOutlined,
  ExportOutlined,
  SearchOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  ArrowRightOutlined,
  SettingOutlined,
  QuestionCircleOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  LinkOutlined,
  UnorderedListOutlined,
} from "@ant-design/icons";
import {
  request,
  allWorkspaces,
  workspaceEvents,
  command,
  navigate,
  match,
  date,
  properties,
  impacts,
  feasibility,
  treatments,
  pages,
  actions,
  evidenceStatus,
  exportFile,
} from "./api.js";
import {
  EntityEditor,
  ProjectCreator,
  ReferencePicker,
  ReferenceDetail,
  FlowEditor,
} from "./editors.jsx";
import { KnowledgeAutomation } from "./automation.jsx";
import { RegulationsView, RegulationPicker } from "./regulations.jsx";
import { Diagram } from "./diagram.js";
import AssessmentSettings, { MethodPreview } from "./AssessmentSettings.jsx";
import { domains, effectiveMethod } from "./method-config.js";
const { Header, Sider, Content } = Layout;
const { Text, Title, Paragraph } = Typography;
const Ctx = createContext();
export const useWorkbench = () => useContext(Ctx);
const navItems = [
  ["overview", <AppstoreOutlined />],
  ["definition", <FileTextOutlined />],
  ["architecture", <ApartmentOutlined />],
  ["assessment", <AlertOutlined />],
  ["treatment", <SafetyCertificateOutlined />],
  ["verification", <AuditOutlined />],
  ["settings", <SettingOutlined />],
];
const stageNames = {
  definition: "相关项定义",
  architecture: "架构与资产",
  assessment: "风险分析",
  treatment: "风险处置",
  verification: "需求验证",
};
export function Risk({ level, provisional = false, method }) {
  const context = useContext(Ctx);
  const levels = effectiveMethod(method || context?.ws?.method).risk_levels;
  const grade = levels[level - 1];
  return level == null ? (
    <Tag>待评估</Tag>
  ) : (
    <Tooltip
      title={
        provisional
          ? "方法尚未确认，此值为暂算结果"
          : grade?.description || `风险等级 ${level} · ${grade?.label || ""}`
      }
    >
      <Tag color={["", "green", "cyan", "gold", "orange", "red"][level]}>
        {level} · {grade?.label || "风险"}
        {provisional ? " · 暂算" : ""}
      </Tag>
    </Tooltip>
  );
}
export function Status({ value }) {
  const map = {
    missing: ["待验证", "default"],
    pending: ["待验证", "processing"],
    fail: ["验证未通过", "error"],
    stale: ["证据已失效", "warning"],
    pass: ["验证通过", "success"],
  };
  const [text, status] = map[value] || map.missing;
  return <Badge status={status} text={text} />;
}
export function PageTitle({ title, description, extra }) {
  return (
    <div className="page-title">
      <div>
        <Title level={3}>{title}</Title>
        {description && <Text type="secondary">{description}</Text>}
      </div>
      <Space wrap>{extra}</Space>
    </div>
  );
}
const compactPage = {
  defaultPageSize: 10,
  showSizeChanger: true,
  showTotal: (n) => `共 ${n} 条`,
};
function StatCards({ items }) {
  return (
    <Row gutter={[16, 16]} className="stat-row">
      {items.map(([title, value, sub]) => (
        <Col xs={12} xl={6} key={title}>
          <Card size="small">
            <Statistic title={title} value={value} />
            <Text type="secondary" className="stat-note">
              {sub}
            </Text>
          </Card>
        </Col>
      ))}
    </Row>
  );
}
export default function Workbench() {
  const { message, modal } = AntApp.useApp();
  const [route, setRoute] = useState(location.hash);
  const [ws, setWs] = useState(null),
    [review, setReview] = useState(null),
    [projects, setProjects] = useState([]),
    [catalog, setCatalog] = useState(null),
    [loading, setLoading] = useState(true),
    [error, setError] = useState(""),
    [collapsed, setCollapsed] = useState(false),
    [editor, setEditor] = useState(null),
    [createOpen, setCreateOpen] = useState(false),
    [picker, setPicker] = useState(null),
    [reference, setReference] = useState(null),
    [automation, setAutomation] = useState(null),
    [flow, setFlow] = useState(null),
    [regulationPicker, setRegulationPicker] = useState(null);
  const parts = route.replace(/^#\/?/, "").split("/"),
    id = parts[0] === "workspace" ? parts[1] : null,
    page = id ? parts[2] || "overview" : parts[0] || "workspaces";
  const sequence = useRef(0);
  const activeWorkspaceId = useRef(id);
  activeWorkspaceId.current = id;
  useEffect(() => {
    setEditor(null);
    setPicker(null);
    setAutomation(null);
    setFlow(null);
    setReference(null);
    setRegulationPicker(null);
  }, [id]);
  useEffect(() => {
    const h = () => {
      setRoute(location.hash);
      window.scrollTo(0, 0);
    };
    window.addEventListener("hashchange", h);
    return () => window.removeEventListener("hashchange", h);
  }, []);
  const reload = useCallback(
    async ({ quiet = false } = {}) => {
      const seq = ++sequence.current;
      setError("");
      if (!quiet) setLoading(true);
      try {
        const [list, library, workspace, inspection] = await Promise.all([
          allWorkspaces(),
          request("/catalog"),
          id ? request("/workspaces/" + id) : null,
          id ? request("/workspaces/" + id + "/review") : null,
        ]);
        if (seq !== sequence.current) return;
        setProjects(list.items);
        setCatalog(library);
        setWs(workspace);
        setReview(inspection);
      } catch (e) {
        if (seq === sequence.current) setError(e.message);
      } finally {
        if (seq === sequence.current) setLoading(false);
      }
    },
    [id],
  );
  useEffect(() => {
    reload();
  }, [reload]);
  const save = async (action, payload) => {
    const result = await command(ws, action, payload);
    if (activeWorkspaceId.current === ws.id) await reload({ quiet: true });
    return result;
  };
  const openIssue = (issue) => {
    if (issue.reference) {
      if (ws.assets.some((a) => a.id === issue.reference))
        return setEditor(
          issue.stage === "assessment"
            ? { kind: "scenario", assetId: issue.reference }
            : { kind: "asset", id: issue.reference },
        );
      if (ws.requirements.some((r) => r.id === issue.reference))
        return setEditor({
          kind:
            issue.stage === "verification" &&
            !review.requirements?.find(
              (r) => r.requirement_id === issue.reference,
            )?.draft
              ? "evidence"
              : "requirement",
          id: issue.reference,
        });
      if (ws.scenarios.some((s) => s.id === issue.reference))
        return setEditor({
          kind:
            issue.stage === "assessment" ||
            !review.evaluations.find((e) => e.scenario_id === issue.reference)
              ?.level
              ? "scenario"
              : "decision",
          id: issue.reference,
        });
    }
    if (issue.message.includes("评估方法")) return navigate("settings", ws.id);
    navigate(issue.stage, ws.id);
  };
  const ctx = {
    ws,
    review,
    projects,
    catalog,
    page,
    save,
    reload,
    message,
    modal,
    setEditor,
    setCreateOpen,
    setPicker,
    setRegulationPicker,
    setReference,
    setAutomation,
    setFlow,
    openIssue,
  };
  const menu = [
    { key: "workspaces", icon: <ProjectOutlined />, label: "项目管理" },
    ...(id
      ? [
          {
            type: "group",
            label: "项目工作区",
            children: navItems.map(([key, icon]) => ({
              key,
              icon,
              label: pages[key],
            })),
          },
        ]
      : []),
    {
      type: "group",
      label: "资源目录",
      children: [
        { key: "library", icon: <DatabaseOutlined />, label: "参考资料" },
        { key: "regulations", icon: <AuditOutlined />, label: "法规数据库" },
      ],
    },
  ];
  const content =
    loading || (!error && (id ? ws?.id !== id || !review : !catalog)) ? (
      <div className="page-loading">
        <Space orientation="vertical" align="center">
          <Spin size="large" />
          <Text type="secondary">正在加载项目数据</Text>
        </Space>
      </div>
    ) : error ? (
      <Result
        status="warning"
        title="数据加载失败"
        subTitle={error}
        extra={<Button onClick={reload}>重新加载</Button>}
      />
    ) : !id ? (
      page === "regulations" ? (
        <RegulationsView />
      ) : page === "library" ? (
        <Library />
      ) : (
        <Projects />
      )
    ) : (
      {
        overview: <Overview />,
        definition: <Definition />,
        architecture: <Architecture />,
        assessment: <Assessment />,
        treatment: <Treatment />,
        verification: <Verification />,
        settings: <AssessmentSettings />,
        library: <Library />,
        regulations: <RegulationsView />,
      }[page] || <Overview />
    );
  return (
    <Ctx.Provider value={ctx}>
      <Layout className="app-layout">
        <Header className="app-header">
          <div className="product-logo">
            <span className="logo-symbol">
              <SafetyCertificateOutlined />
            </span>
            <strong>界衡</strong>
            <span className="product-name">威胁建模平台</span>
          </div>
          <div className="header-right">
            <Tag color="blue" variant="filled">
              {ws
                ? (ws.item.domain || "automotive") === "automotive"
                  ? "汽车 · ISO/SAE 21434"
                  : domains[ws.item.domain]
                : "通用威胁建模"}
            </Tag>
            <Badge status="default" text="本地工作台" />
            <Tooltip title="单机工作区；角色与多人审批尚未启用">
              <QuestionCircleOutlined />
            </Tooltip>
          </div>
        </Header>
        <Layout>
          <Sider
            width={216}
            collapsedWidth={64}
            collapsed={collapsed}
            theme="light"
            className="app-sider"
          >
            <Menu
              mode="inline"
              selectedKeys={[page]}
              items={menu}
              onClick={({ key }) =>
                key === "workspaces" ? navigate(key) : navigate(key, id)
              }
            />
            <Button
              type="text"
              className="collapse-control"
              icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
              onClick={() => setCollapsed(!collapsed)}
            >
              {!collapsed ? "收起导航" : ""}
            </Button>
          </Sider>
          <Layout className="workspace-layout">
            <div className="workspace-toolbar">
              <Breadcrumb
                items={[
                  { title: <a href="#/workspaces">项目管理</a> },
                  ...(ws
                    ? [
                        { title: ws.title },
                        { title: pages[page] || "项目总览" },
                      ]
                    : [
                        {
                          title: pages[page] || "项目列表",
                        },
                      ]),
                ]}
              />
              <Space>
                {ws && (
                  <>
                    <Select
                      aria-label="切换项目"
                      value={ws.id}
                      style={{ width: 210 }}
                      options={projects.map((p) => ({
                        value: p.id,
                        label: p.title,
                      }))}
                      onChange={(v) => navigate("overview", v)}
                    />
                    <Tag>修订 {ws.revision}</Tag>
                  </>
                )}
                <Tooltip title="刷新数据">
                  <Button
                    icon={<ReloadOutlined />}
                    aria-label="刷新数据"
                    onClick={reload}
                  />
                </Tooltip>
              </Space>
            </div>
            <Content className="workspace-content">{content}</Content>
          </Layout>
        </Layout>
      </Layout>
      <ProjectCreator open={createOpen} onClose={() => setCreateOpen(false)} />
      <EntityEditor editor={editor} onClose={() => setEditor(null)} />
      <ReferencePicker config={picker} onClose={() => setPicker(null)} />
      <ReferenceDetail
        referenceKey={reference}
        onClose={() => setReference(null)}
      />
      <KnowledgeAutomation
        config={automation}
        onClose={() => setAutomation(null)}
      />
      <FlowEditor config={flow} onClose={() => setFlow(null)} />
      <RegulationPicker
        config={regulationPicker}
        onClose={() => setRegulationPicker(null)}
      />
    </Ctx.Provider>
  );
}
function Projects() {
  const { projects, setCreateOpen } = useWorkbench();
  const [q, setQ] = useState("");
  const rows = projects.filter((p) => match([p.title, p.description], q));
  return (
    <>
      <PageTitle
        title="项目管理"
        description="集中管理项目、分析记录与评审基线。"
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateOpen(true)}
          >
            新建项目
          </Button>
        }
      />
      <StatCards
        items={[
          ["项目总数", projects.length, "本地项目工作区"],
          [
            "已识别资产",
            projects.reduce((s, p) => s + p.assets, 0),
            "所有项目资产记录",
          ],
          [
            "风险场景",
            projects.reduce((s, p) => s + p.scenarios, 0),
            "损害、威胁与攻击路径",
          ],
          [
            "已建评审基线",
            projects.reduce((s, p) => s + p.baselines, 0),
            "保留历史评审快照",
          ],
        ]}
      />
      <Card>
        <div className="table-toolbar">
          <Space>
            <Text strong>项目列表</Text>
            <Tag>{projects.length}</Tag>
          </Space>
          <Input.Search
            aria-label="搜索项目"
            placeholder="搜索项目名称或说明"
            allowClear
            value={q}
            onChange={(e) => setQ(e.target.value)}
            style={{ width: 300 }}
          />
        </div>
        <Table
          rowKey="id"
          pagination={compactPage}
          dataSource={rows}
          locale={{
            emptyText: (
              <Empty description="暂无项目，创建项目后即可开始分析">
                <Button type="primary" onClick={() => setCreateOpen(true)}>
                  新建项目
                </Button>
              </Empty>
            ),
          }}
          columns={[
            {
              title: "项目名称",
              dataIndex: "title",
              render: (v, p) => (
                <Space orientation="vertical" size={2}>
                  <Button
                    type="link"
                    className="text-button"
                    onClick={() => navigate("overview", p.id)}
                  >
                    {v}
                  </Button>
                  <Text type="secondary" className="cell-description">
                    {p.description || "未填写项目说明"}
                  </Text>
                </Space>
              ),
            },
            {
              title: "资产",
              dataIndex: "assets",
              width: 90,
              sorter: (a, b) => a.assets - b.assets,
            },
            {
              title: "风险场景",
              dataIndex: "scenarios",
              width: 110,
              sorter: (a, b) => a.scenarios - b.scenarios,
            },
            { title: "评审基线", dataIndex: "baselines", width: 110 },
            {
              title: "最近更新",
              dataIndex: "updated_at",
              width: 190,
              render: date,
              sorter: (a, b) => a.updated_at.localeCompare(b.updated_at),
              defaultSortOrder: "descend",
            },
            {
              title: "操作",
              width: 110,
              render: (_, p) => (
                <Button type="link" onClick={() => navigate("overview", p.id)}>
                  进入项目
                </Button>
              ),
            },
          ]}
        />
      </Card>
    </>
  );
}
function Overview() {
  const { ws, review, setEditor, openIssue } = useWorkbench();
  const done = review.stages.filter((s) => s.complete).length;
  const counts = Array.from(
    { length: 5 },
    (_, i) => review.evaluations.filter((e) => e.level === i + 1).length,
  );
  return (
    <>
      <PageTitle
        title="项目总览"
        description={ws.title}
        extra={
          <Button
            icon={<ExportOutlined />}
            onClick={() => navigate("verification", ws.id)}
          >
            检查与导出报告
          </Button>
        }
      />
      <Card size="small" className="project-context">
        <Descriptions
          size="small"
          column={{ xs: 1, md: 3, xl: 4 }}
          items={[
            {
              key: "owner",
              label: "项目负责人",
              children: ws.item.owner || (
                <Button
                  type="link"
                  size="small"
                  onClick={() => navigate("definition", ws.id)}
                >
                  待填写
                </Button>
              ),
            },
            { key: "stage", label: "生命周期", children: ws.item.lifecycle },
            {
              key: "method",
              label: "评估方法",
              children: (
                <Button
                  type="link"
                  size="small"
                  onClick={() => navigate("settings", ws.id)}
                >
                  {ws.method_confirmed ? "已确认" : "待确认"}
                </Button>
              ),
            },
            {
              key: "revision",
              label: "最近更新",
              children: date(ws.updated_at),
            },
          ]}
        />
      </Card>
      <StatCards
        items={[
          ["待办事项", review.issues.length, "根据工作产品自动检查"],
          [
            "风险场景",
            ws.scenarios.length,
            `${review.evaluations.filter((e) => e.level === null).length} 项尚未完成评估`,
          ],
          ["高优先级风险", counts[3] + counts[4], "初始风险等级 4 / 5"],
          ["检查进度", `${done} / 5`, "工作产品完整性检查"],
        ]}
      />
      <Row gutter={[16, 16]}>
        <Col xs={24} xl={16}>
          <Card
            title="待办事项"
            extra={
              <Button
                type="link"
                onClick={() => navigate("verification", ws.id)}
              >
                查看全部
              </Button>
            }
          >
            <Table
              size="small"
              rowKey={(r, i) => r.reference + "-" + r.message + "-" + i}
              dataSource={review.issues.slice(0, 6)}
              pagination={false}
              locale={{
                emptyText: (
                  <Empty
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                    description="当前工作产品检查通过，可提交评审"
                  />
                ),
              }}
              columns={[
                {
                  title: "工作区",
                  dataIndex: "stage",
                  width: 110,
                  render: (v) => <Tag>{stageNames[v]}</Tag>,
                },
                {
                  title: "待完成内容",
                  dataIndex: "message",
                  render: (value) =>
                    Object.entries(properties).reduce(
                      (text, [key, label]) => text.replaceAll(key, label),
                      value,
                    ),
                },
                {
                  title: "操作",
                  width: 80,
                  render: (_, i) => (
                    <Button
                      size="small"
                      type="link"
                      onClick={() => openIssue(i)}
                    >
                      去处理
                    </Button>
                  ),
                },
              ]}
            />
          </Card>
        </Col>
        <Col xs={24} xl={8}>
          <Card
            title="风险分布"
            extra={
              !ws.method_confirmed && <Tag color="warning">方法待确认</Tag>
            }
          >
            <div className="risk-distribution">
              {counts.map((count, i) => (
                <div className="distribution-row" key={i}>
                  <Risk level={i + 1} />
                  <Progress
                    percent={
                      counts.reduce((a, b) => a + b, 0)
                        ? (count / counts.reduce((a, b) => a + b, 0)) * 100
                        : 0
                    }
                    showInfo={false}
                    strokeColor={
                      ["#52c41a", "#13c2c2", "#faad14", "#fa8c16", "#ff4d4f"][i]
                    }
                  />
                  <strong>{count}</strong>
                </div>
              ))}
            </div>
            <Text type="secondary">
              未完成评估{" "}
              {review.evaluations.filter((e) => e.level === null).length}{" "}
              项，不计入风险分布。
            </Text>
          </Card>
        </Col>
      </Row>
      <Card title="工作区导航" className="section-card">
        <div className="workflow-strip">
          {review.stages.map((stage, i) => (
            <button key={stage.key} onClick={() => navigate(stage.key, ws.id)}>
              <span
                className={stage.complete ? "step-number done" : "step-number"}
              >
                {stage.complete ? <CheckCircleOutlined /> : i + 1}
              </span>
              <div>
                <strong>{stageNames[stage.key]}</strong>
                <span>{stage.complete ? "检查通过" : "继续完善"}</span>
              </div>
              <ArrowRightOutlined />
            </button>
          ))}
        </div>
      </Card>
    </>
  );
}
function Definition() {
  const { ws, setEditor } = useWorkbench();
  return (
    <>
      <PageTitle
        title="相关项定义"
        description="明确系统功能、边界和分析假设，为风险分析提供共同输入。"
        extra={
          <Button type="primary" onClick={() => setEditor({ kind: "item" })}>
            编辑相关项
          </Button>
        }
      />
      <Row gutter={[16, 16]}>
        <Col xs={24} xl={16}>
          <Card title="系统与范围">
            <Descriptions
              bordered
              column={1}
              size="middle"
              items={[
                ["domain", "应用领域"],
                ["purpose", "功能与预期用途"],
                ["boundary", "系统边界与外部接口"],
                ["environment", "运行环境"],
                ["assumptions", "假设与依赖"],
                ["owner", "项目负责人"],
                ["lifecycle", "生命周期阶段"],
              ].map(([key, label]) => ({
                key,
                label,
                children: (
                  <div className="pre-wrap">
                    {(key === "domain"
                      ? domains[ws.item.domain || "automotive"]
                      : ws.item[key]) || <Text type="secondary">待补充</Text>}
                  </div>
                ),
              }))}
            />
          </Card>
        </Col>
        <Col xs={24} xl={8}>
          <Card
            title="评估方法"
            extra={
              <Button type="link" onClick={() => navigate("settings", ws.id)}>
                配置
              </Button>
            }
          >
            <Space orientation="vertical" size={16} style={{ width: "100%" }}>
              <Space>
                <Text strong>{ws.method.name}</Text>
                <Tag color={ws.method_confirmed ? "success" : "warning"}>
                  {ws.method_confirmed ? "已确认" : "待确认"}
                </Tag>
              </Space>
              <Text type="secondary">{ws.method.rationale}</Text>
              <MethodPreview method={ws.method} />
            </Space>
          </Card>
        </Col>
      </Row>
    </>
  );
}
function Architecture() {
  const { ws, catalog, setEditor, setPicker, setAutomation } = useWorkbench();
  const [activeTab, setActiveTab] = useState("canvas");
  return (
    <>
      <PageTitle
        title="架构与资产"
        description="绘制实体和数据流，保存后自动生成并同步资产清单。软件与数据存储在实体内维护。"
        extra={
          <>
            <Button
              disabled={!catalog.categories.length}
              onClick={() =>
                setPicker({
                  category: "systems",
                  multiple: false,
                  onSelect: (rows) =>
                    setEditor({ kind: "asset", source: rows[0] }),
                })
              }
            >
              从参考资料识别资产
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setEditor({ kind: "asset" })}
            >
              新增资产
            </Button>
          </>
        }
      />
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: "canvas",
            label: "系统架构",
            children: (
              <Card className="canvas-card">
                <DiagramPanel onAssets={() => setActiveTab("assets")} />
              </Card>
            ),
          },
          {
            key: "assets",
            label: `资产清单（${ws.assets.length}）`,
            children: <AssetTable />,
          },
        ]}
      />
      <Alert
        className="section-card"
        type="info"
        showIcon
        title="实体与通信建模"
        description="保存架构会自动识别实体功能、内部软件、存储数据与传输数据。再次保存会同步未手工修改的内容；手工补充和已有风险分析会保留。安全属性为可编辑建议，数据流已选属性会直接带入。"
      />
    </>
  );
}
function DiagramPanel({ onAssets }) {
  const { ws, save, message, setFlow } = useWorkbench();
  const el = useRef(),
    diagram = useRef();
  const callbacks = useRef();
  callbacks.current = { save, message, setFlow, onAssets };
  const signature = JSON.stringify([
    ws.architecture,
    ws.numbering,
    ws.numbering_state,
  ]);
  useEffect(() => {
    const instance = new Diagram(el.current, ws.architecture, {
      draftKey: "jieheng.graph." + ws.id,
      getCodes: (graph) => buildCodeIndex(ws, graph),
      escape: (v) =>
        String(v ?? "").replace(
          /[&<>"']/g,
          (c) =>
            ({
              "&": "&amp;",
              "<": "&lt;",
              ">": "&gt;",
              '"': "&quot;",
              "'": "&#39;",
            })[c],
        ),
      toast: (t) => callbacks.current.message.info(t),
      onSave: async (graph) => {
        const result = await callbacks.current.save(
          "replace-architecture",
          graph,
        );
        const sync = result.asset_sync;
        callbacks.current.message.success(
          sync
            ? `架构已保存：新增 ${sync.created} 项资产，更新 ${sync.updated} 项，共 ${sync.total} 项${sync.missing ? `；${sync.missing} 项来源已移除，待核对` : ""}`
            : "架构已保存，资产清单已同步",
        );
      },
      onFlow: (value, done) => callbacks.current.setFlow({ value, done }),
    });
    diagram.current = instance;
    const unload = (e) => {
      if (instance.dirty) {
        e.preventDefault();
        e.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", unload);
    return () => {
      instance.destroy();
      window.removeEventListener("beforeunload", unload);
    };
  }, [ws.id, signature]);
  return (
    <>
      <div className="dfd-root" ref={el} />
      <div className="architecture-assets-summary">
        <Text>已识别 {ws.assets.length} 项资产 · 保存架构后自动同步</Text>
        <Button type="link" onClick={() => callbacks.current.onAssets()}>
          查看资产清单
        </Button>
      </div>
    </>
  );
}
function AssetTable() {
  const { ws, catalog, setEditor, setAutomation, setReference } =
    useWorkbench();
  const [q, setQ] = useState("");
  return (
    <Card>
      <Alert
        type="info"
        showIcon
        title="资产随架构保存自动生成"
        description="实体功能、软件、存储数据和传输数据均可识别。安全属性可按实际业务调整；再次保存架构不会覆盖你手工修改的名称、说明和安全属性。"
        className="section-card"
      />
      <div className="table-toolbar">
        <Text strong>受保护资产</Text>
        <Input.Search
          placeholder="搜索资产名称或描述"
          aria-label="搜索资产"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          allowClear
          style={{ width: 280 }}
        />
      </div>
      <Table
        rowKey="id"
        dataSource={ws.assets.filter((a) =>
          match([a, recordCode(ws, "asset", a.id)], q),
        )}
        pagination={compactPage}
        scroll={{ x: 1120 }}
        columns={[
          numberColumn(ws, "asset"),
          {
            title: "资产名称",
            dataIndex: "name",
            width: 210,
            render: (v, a) => (
              <Button
                type="link"
                className="text-button"
                onClick={() => setEditor({ kind: "asset", id: a.id })}
              >
                {v}
              </Button>
            ),
          },
          {
            title: "识别来源",
            dataIndex: "architecture_source",
            width: 150,
            render: (source) =>
              source ? (
                <Space orientation="vertical" size={4}>
                  <Tag color="blue">
                    {
                      {
                        entity: "实体功能",
                        software: "内部软件",
                        store: "存储数据",
                        flow: "传输数据",
                      }[source.kind]
                    }
                  </Tag>
                  {source.status === "missing" && (
                    <Tag color="warning">来源已移除 · 待核对</Tag>
                  )}
                </Space>
              ) : (
                <Text type="secondary">手工维护</Text>
              ),
          },
          {
            title: "安全属性",
            dataIndex: "properties",
            width: 200,
            render: (v) => (
              <Space size={[0, 4]} wrap>
                {v.map((p) => (
                  <Tag key={p}>{properties[p]}</Tag>
                ))}
              </Space>
            ),
          },
          {
            title: "关联实体",
            dataIndex: "node_ids",
            width: 220,
            render: (ids) =>
              ids
                .map(
                  (id) => ws.architecture.nodes.find((n) => n.id === id)?.name,
                )
                .join("、") || <Tag color="warning">待关联</Tag>,
          },
          {
            title: "参考来源",
            dataIndex: "reference",
            width: 130,
            render: (v) =>
              v ? (
                <Button type="link" onClick={() => setReference(v)}>
                  查看来源
                </Button>
              ) : (
                <Text type="secondary">项目自定义</Text>
              ),
          },
          {
            title: "操作",
            width: catalog.categories.length ? 240 : 110,
            fixed: "right",
            render: (_, a) => (
              <Space>
                {catalog.categories.length > 0 && (
                  <Button
                    type="link"
                    onClick={() =>
                      setAutomation({ assetId: a.id, reference: a.reference })
                    }
                  >
                    选用参考资料
                  </Button>
                )}
                <Button
                  type="link"
                  onClick={() => setEditor({ kind: "scenario", assetId: a.id })}
                >
                  新增风险
                </Button>
              </Space>
            ),
          },
        ]}
      />
    </Card>
  );
}
function Assessment() {
  const { ws, review, catalog, setEditor, setAutomation } = useWorkbench();
  const [q, setQ] = useState(""),
    [status, setStatus] = useState("all");
  const rows = ws.scenarios
    .map((s) => ({
      ...s,
      evaluation: review.evaluations.find((e) => e.scenario_id === s.id) || {
        level: null,
        residual: null,
        decision_stale: false,
      },
    }))
    .filter((s) =>
      match(
        [s.title, s.damage, ws.assets.find((a) => a.id === s.asset_id)?.name],
        q,
      ),
    )
    .filter(
      (s) =>
        status === "all" ||
        (status === "pending"
          ? s.evaluation.level === null
          : status === "high"
            ? s.evaluation.level >= 4
            : s.evaluation.decision_stale),
    );
  return (
    <>
      <PageTitle
        title="风险分析"
        description="以场景为单位关联损害、威胁和攻击路径，依据证据完成影响与可行性评估。"
        extra={
          <>
            <Button
              onClick={() => setAutomation({})}
              disabled={!ws.assets.length || !catalog.categories.length}
            >
              批量选用资料
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setEditor({ kind: "scenario" })}
              disabled={!ws.assets.length}
            >
              新增风险场景
            </Button>
          </>
        }
      />
      {!ws.method_confirmed && (
        <Alert
          className="section-card"
          type="warning"
          showIcon
          title="尚未确认项目评估方法，风险等级为暂算结果。"
          action={
            <Button size="small" onClick={() => navigate("settings", ws.id)}>
              配置方法
            </Button>
          }
        />
      )}
      <Card>
        <div className="table-toolbar">
          <Segmented
            value={status}
            onChange={setStatus}
            options={[
              { value: "all", label: `全部 ${ws.scenarios.length}` },
              { value: "pending", label: "待评估" },
              { value: "high", label: "高优先级" },
              { value: "stale", label: "需复核" },
            ]}
          />
          <Input.Search
            placeholder="搜索场景、资产或损害描述"
            aria-label="搜索风险场景"
            allowClear
            value={q}
            onChange={(e) => setQ(e.target.value)}
            style={{ width: 290 }}
          />
        </div>
        <Table
          dataSource={rows}
          rowKey="id"
          pagination={compactPage}
          scroll={{ x: 1080 }}
          locale={{
            emptyText: (
              <Empty
                description={
                  ws.assets.length
                    ? "暂无风险场景，可从参考资料批量选用或手动创建。"
                    : "先建立实体并识别资产，再开始风险分析。"
                }
              >
                <Button onClick={() => navigate("architecture", ws.id)}>
                  前往架构与资产
                </Button>
              </Empty>
            ),
          }}
          expandable={{
            expandedRowRender: (s) => (
              <div className="expanded-detail">
                <Descriptions
                  size="small"
                  column={1}
                  items={[
                    { key: "damage", label: "损害场景", children: s.damage },
                    { key: "threat", label: "威胁场景", children: s.threat },
                    ...(ws.method.impact_aggregation === "coefficient_sum"
                      ? [
                          {
                            key: "sfop",
                            label: "SFOP 原始总分",
                            children: s.evaluation.impact ?? "待评估",
                          },
                          {
                            key: "score",
                            label: "最高路径风险分值",
                            children: s.evaluation.score ?? "待评估",
                          },
                        ]
                      : []),
                    {
                      key: "impact",
                      label: "影响依据",
                      children: s.impact.rationale || "待补充",
                    },
                  ]}
                />
                <Table
                  size="small"
                  rowKey="id"
                  pagination={false}
                  dataSource={s.paths}
                  columns={[
                    numberColumn(ws, "path", s.id),
                    { title: "攻击路径", dataIndex: "name" },
                    {
                      title: "步骤与前提",
                      dataIndex: "steps",
                      render: (v) => v || "待补充",
                    },
                    {
                      title: "可行性",
                      dataIndex: "feasibility",
                      render: (v) =>
                        v == null ? (
                          <Tag>待评估</Tag>
                        ) : (
                          effectiveMethod(ws.method).feasibility_levels[v - 1]
                            .label
                        ),
                    },
                    {
                      title: "判定依据",
                      dataIndex: "rationale",
                      render: (v) => v || "待补充",
                    },
                  ]}
                />
              </div>
            ),
          }}
          columns={[
            numberColumn(ws, "scenario"),
            {
              title: "风险场景",
              dataIndex: "title",
              width: 280,
              render: (v, s) => (
                <Button
                  type="link"
                  className="text-button"
                  onClick={() => setEditor({ kind: "scenario", id: s.id })}
                >
                  {v}
                </Button>
              ),
            },
            {
              title: "关联资产",
              dataIndex: "asset_id",
              width: 170,
              render: (id) => ws.assets.find((a) => a.id === id)?.name,
            },
            {
              title: "安全属性",
              dataIndex: "property",
              width: 100,
              render: (v) => properties[v],
            },
            {
              title: "路径",
              dataIndex: "paths",
              width: 70,
              render: (v) => v.length,
            },
            ...(ws.method.impact_aggregation === "coefficient_sum"
              ? [
                  {
                    title: "SFOP 总分",
                    width: 110,
                    render: (_, s) => s.evaluation.impact ?? "待评估",
                  },
                  {
                    title: "风险分值",
                    width: 110,
                    render: (_, s) => s.evaluation.score ?? "待评估",
                  },
                ]
              : []),
            {
              title: "初始风险",
              width: 115,
              sorter: (a, b) =>
                (a.evaluation.level || 0) - (b.evaluation.level || 0),
              render: (_, s) => (
                <Risk
                  level={s.evaluation.level}
                  provisional={!ws.method_confirmed}
                />
              ),
            },
            {
              title: "残余风险",
              width: 110,
              render: (_, s) => <Risk level={s.evaluation.residual} />,
            },
            {
              title: "状态",
              width: 125,
              render: (_, s) =>
                s.evaluation.decision_stale ? (
                  <Tag color="warning">决策需复核</Tag>
                ) : s.evaluation.level == null ? (
                  <Tag>待评估</Tag>
                ) : (
                  <Tag color="processing">已评估</Tag>
                ),
            },
            {
              title: "操作",
              fixed: "right",
              width: 130,
              render: (_, s) => (
                <Button
                  type="link"
                  onClick={() => setEditor({ kind: "scenario", id: s.id })}
                >
                  编辑评估
                </Button>
              ),
            },
          ]}
        />
      </Card>
    </>
  );
}
function Treatment() {
  const { ws, review, setEditor } = useWorkbench();
  return (
    <>
      <PageTitle
        title="风险处置与需求"
        description="把风险分析转化为处置决策、网络安全目标和可验证的工程需求。"
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            disabled={!ws.scenarios.length}
            onClick={() => setEditor({ kind: "requirement" })}
          >
            新增安全需求
          </Button>
        }
      />
      <Tabs
        items={[
          {
            key: "decisions",
            label: `处置决策（${ws.decisions.length} / ${ws.scenarios.length}）`,
            children: (
              <Card>
                <Table
                  rowKey="id"
                  pagination={compactPage}
                  dataSource={ws.scenarios}
                  scroll={{ x: 950 }}
                  columns={[
                    numberColumn(ws, "scenario"),
                    { title: "风险场景", dataIndex: "title", width: 240 },
                    {
                      title: "初始风险",
                      width: 100,
                      render: (_, s) => (
                        <Risk
                          level={
                            review.evaluations.find(
                              (e) => e.scenario_id === s.id,
                            )?.level
                          }
                          provisional={!ws.method_confirmed}
                        />
                      ),
                    },
                    {
                      title: "处置策略 / 责任人",
                      width: 170,
                      render: (_, s) => {
                        const d = ws.decisions.find(
                          (d) => d.scenario_id === s.id,
                        );
                        return d ? (
                          <Space orientation="vertical" size={2}>
                            <Text>{treatments[d.treatment]}</Text>
                            <Text type="secondary">{d.owner}</Text>
                          </Space>
                        ) : (
                          <Tag>待决策</Tag>
                        );
                      },
                    },
                    {
                      title: "安全目标或声明",
                      render: (_, s) => {
                        const d = ws.decisions.find(
                          (d) => d.scenario_id === s.id,
                        );
                        return (
                          (d?.treatment === "reduce" ? d?.goal : d?.claim) ||
                          "—"
                        );
                      },
                    },
                    {
                      title: "残余风险",
                      width: 110,
                      render: (_, s) => (
                        <Risk
                          level={
                            review.evaluations.find(
                              (e) => e.scenario_id === s.id,
                            )?.residual
                          }
                        />
                      ),
                    },
                    {
                      title: "操作",
                      fixed: "right",
                      width: 145,
                      render: (_, s) => {
                        const ev = review.evaluations.find(
                          (e) => e.scenario_id === s.id,
                        );
                        return (
                          <Tooltip
                            title={
                              ev.level == null
                                ? "请先完成场景评估"
                                : !ws.method_confirmed
                                  ? "请先确认评估方法"
                                  : ""
                            }
                          >
                            <Button
                              type="link"
                              disabled={
                                ev.level == null || !ws.method_confirmed
                              }
                              onClick={() =>
                                setEditor({ kind: "decision", id: s.id })
                              }
                            >
                              {ev.decision_stale
                                ? "重新确认决策"
                                : "记录 / 编辑决策"}
                            </Button>
                          </Tooltip>
                        );
                      },
                    },
                  ]}
                />
              </Card>
            ),
          },
          {
            key: "requirements",
            label: `安全需求（${ws.requirements.length}）`,
            children: <Requirements />,
          },
        ]}
      />
    </>
  );
}
function Requirements() {
  const { ws, review, setEditor } = useWorkbench();
  const [q, setQ] = useState("");
  return (
    <Card>
      <div className="table-toolbar">
        <Text strong>安全需求与验证状态</Text>
        <Input.Search
          placeholder="搜索需求或分配对象"
          aria-label="搜索安全需求"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          style={{ width: 280 }}
          allowClear
        />
      </div>
      <Table
        rowKey="id"
        dataSource={ws.requirements.filter((r) =>
          match(
            [r.statement, r.allocation, recordCode(ws, "requirement", r.id)],
            q,
          ),
        )}
        pagination={compactPage}
        scroll={{ x: 1150 }}
        columns={[
          numberColumn(ws, "requirement"),
          {
            title: "需求陈述",
            dataIndex: "statement",
            width: 310,
            render: (v, r) => (
              <Space orientation="vertical" size={3}>
                <Button
                  type="link"
                  className="text-button"
                  onClick={() => setEditor({ kind: "requirement", id: r.id })}
                >
                  {v}
                </Button>
                <Text className="cell-description" type="secondary">
                  {ws.scenarios.find((s) => s.id === r.scenario_id)?.title}
                </Text>
              </Space>
            ),
          },
          {
            title: "分配对象",
            dataIndex: "allocation",
            width: 180,
            render: (v) => v || <Tag color="warning">待分配</Tag>,
          },
          {
            title: "验证准则",
            dataIndex: "verification",
            width: 250,
            render: (v) => v || <Tag color="warning">待补充</Tag>,
          },
          {
            title: "验证状态",
            width: 160,
            render: (_, r) => <Status value={evidenceStatus(r, ws, review)} />,
          },
          {
            title: "操作",
            fixed: "right",
            width: 185,
            render: (_, r) => (
              <Space>
                <Button
                  type="link"
                  onClick={() => setEditor({ kind: "requirement", id: r.id })}
                >
                  编辑
                </Button>
                <Button
                  type="link"
                  disabled={!r.allocation || !r.verification}
                  onClick={() => setEditor({ kind: "evidence", id: r.id })}
                >
                  记录验证
                </Button>
              </Space>
            ),
          },
        ]}
      />
    </Card>
  );
}
function Verification() {
  const { ws, review, setEditor, openIssue, message } = useWorkbench();
  const [stage, setStage] = useState("all"),
    [busy, setBusy] = useState(""),
    [lastExport, setLastExport] = useState(null),
    [baseline, setBaseline] = useState(""),
    [journal, setJournal] = useState([]),
    [excel, setExcel] = useState(false);
  useEffect(() => {
    let active = true;
    workspaceEvents(ws.id)
      .then((v) => active && setJournal(v.entries))
      .catch((e) => active && message.error(e.message));
    request("/health")
      .then((v) => active && setExcel(v.excel_export))
      .catch((e) => active && message.error(e.message));
    return () => {
      active = false;
    };
  }, [ws.id, ws.revision]);
  const download = async (format) => {
    setBusy(format);
    try {
      const result = await exportFile(ws, format, baseline);
      setLastExport(result);
      message.success("报告已生成并开始下载");
    } catch (e) {
      message.error(e.message);
    } finally {
      setBusy("");
    }
  };
  return (
    <>
      <PageTitle
        title="验证与报告"
        description="处理待办、核对验证证据，选择工作版本或冻结基线生成报告。"
        extra={
          <Tooltip title={review.ready ? "" : "仍有未完成事项，请先处理待办"}>
            <Button
              type="primary"
              disabled={!review.ready}
              onClick={() => setEditor({ kind: "baseline" })}
            >
              建立评审基线
            </Button>
          </Tooltip>
        }
      />
      <Tabs
        items={[
          {
            key: "tasks",
            label: (
              <Space>
                待办事项
                <Badge
                  count={review.issues.length}
                  showZero
                  color={review.ready ? "#52c41a" : "#faad14"}
                />
              </Space>
            ),
            children: (
              <Card>
                <div className="table-toolbar">
                  <Select
                    aria-label="按工作区筛选待办"
                    value={stage}
                    onChange={setStage}
                    options={[
                      { value: "all", label: "全部工作区" },
                      ...Object.entries(stageNames).map(([value, label]) => ({
                        value,
                        label,
                      })),
                    ]}
                    style={{ width: 180 }}
                  />
                  <Text type="secondary">
                    记录保存后自动检查，无需手动标记完成
                  </Text>
                </div>
                <Table
                  rowKey={(r, i) => r.message + r.reference + i}
                  dataSource={review.issues.filter(
                    (i) => stage === "all" || i.stage === stage,
                  )}
                  pagination={compactPage}
                  columns={[
                    {
                      title: "工作区",
                      dataIndex: "stage",
                      width: 130,
                      render: (v) => <Tag>{stageNames[v]}</Tag>,
                    },
                    {
                      title: "待完成事项",
                      dataIndex: "message",
                      render: (value) =>
                        Object.entries(properties).reduce(
                          (text, [key, label]) => text.replaceAll(key, label),
                          value,
                        ),
                    },
                    {
                      title: "关联记录",
                      dataIndex: "reference",
                      render: (id) =>
                        ws.assets.find((a) => a.id === id)?.name ||
                        ws.scenarios.find((s) => s.id === id)?.title ||
                        ws.requirements.find((r) => r.id === id)?.statement ||
                        "项目级检查",
                      ellipsis: true,
                    },
                    {
                      title: "操作",
                      width: 100,
                      render: (_, i) => (
                        <Button type="link" onClick={() => openIssue(i)}>
                          去处理
                        </Button>
                      ),
                    },
                  ]}
                />
                {review.ready && (
                  <Alert
                    type="success"
                    showIcon
                    title="当前完整性检查通过，可进行项目评审。"
                  />
                )}
              </Card>
            ),
          },
          {
            key: "evidence",
            label: `需求验证（${ws.evidence.length} 条证据）`,
            children: (
              <>
                <Requirements />
                <Card className="section-card" title="验证记录">
                  <Table
                    rowKey="id"
                    dataSource={[...ws.evidence].reverse()}
                    pagination={compactPage}
                    columns={[
                      numberColumn(ws, "evidence"),
                      { title: "证据引用", dataIndex: "reference" },
                      {
                        title: "验证结果",
                        dataIndex: "result",
                        render: (v) => <Status value={v} />,
                      },
                      { title: "结果摘要", dataIndex: "summary" },
                      { title: "复核人", dataIndex: "reviewer", width: 130 },
                      {
                        title: "记录时间",
                        dataIndex: "recorded_at",
                        width: 180,
                        render: date,
                      },
                    ]}
                  />
                </Card>
              </>
            ),
          },
          {
            key: "reports",
            label: "评审与报告",
            children: (
              <>
                <Row gutter={[16, 16]}>
                  <Col xs={24} xl={14}>
                    <Card title="生成报告">
                      <Space
                        orientation="vertical"
                        size={20}
                        style={{ width: "100%" }}
                      >
                        <Descriptions
                          size="small"
                          column={2}
                          items={[
                            {
                              key: "state",
                              label: "当前检查",
                              children: review.ready ? (
                                <Tag color="success">检查通过</Tag>
                              ) : (
                                <Tag color="warning">
                                  {review.issues.length} 项待完成
                                </Tag>
                              ),
                            },
                            {
                              key: "revision",
                              label: "工作版本",
                              children: "修订 " + ws.revision,
                            },
                          ]}
                        />
                        {!review.ready && (
                          <Alert
                            type="info"
                            showIcon
                            title="可以导出草稿报告。未评估项将如实留空，建立评审基线前需完成待办。"
                          />
                        )}
                        <div>
                          <Text strong>选择导出版本</Text>
                          <Select
                            aria-label="选择导出版本"
                            value={baseline}
                            onChange={setBaseline}
                            options={[
                              {
                                value: "",
                                label: `当前工作版本 · R${ws.revision}`,
                              },
                              ...ws.baselines.map((b) => ({
                                value: b.id,
                                label: `${b.name} · R${b.revision}`,
                              })),
                            ]}
                            style={{ width: "100%", marginTop: 10 }}
                          />
                        </div>
                        <Space wrap>
                          {excel && (
                            <Button
                              type="primary"
                              icon={<ExportOutlined />}
                              loading={busy === "xlsx"}
                              disabled={!!busy && busy !== "xlsx"}
                              onClick={() => download("xlsx")}
                            >
                              导出 Excel 报告
                            </Button>
                          )}
                          <Button
                            loading={busy === "html"}
                            disabled={!!busy && busy !== "html"}
                            onClick={() => download("html")}
                          >
                            可打印报告
                          </Button>
                          <Button
                            loading={busy === "json"}
                            disabled={!!busy && busy !== "json"}
                            onClick={() => download("json")}
                          >
                            导出工作产品
                          </Button>
                        </Space>
                        {lastExport && (
                          <Alert
                            type="success"
                            showIcon
                            title="文件生成成功"
                            description={`${lastExport.name} · ${(lastExport.size / 1024).toFixed(1)} KB · ${date(lastExport.at)}`}
                          />
                        )}
                      </Space>
                    </Card>
                  </Col>
                  <Col xs={24} xl={10}>
                    <Card title="评审基线">
                      <Table
                        size="small"
                        pagination={false}
                        rowKey="id"
                        dataSource={ws.baselines}
                        columns={[
                          { title: "基线名称", dataIndex: "name" },
                          {
                            title: "版本",
                            dataIndex: "revision",
                            render: (v) => "R" + v,
                          },
                          { title: "评审人", dataIndex: "reviewer" },
                          {
                            title: "操作",
                            render: (_, b) => (
                              <Button
                                type="link"
                                size="small"
                                onClick={() => setBaseline(b.id)}
                              >
                                选择
                              </Button>
                            ),
                          },
                        ]}
                      />
                    </Card>
                  </Col>
                </Row>
                <Card title="操作记录" className="section-card">
                  <Table
                    size="small"
                    rowKey="revision"
                    dataSource={journal}
                    pagination={compactPage}
                    columns={[
                      {
                        title: "修订",
                        dataIndex: "revision",
                        width: 90,
                        render: (v) => "R" + v,
                      },
                      {
                        title: "操作",
                        dataIndex: "action",
                        render: (v) => actions[v] || v,
                      },
                      { title: "时间", dataIndex: "at", render: date },
                    ]}
                  />
                </Card>
              </>
            ),
          },
        ]}
      />
    </>
  );
}
function Library() {
  const { catalog, setReference, message } = useWorkbench();
  const [category, setCategory] = useState("systems"),
    [q, setQ] = useState(""),
    [search, setSearch] = useState(""),
    [page, setPage] = useState(1),
    [result, setResult] = useState({ items: [], total: 0 }),
    [busy, setBusy] = useState(false);
  useEffect(() => {
    if (!catalog.categories.length) return;
    let active = true;
    setBusy(true);
    request(
      `/catalog/${category}/entries?q=${encodeURIComponent(search)}&offset=${(page - 1) * 15}&limit=15`,
    )
      .then((v) => {
        if (active) setResult(v);
      })
      .catch((e) => active && message.error(e.message))
      .finally(() => active && setBusy(false));
    return () => {
      active = false;
    };
  }, [category, search, page, catalog]);
  if (!catalog.categories.length)
    return (
      <>
        <PageTitle
          title="参考资料"
          description="仅收录可用于当前产品的自有或已授权资料。"
        />
        <Card>
          <Empty description="尚未配置参考资料">
            <Text type="secondary">
              历史参考资料已撤下。已有项目的引用可追溯到撤下标记；新增分析可手工录入，法规条款仍可在法规数据库中勾选。
            </Text>
          </Empty>
        </Card>
      </>
    );
  return (
    <>
      <PageTitle
        title="参考资料"
        description="检索本地参考资料，按项目实际选择和调整，保留引用来源。"
      />
      <Card>
        <div className="library-body">
          <Menu
            className="library-menu"
            selectedKeys={[category]}
            items={catalog.categories.map((c) => ({
              key: c.key,
              label: (
                <span className="category-item">
                  {c.label}
                  <span>{c.count.toLocaleString()}</span>
                </span>
              ),
            }))}
            onClick={({ key }) => {
              setCategory(key);
              setPage(1);
              setQ("");
              setSearch("");
            }}
          />
          <div className="library-results">
            <div className="table-toolbar">
              <Text strong>
                {catalog.categories.find((c) => c.key === category)?.label}
              </Text>
              <Input.Search
                placeholder="搜索名称、描述或关键词"
                aria-label="搜索知识条目"
                value={q}
                allowClear
                onChange={(e) => {
                  setQ(e.target.value);
                  if (!e.target.value) {
                    setSearch("");
                    setPage(1);
                  }
                }}
                onSearch={(v) => {
                  setSearch(v);
                  setPage(1);
                }}
                style={{ width: 330 }}
              />
            </div>
            {category === "relationships" && (
              <Alert
                type="warning"
                showIcon
                className="section-card"
                title={`${catalog.gaps.relationships} 条关联存在缺失的控制措施引用，预览时会明确列出。`}
              />
            )}
            <Table
              size="middle"
              loading={busy}
              rowKey="key"
              dataSource={result.items}
              pagination={{
                current: page,
                pageSize: 15,
                total: result.total,
                showSizeChanger: false,
                showTotal: (n) => `共 ${n} 条`,
                onChange: setPage,
              }}
              columns={[
                {
                  title: "名称",
                  dataIndex: "title",
                  width: "36%",
                  render: (v, r) => (
                    <Button
                      type="link"
                      className="text-button"
                      onClick={() => setReference(r.key)}
                    >
                      {v}
                    </Button>
                  ),
                },
                {
                  title: "描述",
                  dataIndex: "description",
                  render: (v) => (
                    <Paragraph
                      className="library-excerpt"
                      ellipsis={{ rows: 2, expandable: false }}
                    >
                      {v || "查看结构化内容"}
                    </Paragraph>
                  ),
                },
                {
                  title: "操作",
                  width: 100,
                  render: (_, r) => (
                    <Button type="link" onClick={() => setReference(r.key)}>
                      查看详情
                    </Button>
                  ),
                },
              ]}
            />
          </div>
        </div>
      </Card>
    </>
  );
}

function numberColumn(ws, kind, parentId) {
  return {
    title: "编号",
    key: "number",
    width: 140,
    render: (_, row) => (
      <Text code style={{ whiteSpace: "nowrap" }}>
        {recordCode(
          ws,
          kind,
          parentId ? pathNumberKey(parentId, row.id) : row.id,
        )}
      </Text>
    ),
  };
}
