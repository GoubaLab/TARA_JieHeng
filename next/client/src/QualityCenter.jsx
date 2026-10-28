import React, { useState } from "react";
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Form,
  Input,
  Modal,
  Row,
  Space,
  Statistic,
  Table,
  Tabs,
  Tag,
  Typography,
} from "antd";
import { PageTitle, useWorkbench } from "./Workbench.jsx";
import { useAuth } from "./AuthGate.jsx";
import { date, navigate, properties } from "./api.js";
import { mutateResource, useResource } from "./platform-ui.js";
import { ProjectMembers } from "./TeamSettings.jsx";
const stageNames = {
  definition: "相关项定义",
  architecture: "架构与资产",
  assessment: "风险分析",
  treatment: "风险处置",
  verification: "需求验证",
};
export default function QualityCenter() {
  const { ws, reload, openIssue } = useWorkbench(),
    { user, mode } = useAuth(),
    { message } = App.useApp();
  const quality = useResource(`/workspaces/${ws.id}/quality`),
    checkpoints = useResource(`/workspaces/${ws.id}/checkpoints`);
  const [creating, setCreating] = useState(false),
    [restoring, setRestoring] = useState(null),
    [name, setName] = useState(""),
    [reason, setReason] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const create = async () => {
    setBusy(true);
    try {
      await mutateResource(
        `/workspaces/${ws.id}/checkpoints`,
        { name },
        ws.revision,
      );
      setCreating(false);
      checkpoints.refresh();
      message.success("还原点已保存");
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  const restore = async () => {
    setBusy(true);
    try {
      await mutateResource(
        `/workspaces/${ws.id}/restorations`,
        { checkpoint_id: restoring.id, reason },
        ws.revision,
      );
      setRestoring(null);
      await reload({ quiet: true });
      quality.refresh();
      checkpoints.refresh();
      message.success("已还原为新的项目修订；评估方法需要重新确认");
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  const q = quality.data,
    canRestore = ["admin", "reviewer"].includes(ws.access_role || user.role);
  return (
    <>
      <PageTitle
        title="交付检查与恢复"
        description="集中处理流程缺项、资产覆盖和资料确认；在关键操作前保存可恢复版本。"
        extra={
          <Space>
            <Button
              onClick={() => {
                quality.refresh();
                checkpoints.refresh();
              }}
            >
              刷新检查
            </Button>
            <Button onClick={() => navigate("verification", ws.id)}>
              验证与报告
            </Button>
          </Space>
        }
      />
      {(error || quality.error || checkpoints.error) && (
        <Alert
          className="section-card"
          type="error"
          title={error || quality.error || checkpoints.error}
          closable
          onClose={() => setError("")}
        />
      )}
      <Row gutter={16} className="section-card">
        {[
          ["当前修订", ws.revision],
          ["待处理问题", q?.issues.length || 0],
          [
            "已关联威胁的资产",
            `${q?.counts.covered_assets || 0} / ${q?.counts.assets || 0}`,
          ],
          ["通信待确认", q?.communication_to_confirm.length || 0],
        ].map(([title, value]) => (
          <Col xs={12} lg={6} key={title}>
            <Card size="small">
              <Statistic title={title} value={value} />
            </Card>
          </Col>
        ))}
      </Row>
      <Card>
        <Tabs
          items={[
            {
              key: "quality",
              label: "流程与覆盖",
              children: (
                <Space orientation="vertical" style={{ width: "100%" }}>
                  <Alert
                    type={q?.ready ? "success" : "info"}
                    showIcon
                    title={
                      q?.ready
                        ? "基础完整性检查通过，请继续核对覆盖提醒与工程结论"
                        : "按问题逐项完善，再建立评审基线"
                    }
                    description="AI 可协助发现和补齐缺项；真实验证、风险接受和评审责任仍由工作人员完成。"
                  />
                  <Table
                    rowKey={(_, i) => i}
                    loading={quality.loading}
                    dataSource={q?.issues || []}
                    columns={[
                      {
                        title: "环节",
                        dataIndex: "stage",
                        width: 150,
                        render: (v) => stageNames[v] || v,
                      },
                      {
                        title: "问题",
                        render: (_, i) =>
                          Object.entries(properties).reduce(
                            (text, [key, label]) => text.replaceAll(key, label),
                            i.message || i.detail || "",
                          ),
                      },
                      {
                        title: "操作",
                        width: 100,
                        render: (_, i) => (
                          <Button
                            type="link"
                            onClick={() =>
                              openIssue({
                                ...i,
                                message:
                                  Object.entries(properties).reduce(
                                    (text, [key, label]) =>
                                      text.replaceAll(key, label),
                                    i.message || i.detail || "",
                                  ) || "",
                              })
                            }
                          >
                            前往处理
                          </Button>
                        ),
                      },
                    ]}
                    pagination={{ pageSize: 12 }}
                  />
                  {!!q?.communication_to_confirm.length && (
                    <Alert
                      type="warning"
                      title="以下连接尚未明确协议"
                      description={q.communication_to_confirm
                        .map((f) => f.name)
                        .join("、")}
                      action={
                        <Button onClick={() => navigate("architecture", ws.id)}>
                          检查连线
                        </Button>
                      }
                    />
                  )}
                  <Button onClick={() => navigate("assistant", ws.id)}>
                    生成辅助复核建议
                  </Button>
                </Space>
              ),
            },
            {
              key: "recovery",
              label: "项目还原点",
              children: (
                <Space orientation="vertical" style={{ width: "100%" }}>
                  <Alert
                    type="info"
                    title="还原不会覆盖历史操作记录和冻结基线；执行前还会保存当前状态，便于撤回本次还原。"
                    description="还原点与项目保存在同一数据库中，不能替代独立离线备份。备份方式见企业部署文档。"
                  />
                  {(ws.access_role || user.role) !== "viewer" && (
                    <Button
                      type="primary"
                      onClick={() => {
                        setName(`修订 ${ws.revision} · 关键修改前`);
                        setCreating(true);
                      }}
                    >
                      创建还原点
                    </Button>
                  )}
                  <Table
                    rowKey="id"
                    dataSource={checkpoints.data?.items || []}
                    loading={checkpoints.loading}
                    columns={[
                      {
                        title: "还原点",
                        dataIndex: "name",
                        render: (v) => v.replace(/ · job_.+$/, ""),
                      },
                      { title: "项目修订", dataIndex: "revision" },
                      {
                        title: "保存时间",
                        dataIndex: "created_at",
                        render: date,
                      },
                      {
                        title: "保存人",
                        dataIndex: "actor",
                        render: (v) => (v === "local" ? "本地管理员" : v),
                      },
                      {
                        title: "操作",
                        render: (_, cp) =>
                          canRestore && (
                            <Button
                              onClick={() => {
                                setRestoring(cp);
                                setReason("");
                              }}
                            >
                              还原到此版本
                            </Button>
                          ),
                      },
                    ]}
                    pagination={{ pageSize: 10 }}
                  />
                </Space>
              ),
            },
            ...(mode === "enterprise"
              ? [
                  {
                    key: "members",
                    label: "项目成员",
                    children: <ProjectMembers workspaceId={ws.id} />,
                  },
                ]
              : []),
          ]}
        />
      </Card>
      <Modal
        open={creating}
        title="创建项目还原点"
        onCancel={() => setCreating(false)}
        onOk={create}
        confirmLoading={busy}
        okButtonProps={{ disabled: !name.trim() }}
      >
        {error && <Alert type="error" title={error} />}
        <Input
          value={name}
          onChange={(e) => setName(e.target.value)}
          maxLength={200}
          placeholder="说明保存目的"
        />
      </Modal>
      <Modal
        open={!!restoring}
        title="还原项目工作产品"
        onCancel={() => setRestoring(null)}
        onOk={restore}
        confirmLoading={busy}
        okText="保存当前状态并还原"
        okButtonProps={{ disabled: !reason.trim() }}
      >
        <Alert
          type="warning"
          title={`将还原到：${restoring?.name || ""}`}
          description="当前内容会先建立自动还原点。本次操作产生新的修订，方法确认状态将重置为待确认，历史审计与冻结基线保留。"
        />
        {error && <Alert type="error" title={error} />}
        <Form layout="vertical">
          <Form.Item label="还原原因" required>
            <Input.TextArea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              maxLength={2000}
            />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
