import React, { useEffect, useRef, useState } from "react";
import {
  Alert,
  Button,
  Checkbox,
  Descriptions,
  Drawer,
  Empty,
  Input,
  Segmented,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from "antd";
import {
  CheckOutlined,
  DatabaseOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
import { request, properties } from "./api.js";
import { useWorkbench } from "./Workbench.jsx";

const { Text, Paragraph } = Typography;
const componentReference = (value) =>
  typeof value === "string" && /^systems:\d+$/.test(value) ? value : "";
const errorText = (error) =>
  typeof error?.message === "string" ? error.message : "操作未完成，请重试。";
const MAX_SELECTION = 100;

export function KnowledgeAutomation({ config, onClose }) {
  const { ws, save, reload, message, setPicker, setReference, setEditor } =
    useWorkbench();
  const open = Boolean(config);
  const workspaceId = ws?.id;
  const [assetId, setAssetId] = useState("");
  const [referenceState, setReferenceState] = useState("");
  const [referenceTitle, setReferenceTitle] = useState("");
  const [preview, setPreview] = useState(null);
  const [selectedKeys, setSelectedKeys] = useState([]);
  const [scope, setScope] = useState("available");
  const [query, setQuery] = useState("");
  const [includeControls, setIncludeControls] = useState(true);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [previewError, setPreviewError] = useState("");
  const [saveError, setSaveError] = useState(null);
  const [refresh, setRefresh] = useState(0);
  const requestSequence = useRef(0);
  const saveInFlight = useRef(false);
  const current = useRef({});
  current.current = { open, workspaceId, assetId, reference: referenceState };

  useEffect(() => {
    const asset =
      ws?.assets.find((row) => row.id === config?.assetId) || ws?.assets[0];
    setAssetId(open ? asset?.id || "" : "");
    setReferenceState(
      open
        ? componentReference(config?.reference) ||
            componentReference(asset?.reference)
        : "",
    );
    setReferenceTitle("");
    setSelectedKeys([]);
    setPreview(null);
    setPreviewError("");
    setSaveError(null);
    setIncludeControls(true);
    setScope("available");
    setQuery("");
  }, [config, workspaceId]);

  useEffect(() => {
    const sequence = ++requestSequence.current;
    const controller = new AbortController();
    setPreview(null);
    setPreviewError("");
    if (
      !open ||
      !workspaceId ||
      !assetId ||
      !componentReference(referenceState)
    ) {
      setLoading(false);
      return () => controller.abort();
    }
    setLoading(true);
    const query = new URLSearchParams({
      asset_id: assetId,
      reference: referenceState,
    });
    request(
      `/workspaces/${encodeURIComponent(workspaceId)}/recommendations?${query}`,
      { signal: controller.signal },
    )
      .then((result) => {
        if (controller.signal.aborted || sequence !== requestSequence.current)
          return;
        setPreview(result);
        setReferenceTitle(result.reference.title);
        setSelectedKeys((keys) =>
          keys.filter((key) =>
            result.items.some((row) => row.key === key && row.selectable),
          ),
        );
      })
      .catch((error) => {
        if (controller.signal.aborted || sequence !== requestSequence.current)
          return;
        setPreviewError(errorText(error));
      })
      .finally(() => {
        if (!controller.signal.aborted && sequence === requestSequence.current)
          setLoading(false);
      });
    return () => controller.abort();
  }, [open, workspaceId, ws?.revision, assetId, referenceState, refresh]);

  const asset = ws?.assets.find((row) => row.id === assetId);
  const matchesPreview =
    preview?.workspace_id === workspaceId &&
    preview?.asset.id === assetId &&
    preview?.reference.key === referenceState;
  const previewStale = matchesPreview && preview.revision !== ws?.revision;
  const items = matchesPreview ? preview.items : [];
  const rank = (row) =>
    row.selectable && !row.already_applied ? 0 : row.already_applied ? 1 : 2;
  const search = query.trim().toLocaleLowerCase();
  const visibleItems = [...items]
    .sort((a, b) => rank(a) - rank(b))
    .filter((row) => scope === "all" || row.selectable)
    .filter(
      (row) =>
        !search ||
        [
          row.title,
          row.property_label,
          row.damage,
          row.threat,
          ...row.warnings,
          ...row.paths.flatMap((path) => [path.title, path.steps]),
          ...row.controls.flatMap((control) => [
            control.title,
            control.statement,
          ]),
        ]
          .join(" ")
          .toLocaleLowerCase()
          .includes(search),
    );
  const selected = items.filter(
    (row) => row.selectable && selectedKeys.includes(row.key),
  );
  const hiddenSelected = selected.filter(
    (row) => !visibleItems.some((visible) => visible.key === row.key),
  ).length;
  const pathCount = selected.reduce((sum, row) => sum + row.paths.length, 0);
  const controlCount = selected.reduce(
    (sum, row) => sum + row.controls.length,
    0,
  );
  const selectedMissing = selected.reduce(
    (sum, row) =>
      sum +
      row.missing_paths.length +
      (includeControls ? row.missing_controls.length : 0),
    0,
  );
  const missingRows = items.filter(
    (row) => row.missing_paths.length || row.missing_controls.length,
  );
  const missingPathCount = missingRows.reduce(
    (sum, row) => sum + row.missing_paths.length,
    0,
  );
  const missingControlCount = missingRows.reduce(
    (sum, row) => sum + row.missing_controls.length,
    0,
  );
  const clearSelection = () => {
    setSelectedKeys([]);
    setPreview(null);
    setPreviewError("");
    setSaveError(null);
    setScope("available");
    setQuery("");
  };
  const chooseAsset = (value) => {
    clearSelection();
    setAssetId(value);
    setReferenceState(
      componentReference(ws.assets.find((row) => row.id === value)?.reference),
    );
    setReferenceTitle("");
  };
  const chooseReference = () => {
    const target = { workspaceId, assetId };
    setPicker({
      category: "systems",
      multiple: false,
      onSelect: (rows) => {
        const now = current.current;
        if (
          !now.open ||
          now.workspaceId !== target.workspaceId ||
          now.assetId !== target.assetId ||
          saveInFlight.current
        )
          return;
        const row = rows[0];
        const key = componentReference(row?.key);
        if (!key) return;
        clearSelection();
        setReferenceState(key);
        setReferenceTitle(row.title || "");
        setRefresh((value) => value + 1);
      },
    });
  };
  const retryPreview = async () => {
    if (saveError?.status === 409 || previewStale) {
      try {
        await reload();
      } catch (error) {
        setPreviewError(errorText(error));
        return;
      }
    }
    setSaveError(null);
    setRefresh((value) => value + 1);
  };
  const applySelection = async () => {
    if (
      saveInFlight.current ||
      loading ||
      !matchesPreview ||
      previewStale ||
      !selected.length
    )
      return;
    const target = { workspaceId, assetId, reference: referenceState };
    const count = selected.length;
    const payload = {
      asset_id: assetId,
      reference: referenceState,
      relationship_keys: selected.map((row) => row.key),
      include_controls: includeControls,
    };
    saveInFlight.current = true;
    setSaving(true);
    setSaveError(null);
    try {
      await save("apply-reference-selection", payload);
      const now = current.current;
      message.success(
        `已选用 ${count} 项关联知识，请继续完善场景评估${includeControls ? "和需求内容" : ""}。`,
      );
      if (
        now.open &&
        now.workspaceId === target.workspaceId &&
        now.assetId === target.assetId &&
        now.reference === target.reference
      ) {
        setSelectedKeys([]);
        setRefresh((value) => value + 1);
      }
    } catch (error) {
      const now = current.current;
      if (
        now.open &&
        now.workspaceId === target.workspaceId &&
        now.assetId === target.assetId &&
        now.reference === target.reference
      ) {
        setSaveError({ message: errorText(error), status: error.status });
      }
    } finally {
      saveInFlight.current = false;
      setSaving(false);
    }
  };

  const referenceLink = (key, label) => (
    <Button
      type="link"
      size="small"
      className="text-button"
      onClick={() => setReference(key)}
    >
      {label || key}
    </Button>
  );
  const expandedRow = (row) => (
    <div className="preview-detail">
      <Descriptions
        size="small"
        column={1}
        items={[
          {
            key: "damage",
            label: "损害场景",
            children: (
              <div className="pre-wrap">{row.damage || "来源未提供"}</div>
            ),
          },
          {
            key: "threat",
            label: "威胁描述",
            children: (
              <div className="pre-wrap">{row.threat || "来源未提供"}</div>
            ),
          },
          {
            key: "source",
            label: "关联来源",
            children: (
              <Space wrap>
                {referenceLink(row.key, "查看关联知识")}
                {referenceLink(row.pattern_reference, "查看威胁模式")}
              </Space>
            ),
          },
        ]}
      />
      <Text strong>攻击路径（{row.paths.length}）</Text>
      <Table
        size="small"
        rowKey="reference"
        pagination={false}
        dataSource={row.paths}
        locale={{ emptyText: "没有可引用路径，需在项目中补充" }}
        columns={[
          { title: "路径名称", dataIndex: "title", width: 220 },
          {
            title: "步骤与前提",
            dataIndex: "steps",
            render: (value) => (
              <div className="pre-wrap">{value || "待补充"}</div>
            ),
          },
          {
            title: "来源",
            dataIndex: "reference",
            width: 135,
            render: (value) => referenceLink(value),
          },
        ]}
      />
      <Text strong>控制措施（{row.controls.length}）</Text>
      <Table
        size="small"
        rowKey="reference"
        pagination={false}
        dataSource={row.controls}
        locale={{ emptyText: "没有可引用控制措施" }}
        columns={[
          { title: "措施名称", dataIndex: "title", width: 220 },
          {
            title: "需求草稿内容",
            dataIndex: "statement",
            render: (value) => <div className="pre-wrap">{value}</div>,
          },
          {
            title: "来源",
            dataIndex: "reference",
            width: 135,
            render: (value) => referenceLink(value),
          },
        ]}
      />
      {(row.missing_paths.length > 0 || row.missing_controls.length > 0) && (
        <Alert
          type="warning"
          showIcon
          title="以下引用缺失，对应内容将跳过"
          description={
            <Space orientation="vertical" size={4}>
              {row.missing_paths.length > 0 && (
                <Text>攻击技术：{row.missing_paths.join("、")}</Text>
              )}
              {row.missing_controls.length > 0 && (
                <Text>控制措施：{row.missing_controls.join("、")}</Text>
              )}
            </Space>
          }
        />
      )}
    </div>
  );

  return (
    <Drawer
      title="批量选用参考资料"
      open={open}
      width={1100}
      onClose={() => !saveInFlight.current && onClose()}
      closable={!saving}
      maskClosable={!saving}
      keyboard={!saving}
      footer={
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            gap: 16,
            flexWrap: "wrap",
          }}
        >
          <Text type="secondary">
            已勾选 {selected.length} 项
            {selected.length > 0 &&
              ` · ${pathCount} 条路径${includeControls ? ` · ${controlCount} 条需求草稿` : ""}`}
          </Text>
          <Space>
            <Button disabled={saving} onClick={onClose}>
              取消
            </Button>
            <Button
              type="primary"
              icon={<CheckOutlined />}
              loading={saving}
              disabled={
                loading ||
                previewStale ||
                !selected.length ||
                Boolean(previewError) ||
                saveError?.status === 409
              }
              onClick={applySelection}
            >
              确认选用已勾选的 {selected.length} 项
            </Button>
          </Space>
        </div>
      }
    >
      <Space orientation="vertical" size={20} style={{ width: "100%" }}>
        <Paragraph type="secondary" style={{ marginBottom: 0 }}>
          按资产和组件参考查看已有知识关系，勾选后生成可编辑的场景与需求草稿。影响、攻击可行性及残余风险由项目人员补充依据并确认。
        </Paragraph>
        <Space wrap align="end" size={20}>
          <div>
            <Text strong>目标资产</Text>
            <Select
              aria-label="选择目标资产"
              placeholder="选择项目资产"
              value={assetId || undefined}
              disabled={saving || !ws?.assets.length}
              onChange={chooseAsset}
              showSearch
              optionFilterProp="label"
              style={{ display: "block", width: 300, marginTop: 8 }}
              options={(ws?.assets || []).map((row) => ({
                value: row.id,
                label: row.name,
              }))}
            />
          </div>
          <div>
            <Text strong>组件参考</Text>
            <Space style={{ display: "flex", marginTop: 8 }}>
              <Button
                icon={<DatabaseOutlined />}
                disabled={saving || !assetId}
                onClick={chooseReference}
              >
                {referenceState ? "更换组件参考" : "选择组件参考"}
              </Button>
              {referenceState &&
                referenceLink(referenceState, referenceTitle || referenceState)}
            </Space>
          </div>
          <Button
            icon={<ReloadOutlined />}
            loading={loading}
            disabled={saving || !assetId || !referenceState}
            onClick={retryPreview}
          >
            刷新预览
          </Button>
        </Space>
        {asset && (
          <Space wrap>
            <Text type="secondary">资产安全属性</Text>
            {asset.properties.map((key) => (
              <Tag key={key}>{properties[key] || key}</Tag>
            ))}
          </Space>
        )}
        {!ws?.assets.length && (
          <Empty description="先识别项目资产，再选用参考资料">
            <Button
              onClick={() => {
                onClose();
                setEditor({ kind: "asset" });
              }}
            >
              新增资产
            </Button>
          </Empty>
        )}
        {assetId && !referenceState && (
          <Alert
            type="info"
            showIcon
            title="请选择与该资产对应的组件参考"
            description="将根据本地已有的组件关联查询威胁、攻击路径和控制措施。"
          />
        )}
        {previewError && (
          <Alert
            type="error"
            showIcon
            title="预览未加载"
            description={previewError}
            action={
              <Button size="small" disabled={saving} onClick={retryPreview}>
                重试
              </Button>
            }
          />
        )}
        {previewStale && (
          <Alert
            type="warning"
            showIcon
            title="项目版本已变化，请刷新项目与预览后核对所选内容。"
            action={
              <Button
                size="small"
                disabled={saving || loading}
                onClick={retryPreview}
              >
                刷新预览
              </Button>
            }
          />
        )}
        {saveError && (
          <Alert
            type="error"
            showIcon
            title="选用结果待确认"
            description={`${saveError.message}${saveError.status === 409 ? " 请刷新预览后核对所选内容。" : " 可保留当前选择重试，或刷新预览核对已选用状态。"}`}
            action={
              <Button
                size="small"
                disabled={saving || loading}
                onClick={retryPreview}
              >
                刷新预览
              </Button>
            }
          />
        )}
        {(loading || matchesPreview) && (
          <>
            {matchesPreview && (
              <div className="automation-summary">
                <Space wrap size={20}>
                  <Text>
                    关联知识 <Text strong>{preview.summary.total}</Text> 项
                  </Text>
                  <Tag color="blue">可选用 {preview.summary.available}</Tag>
                  <Tag color="success">
                    已选用 {preview.summary.already_applied}
                  </Tag>
                  <Tag color={preview.summary.blocked ? "warning" : "default"}>
                    暂不可用 {preview.summary.blocked}
                  </Tag>
                  <Text type="secondary">预览版本 R{preview.revision}</Text>
                </Space>
              </div>
            )}
            {missingRows.length > 0 && (
              <Alert
                type="warning"
                showIcon
                title={`全部关联知识中，${missingRows.length} 项存在缺失引用`}
                description={`缺失攻击路径引用 ${missingPathCount} 条、控制措施引用 ${missingControlCount} 条。对应内容将跳过，可展开条目查看原因；统计包含筛选外条目。`}
                action={
                  <Button
                    size="small"
                    disabled={saving}
                    onClick={() => {
                      setScope("all");
                      setQuery("");
                    }}
                  >
                    查看全部
                  </Button>
                }
              />
            )}
            <Space
              wrap
              style={{ justifyContent: "space-between", width: "100%" }}
            >
              <Segmented
                aria-label="筛选知识选用状态"
                disabled={saving}
                value={scope}
                onChange={setScope}
                options={[
                  {
                    value: "available",
                    label: `仅看可选用${matchesPreview ? `（${preview.summary.available}）` : ""}`,
                  },
                  { value: "all", label: "全部" },
                ]}
              />
              <Input.Search
                aria-label="搜索关联知识"
                placeholder="搜索威胁、属性、路径或措施"
                allowClear
                value={query}
                disabled={saving}
                onChange={(event) => setQuery(event.target.value)}
                style={{ width: 310 }}
              />
            </Space>
            {hiddenSelected > 0 && (
              <Text type="secondary">
                已勾选的 {hiddenSelected}{" "}
                项被当前筛选隐藏，选择仍然保留并计入确认数量。
              </Text>
            )}
            <Table
              rowKey="key"
              loading={loading}
              size="middle"
              dataSource={visibleItems}
              scroll={{ x: 900 }}
              pagination={{
                defaultPageSize: 8,
                showSizeChanger: true,
                showTotal: (total) => `当前显示 ${total} 项关联知识`,
              }}
              locale={{
                emptyText: (
                  <Empty
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                    description={
                      !items.length
                        ? "该组件没有关联知识，可更换组件参考或手动建立场景。"
                        : scope === "available" && !preview.summary.available
                          ? "该资产当前没有可直接选用的关联知识。切换“全部”查看已选用记录和不可用原因。"
                          : "没有符合当前关键词的关联知识。"
                    }
                  >
                    {items.length > 0 && (
                      <Space>
                        {scope === "available" && (
                          <Button
                            onClick={() => {
                              setScope("all");
                              setQuery("");
                            }}
                          >
                            查看全部及原因
                          </Button>
                        )}
                        {query && (
                          <Button onClick={() => setQuery("")}>
                            清空关键词
                          </Button>
                        )}
                      </Space>
                    )}
                  </Empty>
                ),
              }}
              rowSelection={{
                selectedRowKeys: selectedKeys,
                preserveSelectedRowKeys: true,
                getCheckboxProps: (row) => ({
                  disabled: saving || previewStale || !row.selectable,
                }),
                onChange: (keys) => {
                  const valid = keys.filter((key) =>
                    items.some((row) => row.key === key && row.selectable),
                  );
                  if (valid.length > MAX_SELECTION)
                    message.info(
                      `每次最多选用 ${MAX_SELECTION} 项，已保留前 ${MAX_SELECTION} 项。`,
                    );
                  setSelectedKeys(valid.slice(0, MAX_SELECTION));
                  setSaveError(null);
                },
              }}
              expandable={{ expandedRowRender: expandedRow }}
              columns={[
                {
                  title: "关联威胁",
                  dataIndex: "title",
                  width: 250,
                  render: (value) => <Text>{value}</Text>,
                },
                {
                  title: "安全属性",
                  dataIndex: "property_label",
                  width: 100,
                  render: (value) => value || "未提供",
                },
                {
                  title: "路径",
                  dataIndex: "paths",
                  width: 65,
                  render: (value) => value.length,
                },
                {
                  title: "措施",
                  dataIndex: "controls",
                  width: 65,
                  render: (value) => value.length,
                },
                {
                  title: "选用状态与提示",
                  width: 285,
                  render: (_, row) => (
                    <Space orientation="vertical" size={4}>
                      <Space>
                        {row.already_applied ? (
                          <Tag color="success">已选用</Tag>
                        ) : row.selectable ? (
                          <Tag color="blue">可选用</Tag>
                        ) : (
                          <Tag color="warning">暂不可用</Tag>
                        )}
                        {row.already_applied && row.scenario_id && (
                          <Button
                            type="link"
                            size="small"
                            disabled={saving}
                            onClick={() => {
                              onClose();
                              setEditor({
                                kind: "scenario",
                                id: row.scenario_id,
                              });
                            }}
                          >
                            查看场景
                          </Button>
                        )}
                      </Space>
                      {row.warnings.map((warning, index) => (
                        <Text key={index} type="secondary">
                          {warning}
                        </Text>
                      ))}
                    </Space>
                  ),
                },
              ]}
            />
            <Checkbox
              checked={includeControls}
              disabled={saving}
              onChange={(event) => {
                setIncludeControls(event.target.checked);
                setSaveError(null);
              }}
            >
              同时将已有控制措施生成安全需求草稿
            </Checkbox>
            <Text type="secondary">
              需求分配对象和验证准则仍需补充；本地缺失的引用会保留提示。每次最多选用{" "}
              {MAX_SELECTION} 项。
            </Text>
            {selectedMissing > 0 && (
              <Alert
                type="warning"
                showIcon
                title={`已选内容涉及 ${selectedMissing} 条缺失引用，对应内容将跳过。请展开条目核对。`}
              />
            )}
          </>
        )}
      </Space>
    </Drawer>
  );
}
