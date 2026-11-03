import React, { useEffect, useRef, useState } from "react";
import {
  Alert,
  AutoComplete,
  Button,
  Card,
  Col,
  Form,
  Input,
  Modal,
  Popconfirm,
  Row,
  Select,
  Space,
  Table,
  Tag,
} from "antd";
import { PageTitle, useWorkbench } from "./Workbench.jsx";
import {
  blankCommunication,
  communicationForSave,
  media,
  ranges,
  technologies,
  frequencyBands,
} from "./communication.js";
const required = [{ required: true, whitespace: true, message: "请填写此项" }];
export default function CommunicationSettings() {
  const { ws, save, message } = useWorkbench();
  const [presets, setPresets] = useState([]),
    [editing, setEditing] = useState(null),
    [dirty, setDirty] = useState(false),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const pending = useRef(false),
    [form] = Form.useForm();
  const medium =
    Form.useWatch(["communication", "medium"], form) || "unspecified";
  const readOnly = ws.access_role === "viewer";
  useEffect(() => {
    setPresets(structuredClone(ws.communication_settings?.presets || []));
    setDirty(false);
    setError("");
    setEditing(null);
  }, [ws.id, ws.revision]);
  useEffect(() => {
    if (!dirty) return;
    const warn = (e) => {
      e.preventDefault();
      e.returnValue = "";
    };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);
  const open = (record) => {
    form.resetFields();
    const item = record || {
      id: crypto.randomUUID(),
      name: "",
      protocol: "",
      communication: blankCommunication(),
    };
    setEditing(item);
    form.setFieldsValue(structuredClone(item));
  };
  const confirm = async () => {
    try {
      const values = await form.validateFields();
      const communication =
        communicationForSave({
          ...values.communication,
          security_properties: [],
        }) || {};
      delete communication.security_properties;
      const item = {
        id: editing.id,
        name: values.name.trim(),
        protocol: values.protocol || "",
        communication,
      };
      setPresets((items) =>
        items.some((r) => r.id === item.id)
          ? items.map((r) => (r.id === item.id ? item : r))
          : [...items, item],
      );
      setDirty(true);
      setEditing(null);
    } catch (e) {
      if (!e.errorFields) setError(e.message);
    }
  };
  const persist = async () => {
    if (pending.current || readOnly) return;
    pending.current = true;
    setBusy(true);
    setError("");
    try {
      await save("configure-communication", { presets });
      message.success("通信预设已保存，已有数据流保持不变");
    } catch (e) {
      setError(e.message);
    } finally {
      pending.current = false;
      setBusy(false);
    }
  };
  const options = (items) => items.map((value) => ({ value }));
  return (
    <>
      <PageTitle
        title="通信预设"
        description="保存常用通信技术、频段和上层协议，在数据流属性中选用。"
        extra={
          <Space>
            {dirty && <Tag color="orange">有未保存的更改</Tag>}
            <Button
              type="primary"
              disabled={readOnly || !dirty}
              loading={busy}
              onClick={persist}
            >
              保存通信预设
            </Button>
          </Space>
        }
      />
      <Alert
        className="section-card"
        type="info"
        showIcon
        title="项目内复用，数据流独立维护"
        description="在数据流属性中应用预设后，仍需保存架构。修改或移除预设不会改变已有数据流；每条数据流的安全属性需单独确认。"
      />
      {error && (
        <Alert className="section-card" type="error" showIcon title={error} />
      )}
      <Card
        title={`通信预设（${presets.length} / 100）`}
        extra={
          <Button
            disabled={readOnly || busy || presets.length >= 100}
            onClick={() => open()}
          >
            新增预设
          </Button>
        }
      >
        <Table
          rowKey="id"
          dataSource={presets}
          scroll={{ x: 850 }}
          locale={{
            emptyText:
              "暂无通信预设，可添加项目常用的有线、无线或逻辑连接参数。",
          }}
          columns={[
            { title: "预设名称", dataIndex: "name" },
            {
              title: "传输方式",
              render: (_, r) => media[r.communication?.medium] || "未分类",
            },
            {
              title: "通信技术",
              render: (_, r) => r.communication?.technology || "—",
            },
            {
              title: "频段 / 频率",
              render: (_, r) => r.communication?.frequency_band || "—",
            },
            { title: "上层协议 / 通道", dataIndex: "protocol" },
            {
              title: "操作",
              render: (_, r) => (
                <Space>
                  <Button
                    type="link"
                    disabled={readOnly || busy}
                    onClick={() => open(r)}
                  >
                    编辑
                  </Button>
                  <Popconfirm
                    title="移除此预设？"
                    description="保存设置后生效，已有数据流保持不变。"
                    onConfirm={() => {
                      setPresets((items) => items.filter((x) => x.id !== r.id));
                      setDirty(true);
                    }}
                  >
                    <Button type="link" danger disabled={readOnly || busy}>
                      移除
                    </Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Card>
      <Modal
        title="通信预设"
        open={!!editing}
        onCancel={() => setEditing(null)}
        onOk={confirm}
        okText="加入预设列表"
        cancelText="取消"
        width={720}
        okButtonProps={{ disabled: readOnly }}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="预设名称" rules={required}>
            <Input
              maxLength={200}
              placeholder="例如：设备公网遥测、园区无线控制"
            />
          </Form.Item>
          <Row gutter={16}>
            <Col span={10}>
              <Form.Item name={["communication", "medium"]} label="传输方式">
                <Select
                  options={Object.entries(media).map(([value, label]) => ({
                    value,
                    label,
                  }))}
                  onChange={(v) => {
                    if (v !== "wireless")
                      form.setFieldValue(
                        ["communication", "frequency_band"],
                        "",
                      );
                  }}
                />
              </Form.Item>
            </Col>
            <Col span={14}>
              <Form.Item
                name={["communication", "technology"]}
                label="通信技术"
              >
                <AutoComplete
                  options={options(technologies[medium] || [])}
                  placeholder="选择或输入自定义技术"
                >
                  <Input maxLength={200} />
                </AutoComplete>
              </Form.Item>
            </Col>
          </Row>
          {medium === "wireless" && (
            <Form.Item
              name={["communication", "frequency_band"]}
              label="频段 / 频率"
              extra="填写实际部署使用的频段名称、频率或范围，支持自定义。"
            >
              <AutoComplete
                options={options(frequencyBands)}
                placeholder="例如：n78、2.4 GHz、433 MHz"
              >
                <Input maxLength={200} />
              </AutoComplete>
            </Form.Item>
          )}
          <Row gutter={16}>
            <Col span={10}>
              <Form.Item name={["communication", "range"]} label="覆盖范围">
                <Select
                  options={Object.entries(ranges).map(([value, label]) => ({
                    value,
                    label,
                  }))}
                />
              </Form.Item>
            </Col>
            <Col span={14}>
              <Form.Item name={["communication", "purpose"]} label="用途">
                <Input maxLength={4000} placeholder="例如：遥测、指令、升级" />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="protocol" label="上层协议 / 通道">
            <Input
              maxLength={200}
              placeholder="例如：MQTT/TLS、HTTPS、自定义协议"
            />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
