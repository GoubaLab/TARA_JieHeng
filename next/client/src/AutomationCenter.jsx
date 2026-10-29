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
  Input,
  List,
  Modal,
  Row,
  Select,
  Space,
  Statistic,
  Table,
  Tag,
  Typography,
} from "antd";
import { ReloadOutlined } from "@ant-design/icons";
import { PageTitle, useWorkbench } from "./Workbench.jsx";
import { useAuth } from "./AuthGate.jsx";
import { navigate, properties } from "./api.js";
import { mutateResource, useResource } from "./platform-ui.js";

const { Text, Paragraph } = Typography;
const subjectLabels = {
  flow: "数据流",
  node: "实体 / 处理过程",
  child: "内部存储",
};
const strideLabels = {
  Spoofing: "身份冒用",
  Tampering: "篡改",
  Repudiation: "抵赖",
  "Information Disclosure": "信息泄露",
  "Denial of Service": "拒绝服务",
  "Elevation of Privilege": "权限提升",
};
const factLabels = {
  medium: "介质",
  technology: "协议",
  family: "操作系统类型",
  name: "名称",
  version: "版本",
  architecture: "平台架构",
  notes: "备注",
  source_zone: "发送端信任域",
  target_zone: "接收端信任域",
  source_external: "发送端为外部主体",
  target_external: "接收端为外部主体",
};
const valueLabels = {
  wired: "有线",
  wireless: "无线",
  logical: "逻辑连接",
  unspecified: "未明确",
  unknown: "未知",
  store: "存储",
  data_store: "数据存储",
  wifi: "Wi-Fi",
  bluetooth: "蓝牙",
  ble: "低功耗蓝牙",
  ethernet: "以太网",
  linux: "Linux",
  rtos: "RTOS",
  qnx: "QNX",
  windows: "Windows",
  android: "Android",
  ios: "iOS",
  macos: "macOS",
};
function candidateState(candidate) {
  if (candidate.already_covered) return ["已有关联分析", "success"];
  if (candidate.selectable) return ["待审阅", "blue"];
  return ["需完善资产", "warning"];
}
function BulletList({ items }) {
  return (
    <List
      size="small"
      dataSource={items || []}
      renderItem={(item) => <List.Item>{item}</List.Item>}
    />
  );
}

export default function AutomationCenter() {
  const { ws, reload } = useWorkbench();
  const { user } = useAuth();
  const { message } = App.useApp();
  const plan = useResource(
    `/workspaces/${encodeURIComponent(ws.id)}/automation-plan`,
  );
  const [selected, setSelected] = useState([]);
  const [filter, setFilter] = useState("all");
  const [query, setQuery] = useState("");
  const [detailID, setDetailID] = useState(null);
  const [adopting, setAdopting] = useState(false);
  const [owner, setOwner] = useState("");
  const [reviewed, setReviewed] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(null);
  const lastRevision = useRef(`${ws.id}:${ws.revision}`);
  const attempt = useRef(null);
  const canAdopt = (ws.access_role || user?.role) !== "viewer";
  const candidates = plan.data?.candidates || [];
  const summary = plan.data?.summary || {};
  const stale = plan.data && plan.data.base_revision !== ws.revision;

  useEffect(() => {
    const current = `${ws.id}:${ws.revision}`;
    if (current !== lastRevision.current) {
      lastRevision.current = current;
      plan.refresh();
    }
  }, [ws.id, ws.revision, plan.refresh]);
  useEffect(() => {
    const available = new Set(
      (plan.data?.candidates || [])
        .filter((c) => c.selectable)
        .map((c) => c.id),
    );
    setSelected((current) => current.filter((id) => available.has(id)));
    setReviewed(false);
    setAdopting(false);
  }, [plan.data?.plan_hash, ws.id]);
  useEffect(() => {
    setSelected([]);
    setDetailID(null);
    setSuccess(null);
    setError("");
  }, [ws.id]);

  const names = useMemo(
    () =>
      new Map(
        [
          ...(ws.architecture?.nodes || []),
          ...(ws.architecture?.flows || []),
          ...(ws.assets || []),
        ].map((row) => [row.id, row.name]),
      ),
    [ws.architecture, ws.assets],
  );
  const readableFact = (value) => {
    if (value == null || value === "") return "未登记";
    if (typeof value === "boolean") return value ? "是" : "否";
    if (Array.isArray(value))
      return value.map(readableFact).join("、") || "未登记";
    if (typeof value === "object")
      return Object.entries(value)
        .filter(([, v]) => v !== "")
        .map(([key, v]) => `${factLabels[key] || key}：${readableFact(v)}`)
        .join("；");
    return (
      names.get(value) ||
      properties[value] ||
      valueLabels[value] ||
      String(value)
    );
  };
  const rows = candidates.filter((candidate) => {
    const matchesState =
      filter === "all" ||
      (filter === "selectable" && candidate.selectable) ||
      (filter === "covered" && candidate.already_covered) ||
      (filter === "assets" &&
        !candidate.selectable &&
        !candidate.already_covered);
    const text = [
      candidate.title,
      candidate.rule_id,
      candidate.subject?.name,
      properties[candidate.property],
      ...(candidate.asset_options || []).map((asset) => asset.name),
    ]
      .join(" ")
      .toLocaleLowerCase();
    return matchesState && text.includes(query.trim().toLocaleLowerCase());
  });
  const selectedRows = candidates.filter((candidate) =>
    selected.includes(candidate.id),
  );
  const detail = candidates.find((candidate) => candidate.id === detailID);
  const refresh = async () => {
    setError("");
    await reload({ quiet: true });
    await plan.refresh();
  };
  const openAdoption = () => {
    setOwner(ws.item?.owner || user?.name || "");
    setReviewed(false);
    setError("");
    attempt.current = null;
    setAdopting(true);
  };
  const adopt = async () => {
    if (!reviewed || !owner.trim() || !selected.length || stale || busy) return;
    const body = {
      plan_hash: plan.data.plan_hash,
      candidate_ids: [...selected].sort(),
      owner: owner.trim(),
      reviewed: true,
    };
    const fingerprint = JSON.stringify({
      body,
      revision: plan.data.base_revision,
    });
    if (attempt.current?.fingerprint !== fingerprint)
      attempt.current = { fingerprint, key: crypto.randomUUID() };
    setBusy(true);
    setError("");
    try {
      const result = await mutateResource(
        `/workspaces/${encodeURIComponent(ws.id)}/automation-adoptions`,
        body,
        plan.data.base_revision,
        "POST",
        attempt.current.key,
      );
      setSuccess({
        count: result.counts?.scenarios ?? selected.length,
        revision: result.revision,
        checkpoint: result.checkpoint_id || result.checkpoint?.id,
      });
      setSelected([]);
      setAdopting(false);
      attempt.current = null;
      await reload({ quiet: true });
      await plan.refresh();
      message.success("分析草案已加入项目，评分与验证待完成");
    } catch (e) {
      setError(e.message);
      if (e.status === 409 || e.status === 412) {
        setAdopting(false);
        setSelected([]);
        await reload({ quiet: true });
        await plan.refresh();
        message.warning("项目或分析计划已变化，请重新审阅候选后采用");
      }
    } finally {
      setBusy(false);
    }
  };
  const assetNames = (candidate) =>
    (candidate.asset_options || [])
      .filter((asset) => (candidate.asset_ids || []).includes(asset.id))
      .map((asset) => asset.name)
      .join("、");

  return (
    <>
      <PageTitle
        title="自动分析"
        description="从已保存的架构、操作系统和通信属性识别待分析项，审阅后补充损害、威胁路径和安全需求。"
        extra={
          <Space wrap>
            <Button
              icon={<ReloadOutlined />}
              onClick={refresh}
              loading={plan.loading}
              disabled={busy}
            >
              刷新分析
            </Button>
            <Button onClick={() => navigate("architecture", ws.id)}>
              完善架构与资产
            </Button>
            <Button
              type="primary"
              onClick={openAdoption}
              disabled={
                !canAdopt || !selected.length || !!stale || plan.loading || busy
              }
            >
              采用所选草案{selected.length ? `（${selected.length}）` : ""}
            </Button>
          </Space>
        }
      />
      {(error || plan.error) && (
        <Alert
          className="section-card"
          type="error"
          showIcon
          title={error || plan.error}
        />
      )}
      {success && (
        <Alert
          className="section-card"
          type="success"
          showIcon
          closable
          onClose={() => setSuccess(null)}
          title={`已加入 ${success.count} 个威胁分析草案${success.revision != null ? `，项目修订 ${success.revision}` : ""}`}
          description="已关联资产、损害场景和安全需求，并保存采用前还原点。下一步：评估影响与可行性，完善安全目标。"
          action={
            <Button onClick={() => navigate("assessment", ws.id)}>
              继续风险评估
            </Button>
          }
        />
      )}
      <Alert
        className="section-card"
        type="info"
        showIcon
        title="基于架构与资产的规则分析"
        description="根据模型事实匹配候选，核对适用前提后可加入项目。已有分析保留；新增内容的评分、处置和验证待完成。"
      />
      {stale && (
        <Alert
          className="section-card"
          type="warning"
          showIcon
          title="项目已更新，请刷新分析后再选择采用"
        />
      )}
      {!canAdopt && (
        <Alert
          className="section-card"
          type="info"
          title="当前为只读权限，可查看全部候选及其依据。"
        />
      )}
      <Row gutter={[16, 16]} className="section-card">
        {[
          ["匹配候选", summary.total ?? 0],
          ["可审阅采用", summary.selectable ?? 0],
          ["已有相同分析", summary.already_covered ?? 0],
          ["需完善资产关联", summary.missing_assets ?? 0],
        ].map(([title, value]) => (
          <Col xs={12} lg={6} key={title}>
            <Card size="small">
              <Statistic title={title} value={value} />
            </Card>
          </Col>
        ))}
      </Row>
      <Card
        title="候选分析"
        extra={<Text type="secondary">每次最多采用 50 项</Text>}
      >
        <Space wrap style={{ marginBottom: 16 }}>
          <Input.Search
            placeholder="搜索实体、数据流、资产或安全属性"
            allowClear
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            style={{ width: 320, maxWidth: "100%" }}
          />
          <Select
            aria-label="候选状态"
            value={filter}
            onChange={setFilter}
            style={{ width: 180 }}
            options={[
              { value: "all", label: "全部状态" },
              { value: "selectable", label: "待审阅" },
              { value: "covered", label: "已有关联分析" },
              { value: "assets", label: "需完善资产" },
            ]}
          />
          {!!selected.length && (
            <Button type="link" onClick={() => setSelected([])}>
              清空选择
            </Button>
          )}
        </Space>
        <Table
          rowKey="id"
          loading={plan.loading}
          dataSource={rows}
          scroll={{ x: 980 }}
          rowSelection={{
            selectedRowKeys: selected,
            preserveSelectedRowKeys: true,
            onChange: (keys) => {
              if (keys.length > 50)
                message.warning("每次最多采用 50 项，请分批审阅");
              setSelected(keys.slice(0, 50));
            },
            getCheckboxProps: (candidate) => ({
              disabled: !canAdopt || !candidate.selectable || !!stale || busy,
            }),
          }}
          columns={[
            {
              title: "分析对象与主题",
              dataIndex: "title",
              render: (title, candidate) => (
                <Space orientation="vertical" size={2}>
                  <Button
                    type="link"
                    style={{
                      padding: 0,
                      height: "auto",
                      whiteSpace: "normal",
                      textAlign: "left",
                    }}
                    onClick={() => setDetailID(candidate.id)}
                  >
                    {title}
                  </Button>
                  <Text type="secondary">
                    {subjectLabels[candidate.subject?.kind] || "分析对象"} ·{" "}
                    {candidate.subject?.name}
                  </Text>
                </Space>
              ),
            },
            {
              title: "安全属性",
              dataIndex: "property",
              width: 110,
              render: (value) => <Tag>{properties[value] || value}</Tag>,
            },
            {
              title: "关联资产",
              width: 230,
              render: (_, candidate) =>
                assetNames(candidate) || (
                  <Text type="warning">
                    {candidate.asset_options?.length
                      ? "关联待明确"
                      : "尚无适合的资产"}
                  </Text>
                ),
            },
            {
              title: "状态",
              width: 140,
              render: (_, candidate) => {
                const [label, color] = candidateState(candidate);
                return <Tag color={color}>{label}</Tag>;
              },
            },
            {
              title: "审阅",
              width: 100,
              render: (_, candidate) => (
                <Button type="link" onClick={() => setDetailID(candidate.id)}>
                  查看依据
                </Button>
              ),
            },
          ]}
          pagination={{
            defaultPageSize: 12,
            showSizeChanger: true,
            showTotal: (total) => `${total} 项候选`,
          }}
          locale={{
            emptyText: (
              <Empty
                description={
                  candidates.length
                    ? "没有符合筛选条件的候选"
                    : "暂无规则匹配项。可完善实体、操作系统、通信属性和资产，或手工建立风险场景。"
                }
              />
            ),
          }}
        />
      </Card>
      <Drawer
        title="候选依据与适用范围"
        open={!!detail}
        onClose={() => setDetailID(null)}
        size={760}
      >
        {detail && (
          <Space orientation="vertical" size={20} style={{ width: "100%" }}>
            <Typography.Title level={4} style={{ margin: 0 }}>
              {detail.title}
            </Typography.Title>
            <Descriptions
              size="small"
              column={1}
              items={[
                {
                  key: "subject",
                  label: "分析对象",
                  children: `${subjectLabels[detail.subject?.kind] || "对象"} · ${detail.subject?.name}`,
                },
                {
                  key: "property",
                  label: "安全属性",
                  children: properties[detail.property] || detail.property,
                },
                {
                  key: "stride",
                  label: "威胁类别",
                  children: strideLabels[detail.stride] || detail.stride,
                },
                {
                  key: "asset",
                  label: "拟关联资产",
                  children: assetNames(detail) || "无法确定，请先完善关联",
                },
                {
                  key: "rule",
                  label: "规则版本",
                  children: (
                    <Text type="secondary">
                      {detail.rule_id} · {detail.rule_version}
                    </Text>
                  ),
                },
              ]}
            />
            <Card title="匹配依据" size="small" style={{ width: "100%" }}>
              <List
                dataSource={detail.matched_conditions || []}
                renderItem={(condition) => (
                  <List.Item>
                    <Space orientation="vertical" size={3}>
                      <Text>{condition.explanation}</Text>
                      <Text type="secondary">
                        {readableFact(condition.value)}
                      </Text>
                    </Space>
                  </List.Item>
                )}
              />
            </Card>
            <Card title="适用前提" size="small" style={{ width: "100%" }}>
              <BulletList items={detail.prerequisites} />
            </Card>
            <Card title="控制检查项" size="small" style={{ width: "100%" }}>
              <BulletList items={detail.controls_to_check} />
            </Card>
            {!!detail.asset_options?.length && (
              <Card
                title="可关联的资产范围"
                size="small"
                style={{ width: "100%" }}
              >
                <List
                  dataSource={detail.asset_options}
                  renderItem={(asset) => (
                    <List.Item>
                      <Space orientation="vertical" size={2}>
                        <Text>
                          {asset.name}
                          {detail.asset_ids?.includes(asset.id)
                            ? "（已选资产）"
                            : ""}
                        </Text>
                        <Text type="secondary">
                          {(asset.properties || [])
                            .map((property) => properties[property] || property)
                            .join("、")}
                        </Text>
                      </Space>
                    </List.Item>
                  )}
                />
              </Card>
            )}
            {!!detail.existing_scenario_ids?.length && (
              <Alert
                type="success"
                title={`已有 ${detail.existing_scenario_ids.length} 项已覆盖分析，采用时跳过重复项`}
                action={
                  <Button
                    size="small"
                    onClick={() => navigate("assessment", ws.id)}
                  >
                    查看分析
                  </Button>
                }
              />
            )}
            {!!detail.related_scenario_ids?.length && (
              <Alert
                type="info"
                title={`另有 ${detail.related_scenario_ids.length} 项同类分析，请核对覆盖范围`}
                description="请按入口、攻击前提和影响对象核对场景覆盖范围。"
              />
            )}
            <Alert
              type="warning"
              showIcon
              title="复核要点"
              description={<BulletList items={detail.warnings} />}
            />
            <Button
              type="primary"
              disabled={
                !canAdopt ||
                !detail.selectable ||
                !!stale ||
                selected.includes(detail.id) ||
                selected.length >= 50
              }
              onClick={() => setSelected((current) => [...current, detail.id])}
            >
              {selected.includes(detail.id)
                ? "已加入待采用列表"
                : "加入待采用列表"}
            </Button>
          </Space>
        )}
      </Drawer>
      <Modal
        title="将所选分析加入项目"
        open={adopting}
        onCancel={() => !busy && setAdopting(false)}
        maskClosable={!busy}
        closable={!busy}
        confirmLoading={busy}
        okText="采用为待评估草案"
        cancelText="继续审阅"
        onOk={adopt}
        okButtonProps={{
          disabled:
            !owner.trim() ||
            !reviewed ||
            !selected.length ||
            !!stale ||
            plan.loading,
        }}
        cancelButtonProps={{ disabled: busy }}
        width={680}
      >
        <Space orientation="vertical" size={16} style={{ width: "100%" }}>
          <Alert
            type="info"
            showIcon
            title={`将补充 ${selectedRows.length} 个威胁分析草案及相关损害、路径和安全需求`}
            description="影响与可行性评分保持待确认。采用前自动保存还原点，现有人工分析继续保留。"
          />
          {error && <Alert type="error" showIcon title={error} />}
          <div style={{ maxHeight: 220, overflowY: "auto", width: "100%" }}>
            <List
              size="small"
              dataSource={selectedRows}
              renderItem={(candidate) => (
                <List.Item>
                  <Space orientation="vertical" size={2}>
                    <Text>{candidate.title}</Text>
                    <Text type="secondary">{assetNames(candidate)}</Text>
                  </Space>
                </List.Item>
              )}
            />
          </div>
          <div style={{ width: "100%" }}>
            <Paragraph style={{ marginBottom: 6 }}>
              <label htmlFor="automation-owner">分析责任人</label>
            </Paragraph>
            <Input
              id="automation-owner"
              value={owner}
              maxLength={200}
              onChange={(event) => setOwner(event.target.value)}
              placeholder="填写分析责任人"
              disabled={busy}
            />
          </div>
          <Checkbox
            checked={reviewed}
            onChange={(event) => setReviewed(event.target.checked)}
            disabled={busy}
          >
            已核对匹配依据、适用前提和控制检查项，确认作为待评估草案加入项目。
          </Checkbox>
        </Space>
      </Modal>
    </>
  );
}
