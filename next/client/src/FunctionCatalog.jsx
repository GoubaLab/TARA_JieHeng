import React, { useState } from "react";
import {
  Alert,
  Button,
  Card,
  Drawer,
  Empty,
  Form,
  Input,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from "antd";
import { useWorkbench } from "./Workbench.jsx";
import { functionCatalog } from "./tara-guidance.js";
const { Text, Paragraph } = Typography;
export function FunctionReferences({ ws, value = [], onChange }) {
  const catalog = functionCatalog(ws),
    selected = value.map(
      (id) =>
        catalog.find((f) => f.id === id) || {
          id,
          name: "功能内容未定义",
          description: "请在相关项定义的功能目录补充此编号的具体内容。",
          missing: true,
        },
    );
  return (
    <Space orientation="vertical" style={{ width: "100%" }}>
      <Select
        mode="multiple"
        style={{ width: "100%" }}
        value={value}
        onChange={onChange}
        showSearch
        optionFilterProp="label"
        placeholder="按编号或功能名称搜索"
        options={[...catalog, ...selected.filter((f) => f.missing)].map(
          (f) => ({
            value: f.id,
            label: `${f.id} · ${f.name}${f.group ? "（范围定义）" : ""}`,
          }),
        )}
      />
      {!catalog.length && (
        <Text type="secondary">
          请先在“相关项定义 → 功能目录”建立功能，填写编号、名称和具体内容。
        </Text>
      )}
      {selected.map((f) => (
        <Card size="small" key={f.id} title={`${f.id} · ${f.name}`}>
          <Paragraph style={{ marginBottom: 0 }}>
            {f.description || f.name}
          </Paragraph>
          {f.group && (
            <Text type="secondary">
              {f.group}{" "}
              共用的范围说明；原始资料未逐项定义该编号，需按实际需求补充。
            </Text>
          )}
          {f.missing && <Tag color="warning">待补充</Tag>}
        </Card>
      ))}
    </Space>
  );
}
export default function FunctionCatalogPanel() {
  const { ws, save, message } = useWorkbench(),
    [editing, setEditing] = useState(null),
    [form] = Form.useForm(),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const rows = functionCatalog(ws),
    readOnly = ws.access_role === "viewer";
  const edit = (r) => {
    setError("");
    setEditing(r || {});
    form.resetFields();
    form.setFieldsValue(r || {});
  };
  const submit = async () => {
    try {
      const v = await form.validateFields();
      setBusy(true);
      setError("");
      if (!editing?.id && rows.some((f) => f.id === v.id))
        throw Error("功能编号已存在，请编辑已有功能");
      await save("define-item", {
        ...ws.item,
        functions: [
          ...(ws.item.functions || []).filter((f) => f.id !== editing.id),
          v,
        ],
      });
      message.success("功能内容已保存");
      setEditing(null);
    } catch (e) {
      if (!e.errorFields) setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <Card
      className="section-card"
      title="功能目录"
      extra={
        <Button disabled={readOnly} onClick={() => edit()}>
          新增功能
        </Button>
      }
    >
      <Paragraph type="secondary">
        资产和损害场景引用此目录。编号与功能内容同时显示，范围定义可进一步补充为单项功能。
      </Paragraph>
      <Table
        rowKey="id"
        size="small"
        pagination={{ pageSize: 8 }}
        dataSource={rows}
        locale={{ emptyText: <Empty description="尚未建立功能目录" /> }}
        columns={[
          { title: "编号", dataIndex: "id", width: 120 },
          { title: "功能", dataIndex: "name", width: 180 },
          {
            title: "具体内容",
            dataIndex: "description",
            render: (v, r) => (
              <>
                <div style={{ whiteSpace: "pre-wrap" }}>{v}</div>
                {r.group && <Tag>范围定义 {r.group}</Tag>}
              </>
            ),
          },
          {
            title: "操作",
            width: 90,
            render: (_, r) => (
              <Button type="link" disabled={readOnly} onClick={() => edit(r)}>
                补充／编辑
              </Button>
            ),
          },
        ]}
      />
      <Drawer
        open={!!editing}
        title="功能定义"
        size={640}
        onClose={() => !busy && setEditing(null)}
        extra={
          <Button type="primary" loading={busy} onClick={submit}>
            保存
          </Button>
        }
      >
        {error && <Alert type="error" title={error} className="section-card" />}
        <Form form={form} layout="vertical">
          <Form.Item
            name="id"
            label="功能编号"
            rules={[{ required: true, whitespace: true }]}
          >
            <Input disabled={!!editing?.id} maxLength={100} />
          </Form.Item>
          <Form.Item
            name="name"
            label="功能名称"
            rules={[{ required: true, whitespace: true }]}
          >
            <Input maxLength={300} />
          </Form.Item>
          <Form.Item
            name="description"
            label="功能内容"
            rules={[{ required: true, whitespace: true }]}
          >
            <Input.TextArea rows={8} maxLength={6000} />
          </Form.Item>
        </Form>
      </Drawer>
    </Card>
  );
}
