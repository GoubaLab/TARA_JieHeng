import React, { useEffect, useMemo, useRef, useState } from "react";
import {
  Alert,
  Button,
  Card,
  Checkbox,
  Descriptions,
  Drawer,
  Empty,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from "antd";
import { ReloadOutlined } from "@ant-design/icons";
import { request, properties } from "./api.js";
import {
  existingTemplateCodes,
  MAX_ATTACK_PATHS,
} from "./attack-path-templates.js";

const { Text, Paragraph } = Typography;
function TextList({ items, ordered = false }) {
  const List = ordered ? "ol" : "ul";
  return (
    <List style={{ paddingLeft: 22, margin: 0, lineHeight: 1.8 }}>
      {(items || []).map((item, i) => (
        <li key={i}>{item}</li>
      ))}
    </List>
  );
}

export default function AttackPathTemplatePicker({
  open,
  onClose,
  workspace,
  assetId,
  property,
  existingPaths = [],
  onSelect,
}) {
  const [catalog, setCatalog] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [category, setCategory] = useState("");
  const [recommendedOnly, setRecommendedOnly] = useState(false);
  const [selected, setSelected] = useState([]);
  const [detailID, setDetailID] = useState(null);
  const [refreshToken, setRefreshToken] = useState(0);
  const generation = useRef(0);
  const items = catalog?.items || [];
  const codes = existingTemplateCodes(existingPaths);
  const remaining = Math.max(0, MAX_ATTACK_PATHS - existingPaths.length);
  const selectedRows = items.filter(
    (t) => selected.includes(t.id) && !codes.has(t.source_code),
  );
  const detail = items.find((t) => t.id === detailID);
  const asset = workspace?.assets?.find((a) => a.id === assetId);
  const canEdit = workspace?.access_role !== "viewer";

  useEffect(() => {
    setSelected([]);
    setDetailID(null);
    setQuery("");
    setCategory("");
    setRecommendedOnly(false);
  }, [open, workspace?.id]);
  useEffect(() => {
    const current = ++generation.current;
    setCatalog(null);
    setSelected([]);
    setDetailID(null);
    setError("");
    if (!open || !workspace?.id) {
      setLoading(false);
      return;
    }
    const controller = new AbortController();
    const params = new URLSearchParams();
    if (assetId) params.set("asset_id", assetId);
    if (property) params.set("property", property);
    setLoading(true);
    request(
      `/workspaces/${encodeURIComponent(workspace.id)}/attack-path-templates${params.size ? `?${params}` : ""}`,
      { signal: controller.signal },
    )
      .then((value) => {
        if (current !== generation.current) return;
        if (!Array.isArray(value.items))
          throw Error("模板列表格式无效，请重新加载");
        setCatalog(value);
      })
      .catch((e) => {
        if (current === generation.current && !controller.signal.aborted)
          setError(e.message);
      })
      .finally(() => {
        if (current === generation.current) setLoading(false);
      });
    return () => {
      generation.current++;
      controller.abort();
    };
  }, [
    open,
    workspace?.id,
    workspace?.revision,
    assetId,
    property,
    refreshToken,
  ]);

  const categories = useMemo(
    () => [...new Set(items.map((t) => t.category))],
    [items],
  );
  const filtered = items
    .filter((t) => {
      const haystack = [
        t.name,
        t.summary,
        t.category,
        t.entry_point,
        t.actor,
        t.objective,
        ...(t.tags || []),
        ...(t.properties || []).map((p) => properties[p] || p),
      ]
        .join(" ")
        .toLocaleLowerCase();
      return (
        (!category || t.category === category) &&
        (!recommendedOnly || t.match?.score > 0) &&
        haystack.includes(query.trim().toLocaleLowerCase())
      );
    })
    .sort(
      (a, b) =>
        (b.match?.score || 0) - (a.match?.score || 0) ||
        a.name.localeCompare(b.name, "zh-CN"),
    );

  const select = async () => {
    if (!selectedRows.length || loading || !canEdit) return;
    if (selectedRows.length > remaining) {
      setError(`当前最多还能添加 ${remaining} 条路径，请减少选择`);
      return;
    }
    try {
      await onSelect(selectedRows);
    } catch (e) {
      setError(e.message || "添加路径未完成");
    }
  };
  const addDetailToSelection = () => {
    if (
      !detail ||
      codes.has(detail.source_code) ||
      selected.includes(detail.id)
    )
      return;
    if (selectedRows.length >= remaining) {
      setError(`当前最多还能添加 ${remaining} 条路径`);
      return;
    }
    setSelected((value) => [...value, detail.id]);
    setDetailID(null);
  };

  return (
    <>
      <Modal
        open={open}
        title="选用攻击路径模板"
        width={1100}
        onCancel={onClose}
        onOk={select}
        okText={`加入当前场景${selectedRows.length ? `（${selectedRows.length}）` : ""}`}
        cancelText="取消"
        okButtonProps={{
          disabled:
            loading ||
            !canEdit ||
            !selectedRows.length ||
            selectedRows.length > remaining,
        }}
        destroyOnHidden
        styles={{ body: { maxHeight: "70vh", overflowY: "auto" } }}
      >
        <Space orientation="vertical" size={16} style={{ width: "100%" }}>
          <Alert
            type="info"
            title="按项目条件选择，再补充实际攻击链"
            description="模板会加入当前场景的编辑表单，保存场景后生效。入口、前提和控制实施情况仍需核实，影响与可行性不会自动评分。"
          />
          <Space wrap>
            <Text type="secondary">当前资产：{asset?.name || "尚未选择"}</Text>
            {property && <Tag>{properties[property] || property}</Tag>}
            <Text type="secondary">
              已有 {existingPaths.length} 条，可再添加 {remaining} 条
            </Text>
          </Space>
          {error && (
            <Alert
              type="error"
              title={error}
              closable
              onClose={() => setError("")}
            />
          )}
          {!remaining && (
            <Alert
              type="warning"
              title="当前场景已达到 100 条路径上限，请先整理已有路径。"
            />
          )}
          <Space wrap>
            <Input.Search
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="搜索入口、协议、攻击主体或目标"
              allowClear
              style={{ width: 310, maxWidth: "100%" }}
              aria-label="搜索攻击路径模板"
            />
            <Select
              value={category}
              onChange={setCategory}
              options={[
                { value: "", label: "全部分类" },
                ...categories.map((value) => ({ value, label: value })),
              ]}
              style={{ width: 170 }}
              aria-label="攻击路径模板分类"
            />
            <Checkbox
              checked={recommendedOnly}
              onChange={(e) => setRecommendedOnly(e.target.checked)}
            >
              只看有匹配线索
            </Checkbox>
            <Button
              icon={<ReloadOutlined />}
              loading={loading}
              onClick={() => setRefreshToken((v) => v + 1)}
            >
              刷新模板
            </Button>
          </Space>
          <Table
            rowKey="id"
            dataSource={filtered}
            loading={loading}
            size="small"
            tableLayout="fixed"
            scroll={{ x: 700 }}
            locale={{
              emptyText: (
                <Empty
                  description={
                    error
                      ? "模板尚未加载，请重试"
                      : "没有符合筛选条件的模板，可调整搜索或查看全部分类"
                  }
                />
              ),
            }}
            rowSelection={{
              columnWidth: "5%",
              selectedRowKeys: selectedRows.map((t) => t.id),
              preserveSelectedRowKeys: true,
              getCheckboxProps: (t) => ({
                disabled:
                  !canEdit ||
                  codes.has(t.source_code) ||
                  (selectedRows.length >= remaining &&
                    !selected.includes(t.id)),
              }),
              onChange: (keys) => {
                const available = new Set(
                  items
                    .filter((t) => !codes.has(t.source_code))
                    .map((t) => t.id),
                );
                const next = keys.filter((id) => available.has(id));
                if (next.length > remaining) {
                  setError(`当前最多还能添加 ${remaining} 条路径，请分批选择`);
                  return;
                }
                setSelected(next);
              },
            }}
            columns={[
              {
                title: "路径模板",
                width: "32%",
                render: (_, t) => (
                  <>
                    <Button
                      type="link"
                      style={{
                        whiteSpace: "normal",
                        height: "auto",
                        textAlign: "left",
                        padding: 0,
                        display: "block",
                        width: "100%",
                        overflowWrap: "anywhere",
                      }}
                      onClick={() => setDetailID(t.id)}
                    >
                      {t.name}
                    </Button>
                    <Paragraph
                      type="secondary"
                      style={{
                        margin: "4px 0 0",
                        fontSize: 12,
                        overflowWrap: "anywhere",
                      }}
                    >
                      {t.summary}
                    </Paragraph>
                  </>
                ),
              },
              {
                title: "分类 / 安全属性",
                width: "18%",
                render: (_, t) => (
                  <>
                    <Text>{t.category}</Text>
                    <div style={{ marginTop: 6 }}>
                      {(t.properties || []).map((p) => (
                        <Tag key={p} style={{ marginBottom: 4 }}>
                          {properties[p] || p}
                        </Tag>
                      ))}
                    </div>
                  </>
                ),
              },
              {
                title: "入口与匹配线索",
                width: "33%",
                render: (_, t) => (
                  <>
                    <Text>{t.entry_point}</Text>
                    <div style={{ marginTop: 6 }}>
                      {t.match?.score > 0 ? (
                        <>
                          <Tag color="blue">有匹配线索</Tag>
                          <Text type="secondary" style={{ fontSize: 12 }}>
                            {(t.match.reasons || []).join("；")}
                          </Text>
                        </>
                      ) : (
                        <Text type="secondary">需自行核对适用条件</Text>
                      )}
                    </div>
                  </>
                ),
              },
              {
                title: "操作",
                width: "12%",
                render: (_, t) => (
                  <Space orientation="vertical" size={4}>
                    {codes.has(t.source_code) && <Tag>已添加</Tag>}
                    <Button
                      type="link"
                      size="small"
                      style={{
                        paddingInline: 0,
                        whiteSpace: "normal",
                        height: "auto",
                      }}
                      onClick={() => setDetailID(t.id)}
                    >
                      查看详情
                    </Button>
                  </Space>
                ),
              },
            ]}
            pagination={{
              pageSize: 6,
              showSizeChanger: false,
              showTotal: (n) =>
                `${n} 个模板 · 已选择 ${selectedRows.length} 个`,
            }}
          />
          <Text type="secondary">
            模板版本：{catalog?.version || "—"}
            。匹配线索仅用于排序，未匹配模板仍可在核对条件后选用。
          </Text>
        </Space>
      </Modal>
      <Drawer
        open={open && !!detail}
        title="攻击路径模板详情"
        width={700}
        onClose={() => setDetailID(null)}
        destroyOnHidden
        footer={
          <Space>
            <Button onClick={() => setDetailID(null)}>返回列表</Button>
            <Button
              type="primary"
              disabled={
                !canEdit ||
                !detail ||
                codes.has(detail?.source_code) ||
                selected.includes(detail?.id) ||
                selectedRows.length >= remaining
              }
              onClick={addDetailToSelection}
            >
              {detail && codes.has(detail.source_code)
                ? "当前场景已添加"
                : selected.includes(detail?.id)
                  ? "已加入选择"
                  : "加入选择"}
            </Button>
          </Space>
        }
      >
        {detail && (
          <Space orientation="vertical" size={16} style={{ width: "100%" }}>
            <Typography.Title level={4} style={{ margin: 0 }}>
              {detail.name}
            </Typography.Title>
            <Paragraph style={{ margin: 0 }}>{detail.summary}</Paragraph>
            <Descriptions
              column={1}
              items={[
                { key: "entry", label: "入口", children: detail.entry_point },
                { key: "actor", label: "攻击主体", children: detail.actor },
                {
                  key: "objective",
                  label: "目标与可能后果",
                  children: detail.objective,
                },
                {
                  key: "source",
                  label: "模板来源",
                  children: detail.source_code,
                },
              ]}
            />
            {!!detail.match?.reasons?.length && (
              <Alert
                type="info"
                title="匹配线索"
                description={<TextList items={detail.match.reasons} />}
              />
            )}
            <Card size="small" title="适用前提">
              <TextList items={detail.prerequisites} />
            </Card>
            <Card size="small" title="攻击步骤">
              <TextList items={detail.steps} ordered />
            </Card>
            <Card size="small" title="待核对的控制">
              <TextList items={detail.controls_to_check} />
            </Card>
            <Alert
              type="warning"
              title="这是分析起点，未认定入口可达、漏洞存在或控制缺失。请改写为项目实际路径后再评估。"
            />
          </Space>
        )}
      </Drawer>
    </>
  );
}
