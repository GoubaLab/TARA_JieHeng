import React, { useEffect, useRef, useState } from "react";
import { Alert, App, Form, Input, Modal, Spin } from "antd";
import { request } from "./api.js";

export default function WorkspaceProfileEditor({
  workspaceID,
  onClose,
  onSaved,
}) {
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [workspace, setWorkspace] = useState(null);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const attempt = useRef(null);

  useEffect(() => {
    setWorkspace(null);
    setError("");
    attempt.current = null;
    if (!workspaceID) return;
    const controller = new AbortController();
    let active = true;
    setLoading(true);
    form.resetFields();
    request(`/workspaces/${encodeURIComponent(workspaceID)}`, {
      signal: controller.signal,
    })
      .then((value) => {
        if (!active) return;
        setWorkspace(value);
        form.setFieldsValue({
          title: value.title,
          description: value.description,
        });
      })
      .catch((e) => {
        if (active) setError(e.message);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
      controller.abort();
    };
  }, [workspaceID, form]);

  const save = async () => {
    if (!workspace || workspace.access_role === "viewer" || saving) return;
    try {
      const values = await form.validateFields();
      const body = JSON.stringify(values);
      if (attempt.current?.body !== body)
        attempt.current = { body, key: crypto.randomUUID() };
      setSaving(true);
      setError("");
      await request(`/workspaces/${encodeURIComponent(workspace.id)}/profile`, {
        method: "PUT",
        headers: {
          "If-Match": `"${workspace.revision}"`,
          "Idempotency-Key": attempt.current.key,
        },
        body,
      });
      await onSaved();
      message.success("项目基本信息已保存");
      onClose();
    } catch (e) {
      if (!e.errorFields)
        setError(
          e.status === 412 ? "项目已更新，请关闭此窗口后重新编辑。" : e.message,
        );
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal
      title="项目基本信息"
      open={!!workspaceID}
      onCancel={() => !saving && onClose()}
      onOk={save}
      okText="保存"
      cancelText="取消"
      confirmLoading={saving}
      okButtonProps={{
        disabled: loading || !workspace || workspace.access_role === "viewer",
      }}
      cancelButtonProps={{ disabled: saving }}
      closable={!saving}
      maskClosable={!saving}
      destroyOnHidden
    >
      {error && (
        <Alert
          type="error"
          title={error}
          showIcon
          style={{ marginBottom: 16 }}
        />
      )}
      {workspace?.access_role === "viewer" && (
        <Alert
          type="info"
          title="当前项目为只读权限"
          style={{ marginBottom: 16 }}
        />
      )}
      <Spin spinning={loading}>
        <Form
          form={form}
          layout="vertical"
          disabled={loading || saving || workspace?.access_role === "viewer"}
        >
          <Form.Item
            name="title"
            label="项目名称"
            rules={[
              { required: true, whitespace: true, message: "请填写项目名称" },
            ]}
          >
            <Input maxLength={200} />
          </Form.Item>
          <Form.Item name="description" label="项目说明">
            <Input.TextArea
              rows={4}
              maxLength={4000}
              showCount
              placeholder="填写产品范围和分析目的"
            />
          </Form.Item>
        </Form>
      </Spin>
    </Modal>
  );
}
