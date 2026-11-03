import React, { useRef, useState } from "react";
import { Alert, Button, Modal, Space, Tooltip, Typography } from "antd";
import { useWorkbench } from "./Workbench.jsx";
import { recordCode } from "./numbering.js";
import { navigate } from "./api.js";
import { deletionBlockReason, deletionReferences } from "./asset-scope.js";
const labels = {
  asset: "资产",
  damage: "损害场景",
  scenario: "威胁场景",
  requirement: "安全需求",
  goal: "安全目标",
  decision: "处置决策",
  draft: "未确认处置草案",
};
export default function RecordDeleteButton({
  kind,
  record,
  onDeleted,
  disabled = false,
}) {
  const { ws, save, message, setEditor } = useWorkbench();
  const [open, setOpen] = useState(false),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const pending = useRef(false),
    isDraft = kind === "draft";
  const blocked = deletionBlockReason(ws, kind, record);
  const scenario = isDraft
    ? ws.scenarios.find((s) => s.id === record.scenario_id)
    : null;
  const title =
    scenario?.title ||
    record.name ||
    record.title ||
    record.description ||
    record.statement ||
    "";
  const code = isDraft
    ? recordCode(ws, "scenario", record.scenario_id)
    : record.source_code || recordCode(ws, kind, record.id);
  const references = isDraft ? [] : deletionReferences(ws, kind, record.id);
  const remove = async () => {
    if (pending.current || blocked || disabled) return;
    pending.current = true;
    setBusy(true);
    setError("");
    try {
      await save(
        isDraft ? "discard-treatment-draft" : `delete-${kind}`,
        isDraft ? { scenario_id: record.scenario_id } : { id: record.id },
      );
      setOpen(false);
      message.success(
        `${isDraft ? "未确认处置草案已放弃" : `${labels[kind]}已删除`}，可从还原点恢复`,
      );
      onDeleted?.();
    } catch (e) {
      setError(e.message || "操作未完成，请重试");
    } finally {
      pending.current = false;
      setBusy(false);
    }
  };
  const resolve = ({ kind, record }) => {
    if (pending.current) return;
    setOpen(false);
    navigate(
      {
        asset: "assets",
        damage: "damages",
        scenario: "threats",
        requirement: "treatment",
        goal: "treatment",
        decision: "treatment",
      }[kind],
      ws.id,
    );
    setEditor({
      kind,
      id: record.id || record.scenario_id,
      ...(kind === "scenario" ? { step: "threat" } : {}),
    });
  };
  return (
    <>
      <Tooltip title={blocked}>
        <span>
          <Button
            type="link"
            danger
            disabled={!!blocked || disabled}
            loading={busy}
            aria-label={`${isDraft ? "放弃处置草案" : `删除${labels[kind]}`} ${code}`}
            onClick={() => {
              setError("");
              setOpen(true);
            }}
          >
            {isDraft ? "放弃草案" : "删除"}
          </Button>
        </span>
      </Tooltip>
      <Modal
        title={isDraft ? "放弃未确认处置草案？" : `删除${labels[kind]}？`}
        open={open}
        onCancel={() => !pending.current && setOpen(false)}
        onOk={remove}
        okText={isDraft ? "确认放弃" : "确认删除"}
        cancelText="取消"
        okButtonProps={{ danger: true, disabled: !!blocked || disabled }}
        confirmLoading={busy}
        closable={!busy}
        maskClosable={!busy}
        keyboard={!busy}
        destroyOnHidden
      >
        <Space orientation="vertical" style={{ width: "100%" }}>
          <Typography.Text strong>
            {code} · {title}
          </Typography.Text>
          <Typography.Paragraph style={{ margin: 0 }}>
            {isDraft
              ? "只放弃此威胁的未确认处置草案；威胁、安全目标、安全需求和已确认处置保留。操作将写入项目历史。"
              : "记录将从当前工作版本移除，历史操作记录、已分配编号和冻结基线保留。"}
          </Typography.Paragraph>
          <Typography.Text type="secondary">
            操作前自动建立还原点，可在交付检查中恢复项目版本。
          </Typography.Text>
          {kind === "scenario" && (
            <Typography.Text>
              该威胁包含的攻击路径、处置决策和处置草稿也将移除。
            </Typography.Text>
          )}
          {kind === "asset" && record.architecture_source && (
            <Alert
              type="info"
              showIcon
              title={
                ws.architecture.asset_generation === "catalog"
                  ? "资产目录模式"
                  : "资产由架构自动维护"
              }
              description={
                ws.architecture.asset_generation === "catalog"
                  ? "删除当前资产不会移除架构对象。以后重新启用自动生成模式时，原来源可能再次生成资产。"
                  : "自动生成模式下不能删除仍有活动来源的资产。可先移除架构来源，或切换为资产目录模式。"
              }
              action={
                <Button
                  disabled={busy}
                  onClick={() => {
                    setOpen(false);
                    navigate("architecture", ws.id);
                  }}
                >
                  查看架构
                </Button>
              }
            />
          )}
          {!isDraft && (
            <Typography.Text type="secondary">
              存在依赖引用时无法删除，请先处理关联工作产品。
            </Typography.Text>
          )}
          {!!references.length && (
            <Alert
              type="info"
              showIcon
              title={`发现 ${references.length} 项关联记录`}
              description={
                <Space
                  orientation="vertical"
                  size={4}
                  style={{ width: "100%" }}
                >
                  {references.map((ref, i) =>
                    ref.kind === "draft" ? (
                      <Space key={i} wrap>
                        <Typography.Text>关联未确认处置草案</Typography.Text>
                        <RecordDeleteButton
                          kind="draft"
                          record={ref.record}
                          disabled={busy}
                          onDeleted={() => setError("")}
                        />
                      </Space>
                    ) : (
                      <Button
                        key={i}
                        type="link"
                        disabled={busy}
                        style={{
                          height: "auto",
                          whiteSpace: "normal",
                          textAlign: "left",
                          padding: 0,
                        }}
                        onClick={() => resolve(ref)}
                      >
                        处理{labels[ref.kind]} ·{" "}
                        {ref.record.name ||
                          ref.record.title ||
                          ref.record.description ||
                          ref.record.statement ||
                          recordCode(ws, "scenario", ref.record.scenario_id)}
                      </Button>
                    ),
                  )}
                </Space>
              }
            />
          )}
          {(error || blocked) && (
            <Alert
              type="error"
              showIcon
              title={isDraft ? "未能放弃草案" : "未能删除"}
              description={error || blocked}
            />
          )}
        </Space>
      </Modal>
    </>
  );
}
