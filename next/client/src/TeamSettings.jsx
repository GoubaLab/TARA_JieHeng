import React, { useState } from "react";
import {
  Alert,
  App,
  Button,
  Card,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Typography,
} from "antd";
import { PageTitle } from "./Workbench.jsx";
import { useAuth } from "./AuthGate.jsx";
import {
  mutateResource,
  roleLabels,
  useResource,
  pickFields,
} from "./platform-ui.js";
export default function TeamSettings() {
  const { mode } = useAuth(),
    r = useResource("/enterprise/users"),
    { message } = App.useApp();
  const [editing, setEditing] = useState(null),
    [form] = Form.useForm(),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const open = (p) => {
    setEditing(p);
    form.resetFields();
    form.setFieldsValue(p);
  };
  const save = async () => {
    try {
      const values = await form.validateFields();
      const p = pickFields(values, [
        "username",
        "name",
        "role",
        "enabled",
        "password",
      ]);
      setBusy(true);
      await mutateResource(
        editing.id ? `/enterprise/users/${editing.id}` : "/enterprise/users",
        p,
        editing.revision,
        editing.id ? "PUT" : "POST",
      );
      setEditing(null);
      r.refresh();
      message.success("账号已保存，原登录会话已失效");
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  if (mode !== "enterprise")
    return (
      <>
        <PageTitle
          title="企业账户与权限"
          description="企业部署模式提供身份、角色与项目成员控制。"
        />
        <Alert
          type="info"
          title="当前运行在本机模式"
          description="账户与项目权限管理适用于企业模式。请联系部署管理员配置 HTTPS 入口及初始管理员账户。"
        />
      </>
    );
  return (
    <>
      <PageTitle
        title="企业账户与权限"
        description="组织角色约束最高权限；具体项目还需分配成员。账户停用、改密或角色变更后，原会话失效。"
        extra={
          <Button
            type="primary"
            onClick={() => open({ role: "analyst", enabled: true })}
          >
            添加账户
          </Button>
        }
      />
      {(error || r.error) && <Alert type="error" title={error || r.error} />}
      <Card>
        <Table
          rowKey="id"
          dataSource={r.data?.items || []}
          loading={r.loading}
          columns={[
            { title: "账号", dataIndex: "username" },
            { title: "姓名", dataIndex: "name" },
            {
              title: "组织角色",
              dataIndex: "role",
              render: (v) => roleLabels[v],
            },
            {
              title: "状态",
              dataIndex: "enabled",
              render: (v) => (
                <Tag color={v ? "success" : "default"}>
                  {v ? "启用" : "停用"}
                </Tag>
              ),
            },
            {
              title: "操作",
              render: (_, u) => (
                <Button type="link" onClick={() => open(u)}>
                  编辑账户
                </Button>
              ),
            },
          ]}
          pagination={{ pageSize: 15 }}
        />
      </Card>
      <Modal
        open={!!editing}
        title={editing?.id ? "编辑账户" : "添加账户"}
        onCancel={() => setEditing(null)}
        onOk={save}
        confirmLoading={busy}
      >
        {error && <Alert type="error" title={error} />}
        <Form form={form} layout="vertical">
          <Form.Item
            name="username"
            label="账号"
            rules={[{ required: true, min: 2 }]}
          >
            <Input maxLength={80} autoComplete="off" />
          </Form.Item>
          <Form.Item
            name="name"
            label="姓名"
            rules={[{ required: true, min: 2 }]}
          >
            <Input maxLength={80} />
          </Form.Item>
          <Form.Item name="role" label="组织角色" rules={[{ required: true }]}>
            <Select
              options={Object.entries(roleLabels).map(([value, label]) => ({
                value,
                label,
              }))}
            />
          </Form.Item>
          <Form.Item
            name="password"
            label={editing?.id ? "重设口令（留空保留）" : "初始口令"}
            rules={[{ required: !editing?.id }, { min: 12 }]}
          >
            <Input.Password autoComplete="new-password" maxLength={72} />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
export function ProjectMembers({ workspaceId }) {
  const { user } = useAuth(),
    r = useResource(`/workspaces/${workspaceId}/members`),
    users = useResource(
      user.role === "admin" ? "/enterprise/users" : "/auth/session",
    ),
    { message } = App.useApp();
  const [selected, setSelected] = useState(null),
    [role, setRole] = useState("analyst"),
    [error, setError] = useState("");
  const save = async (id, value) => {
    try {
      await mutateResource(
        `/workspaces/${workspaceId}/members/${id}`,
        { role: value },
        null,
        "PUT",
      );
      r.refresh();
      message.success("项目权限已更新");
    } catch (e) {
      setError(e.message);
    }
  };
  return (
    <Space orientation="vertical" style={{ width: "100%" }}>
      <Alert
        type="info"
        title="只读成员查看项目；分析人员维护工作产品与辅助任务；评审人员还可建立基线、执行还原。管理员可管理所有项目。"
      />
      {(error || r.error) && <Alert type="error" title={error || r.error} />}{" "}
      {user.role === "admin" && (
        <Space wrap>
          <Select
            style={{ minWidth: 220 }}
            placeholder="选择企业账户"
            value={selected}
            onChange={setSelected}
            options={(users.data?.items || [])
              .filter((u) => u.enabled)
              .map((u) => ({
                value: u.id,
                label: `${u.name} (${u.username})`,
              }))}
          />
          <Select
            value={role}
            onChange={setRole}
            options={["viewer", "analyst", "reviewer"].map((value) => ({
              value,
              label: roleLabels[value],
            }))}
          />
          <Button
            type="primary"
            disabled={!selected}
            onClick={() => save(selected, role)}
          >
            分配项目角色
          </Button>
        </Space>
      )}
      <Table
        rowKey="user_id"
        dataSource={r.data?.items || []}
        columns={[
          { title: "成员", dataIndex: "name" },
          { title: "账号", dataIndex: "username" },
          {
            title: "项目角色",
            dataIndex: "role",
            render: (v) => roleLabels[v],
          },
          {
            title: "账户状态",
            dataIndex: "enabled",
            render: (v) => (v ? "启用" : "停用"),
          },
          {
            title: "操作",
            render: (_, u) =>
              user.role === "admin" && (
                <Button onClick={() => save(u.user_id, "none")}>
                  撤销项目访问
                </Button>
              ),
          },
        ]}
        pagination={false}
      />
    </Space>
  );
}
