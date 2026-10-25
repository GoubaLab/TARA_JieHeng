import React, { useState, useEffect, useMemo } from "react";
import {
  Alert,
  Button,
  Card,
  Descriptions,
  Drawer,
  Empty,
  Input,
  Menu,
  Space,
  Table,
  Tag,
  Typography,
  Select,
  Statistic,
  Row,
  Col,
} from "antd";
import {
  CheckOutlined,
  LinkOutlined,
  ReloadOutlined,
  BookOutlined,
} from "@ant-design/icons";
import { useWorkbench, PageTitle } from "./Workbench.jsx";
import { request, apiRoot } from "./api.js";
import "./regulations.css";
const { Text, Paragraph } = Typography;
const coverageLabels = {
  metadata_only: ["正文待补", "warning"],
  partial: ["尚待完整核验", "warning"],
  complete_for_edition: ["该版本正文已收录", "success"],
};
function Coverage({ document }) {
  const value =
    typeof document?.coverage === "string"
      ? document.coverage
      : document?.coverage?.status;
  const [label, color] = coverageLabels[value] || ["待核对", "default"];
  return <Tag color={color}>{label}</Tag>;
}
function sourceLink(url, label = "官方来源") {
  return url ? (
    <a href={url} target="_blank" rel="noreferrer noopener">
      <LinkOutlined /> {label}
    </a>
  ) : (
    <Text type="secondary">来源待补</Text>
  );
}
function sourcePdfs(doc) {
  return (doc?.artifacts || []).filter((a) => a.file?.toLowerCase().endsWith(".pdf"));
}
function localSource(doc, row) {
  const pdfs = sourcePdfs(doc);
  if (!pdfs.length) return null;
  const found = row?.source_file
    ? pdfs.findIndex((a) => a.file === row.source_file || a.file.split("/").pop() === row.source_file.split("/").pop())
    : 0;
  if (found < 0) return null;
  return `${apiRoot}/regulations/${encodeURIComponent(doc.id)}/sources/${found}${row?.page ? `#page=${row.page}` : ""}`;
}
function RegulationExplorer({
  selected,
  onChange,
  enabled = true,
  onRecords,
  maximum = 2000,
}) {
  const [documents, setDocuments] = useState([]),
    [documentId, setDocumentId] = useState(""),
    [q, setQ] = useState(""),
    [query, setQuery] = useState(""),
    [page, setPage] = useState(1),
    [clauses, setClauses] = useState({ items: [], total: 0 }),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [refresh, setRefresh] = useState(0),
    [onlySelected, setOnlySelected] = useState(false);
  useEffect(() => {
    let active = true;
    request("/regulations")
      .then((data) => {
        if (active) {
          setDocuments(data.documents);
          setDocumentId((current) => current || data.documents[0]?.id || "");
        }
      })
      .catch((e) => active && setError(e.message));
    return () => {
      active = false;
    };
  }, [refresh]);
  useEffect(() => {
    if (!documentId) return;
    const controller = new AbortController();
    setBusy(true);
    setError("");
    request(
      `/regulations/${encodeURIComponent(documentId)}/clauses?q=${encodeURIComponent(query)}&offset=${(page - 1) * 20}&limit=20`,
      { signal: controller.signal },
    )
      .then((data) => {
        if (!controller.signal.aborted) {
          setClauses(data);
          onRecords?.(data.items);
        }
      })
      .catch((e) => {
        if (!controller.signal.aborted) setError(e.message);
      })
      .finally(() => {
        if (!controller.signal.aborted) setBusy(false);
      });
    return () => controller.abort();
  }, [documentId, query, page, refresh]);
  const doc = documents.find((d) => d.id === documentId),
    selectedSet = useMemo(() => new Set(selected), [selected]);
  const rows = onlySelected
    ? clauses.items.filter((c) => selectedSet.has(c.key))
    : clauses.items;
  return (
    <div className="regulation-explorer">
      <div className="regulation-documents">
        <div className="regulation-list-title">
          <BookOutlined /> 法规与标准版本
        </div>
        <Menu
          selectedKeys={[documentId]}
          items={documents.map((d) => ({
            key: d.id,
            label: (
              <div className="regulation-document-item">
                <strong>{d.code || d.title}</strong>
                <span>{d.edition}</span>
                <span>
                  <Coverage document={d} />
                  <Text type="secondary">{d.clause_count} 项</Text>
                </span>
              </div>
            ),
          }))}
          onClick={({ key }) => {
            setDocumentId(key);
            setPage(1);
            setQ("");
            setQuery("");
          }}
        />
      </div>
      <div className="regulation-clauses">
        {error && (
          <Alert
            type="error"
            showIcon
            title={error}
            action={
              <Button onClick={() => setRefresh((v) => v + 1)}>重试</Button>
            }
            className="section-card"
          />
        )}
        {doc && (
          <>
            <div className="regulation-heading">
              <div>
                <Typography.Title level={4}>{doc.title}</Typography.Title>
                <Space wrap>
                  <Coverage document={doc} />
                  <Text type="secondary">{doc.edition}</Text>
                  <Text type="secondary">{doc.status}</Text>
                </Space>
              </div>
              <Space wrap>
                {sourcePdfs(doc).map((a, index) => (
                  <a
                    key={a.file}
                    href={`${apiRoot}/regulations/${encodeURIComponent(doc.id)}/sources/${index}`}
                    target="_blank"
                    rel="noreferrer noopener"
                  >
                    {a.label || (index === 0 ? "查看原文" : `补充原件 ${index + 1}`)}
                  </a>
                ))}
                {sourceLink(doc.metadata_source_url || doc.source_url, doc.metadata_source_url ? "官方目录" : "官方来源")}
                {doc.metadata_source_url && sourceLink(doc.source_url, "取得来源")}
              </Space>
            </div>
            {doc.source_note && <Paragraph type="secondary">{doc.source_note}</Paragraph>}
            {doc.editorial_notes?.length > 0 && <Alert className="section-card" type="warning" showIcon title="原文核对说明" description={Array.isArray(doc.editorial_notes) ? doc.editorial_notes.join("；") : doc.editorial_notes} />}
            <Alert
              className="section-card"
              type={
                doc.coverage === "complete_for_edition" ? "info" : "warning"
              }
              showIcon
              title={
                doc.coverage === "complete_for_edition"
                  ? "引用绑定此版本，后续修订需另行核对。"
                  : "本版本尚未完整核验，具体收录范围见下方说明。"
              }
              description={
                doc.coverage_note ||
                "条文勾选仅表示纳入分析范围，不代表已经满足法规要求。"
              }
            />
            {doc.later_amendments_note && (
              <Alert
                type="warning"
                showIcon
                className="section-card"
                title="版本与后续修订"
                description={doc.later_amendments_note}
              />
            )}
            <Paragraph type="secondary">
              目录包含章节、分项、附录和表格行。条目数量不等于独立要求数量；正文依据所列来源逐条收录。
            </Paragraph>
            {doc.amends && (
              <Paragraph type="secondary">
                修改对象：
                {documents.find((d) => d.id === doc.amends)?.code || doc.amends}
                。修改单单独保留来源，引用时同时核对适用的原版条款。
              </Paragraph>
            )}
            <div className="table-toolbar">
              <Space wrap>
                <Text strong>条款目录</Text>
                <Tag>{clauses.total} 项</Tag>
                <Select
                  aria-label="条款显示范围"
                  value={onlySelected ? "selected" : "all"}
                  onChange={(v) => setOnlySelected(v === "selected")}
                  options={[
                    { value: "all", label: "显示全部" },
                    { value: "selected", label: "本页已勾选" },
                  ]}
                  style={{ width: 140 }}
                />
              </Space>
              <Input.Search
                value={q}
                placeholder="搜索条号、正文或关键词"
                aria-label="搜索法规条款"
                allowClear
                onChange={(e) => {
                  setQ(e.target.value);
                  if (!e.target.value) {
                    setQuery("");
                    setPage(1);
                  }
                }}
                onSearch={(value) => {
                  setQuery(value);
                  setPage(1);
                }}
                style={{ width: 300 }}
              />
            </div>
            <Table
              rowKey="key"
              size="small"
              loading={busy}
              dataSource={rows}
              rowSelection={
                enabled
                  ? {
                      selectedRowKeys: selected,
                      preserveSelectedRowKeys: true,
                      onChange: (keys) => {
                        if (keys.length <= maximum) onChange(keys);
                        else setError(`每次最多选择 ${maximum} 条。`);
                      },
                      getCheckboxProps: (r) => ({
                        disabled: r.citable === false,
                        title:
                          r.citable === false
                            ? "仅目录信息，正文尚未核对"
                            : undefined,
                      }),
                    }
                  : undefined
              }
              pagination={{
                current: page,
                pageSize: 20,
                total: clauses.total,
                showSizeChanger: false,
                onChange: setPage,
                showTotal: (n) => `共 ${n} 个条目`,
              }}
              locale={{
                emptyText: (
                  <Empty
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                    description={
                      clauses.total
                        ? "当前筛选没有条款"
                        : "尚无已核对正文，请补充该版本正式文件。"
                    }
                  />
                ),
              }}
              expandable={{
                expandedRowRender: (r) => (
                  <div className="regulation-clause-detail">
                    <Descriptions
                      size="small"
                      column={2}
                      items={[
                        {
                          key: "edition",
                          label: "版本",
                          children: doc.edition,
                        },
                        { key: "number", label: "条号 / 位置", children: r.number },
                        ...(r.source_number && r.source_number !== r.number
                          ? [{ key: "source-number", label: "原文编号", children: r.source_number }]
                          : []),
                        {
                          key: "parent",
                          label: "上级章节",
                          children: r.parent_label || r.section || "正文",
                        },
                        {
                          key: "normative",
                          label: "性质",
                          children:
                            r.normative === false
                              ? "说明 / 定义 / 附录参考"
                              : r.normative === true ? "规范性条文" : "未单独判定规范性",
                        },
                      ]}
                    />
                    <Paragraph className="pre-wrap">
                      {r.body || r.description}
                    </Paragraph>
                    {r.source_note && <Alert type="warning" showIcon title="原文说明" description={r.source_note} />}
                    <Space>
                      {sourceLink(r.source_url || doc.source_url, "来源页面")}
                      {r.page && (
                        <Text type="secondary">来源页码 {r.page}</Text>
                      )}
                      {localSource(doc, r) && <a href={localSource(doc, r)} target="_blank" rel="noreferrer noopener">{r.page ? `定位原文第 ${r.page}${r.page_end && r.page_end !== r.page ? `–${r.page_end}` : ""} 页` : "核对原文"}</a>}
                    </Space>
                  </div>
                ),
              }}
              columns={[
                { title: "条号 / 位置", dataIndex: "number", width: 150,
                  render: (value, r) => (
                    <Space orientation="vertical" size={3} style={{ overflowWrap: "anywhere" }}>
                      <Text>{r.source_number || value}</Text>
                      {r.source_number && r.source_number !== value && <Text type="secondary" style={{fontSize: 12}}>{value}</Text>}
                    </Space>
                  ),
                },
                {
                  title: "条款",
                  dataIndex: "title",
                  width: 220,
                  render: (value, r) => (
                    <Space orientation="vertical" size={3}>
                      <Text strong>{value || r.number}</Text>
                      {r.parent && (
                        <Text type="secondary" style={{ fontSize: 12 }}>
                          {r.parent_label || "上级章节"}
                        </Text>
                      )}
                    </Space>
                  ),
                },
                {
                  title: "正文预览",
                  render: (_, r) => (
                    <Paragraph
                      ellipsis={{ rows: 3 }}
                      style={{ marginBottom: 0 }}
                    >
                      {r.description || r.body}
                    </Paragraph>
                  ),
                },
              ]}
              scroll={{ x: 650 }}
            />
          </>
        )}
        {!doc && !error && <Empty description="正在加载法规版本" />}
      </div>
    </div>
  );
}
export function RegulationsView() {
  const { ws, save, message } = useWorkbench();
  const [selected, setSelected] = useState(ws?.regulation_refs || []),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  useEffect(() => {
    setSelected(ws?.regulation_refs || []);
    setError("");
  }, [ws?.id, ws?.revision]);
  const changed =
    JSON.stringify([...selected].sort()) !==
    JSON.stringify([...(ws?.regulation_refs || [])].sort());
  async function persist() {
    setBusy(true);
    setError("");
    try {
      await save("select-regulations", { references: selected });
      message.success("项目法规引用已保存");
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <PageTitle
        title="法规数据库"
        description="按正式文档版本管理章节与条款，勾选纳入项目，并在安全需求中建立引用。"
        extra={
          ws ? (
            <Button
              type="primary"
              icon={<CheckOutlined />}
              onClick={persist}
              loading={busy}
              disabled={!changed}
            >
              保存项目引用（{selected.length}）
            </Button>
          ) : undefined
        }
      />
      {error && (
        <Alert type="error" title={error} showIcon className="section-card" />
      )}
      <Alert
        type="info"
        showIcon
        className="section-card"
        title={
          ws
            ? `${ws.title} · 已保存 ${(ws.regulation_refs || []).length} 条项目引用${changed ? "，当前选择尚未保存" : ""}`
            : "当前为公共资料浏览。进入项目后可勾选引用。"
        }
        description="逐条引用保留文档版本和来源。引用不等于符合性确认；完整收录状态以各版本说明为准。"
      />
      <Card>
        <RegulationExplorer
          selected={selected}
          onChange={setSelected}
          enabled={!!ws}
        />
      </Card>
    </>
  );
}
export function RegulationPicker({ config, onClose }) {
  const [selected, setSelected] = useState([]),
    [records, setRecords] = useState({});
  useEffect(() => {
    if (config) {
      setSelected(config.initialRefs || []);
      setRecords({});
    }
  }, [config]);
  return (
    <Drawer
      title="勾选法规条款"
      open={!!config}
      onClose={onClose}
      size={1180}
      destroyOnHidden
      footer={
        <div className="editor-footer">
          <Text type="secondary">
            已选择 {selected.length} 条；跨文档、翻页和搜索会保留选择。
          </Text>
          <Space>
            <Button onClick={onClose}>取消</Button>
            <Button
              type="primary"
              onClick={() => {
                config.onSelect(selected, records);
                onClose();
              }}
            >
              引用所选条款
            </Button>
          </Space>
        </div>
      }
    >
      {config && (
        <RegulationExplorer
          selected={selected}
          onChange={setSelected}
          onRecords={(rows) =>
            setRecords((old) => ({
              ...old,
              ...Object.fromEntries(rows.map((row) => [row.key, row])),
            }))
          }
          maximum={500}
        />
      )}
    </Drawer>
  );
}
