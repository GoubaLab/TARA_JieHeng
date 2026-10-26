import React, { useEffect, useState } from "react";
import {
  Alert,
  Button,
  Card,
  Col,
  Form,
  Input,
  InputNumber,
  Row,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from "antd";
import { SaveOutlined } from "@ant-design/icons";
import { PageTitle, useWorkbench } from "./Workbench.jsx";
import {
  effectiveNumbering,
  formatNumber,
  numberingTypes,
} from "./numbering.js";

export default function NumberingSettings() {
  const { ws, save, message } = useWorkbench();
  const [form] = Form.useForm();
  const [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const watched = Form.useWatch([], { form, preserve: true });
  const rules = effectiveNumbering({ ...ws.numbering, ...watched });
  useEffect(() => {
    form.resetFields();
    form.setFieldsValue(effectiveNumbering(ws.numbering));
    setError("");
  }, [ws.id, ws.revision]);
  const apply = async () => {
    try {
      const values = await form.validateFields();
      const prefixes = Object.values(values.prefixes).map((v) =>
        v.toLocaleLowerCase(),
      );
      if (new Set(prefixes).size !== prefixes.length) {
        setError("不同类型请使用不同前缀，不区分大小写。");
        return;
      }
      setBusy(true);
      setError("");
      await save("configure-numbering", values);
      message.success("编号规则已应用，现有序号和关联保持不变");
    } catch (e) {
      if (!e.errorFields) setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <>
      <PageTitle
        title="编号规则"
        description="为不同工作产品设置容易识别的编号，统一用于画布、清单和报告。"
        extra={
          <Button
            type="primary"
            icon={<SaveOutlined />}
            loading={busy}
            onClick={apply}
          >
            保存编号规则
          </Button>
        }
      />
      <Alert
        className="section-card"
        type="info"
        showIcon
        title="仅应用于当前项目"
        description="修改前缀、分隔符或位数会更新已有记录的显示编号，原序号和关联保留。删除记录不回收序号；冻结报告保留当时的编号规则。"
      />
      {error && (
        <Alert className="section-card" type="error" showIcon title={error} />
      )}
      <Form
        form={form}
        layout="vertical"
        initialValues={effectiveNumbering(ws.numbering)}
      >
        <Card title="编号格式" className="section-card">
          <Row gutter={24} align="middle">
            <Col xs={24} md={7}>
              <Form.Item
                name="separator"
                label="分隔符"
                rules={[{ required: true }]}
              >
                <Select
                  options={[
                    { value: "-", label: "短横线  -" },
                    { value: "_", label: "下划线  _" },
                    { value: ".", label: "点号  ." },
                    { value: "/", label: "斜线  /" },
                  ]}
                />
              </Form.Item>
            </Col>
            <Col xs={24} md={7}>
              <Form.Item
                name="padding"
                label="数字最少位数"
                rules={[{ required: true }]}
              >
                <InputNumber
                  min={2}
                  max={6}
                  precision={0}
                  style={{ width: "100%" }}
                />
              </Form.Item>
            </Col>
            <Col xs={24} md={10}>
              <Space orientation="vertical">
                <Typography.Text type="secondary">
                  示例：资产编号
                </Typography.Text>
                <Tag color="blue" style={{ fontSize: 18, padding: "6px 14px" }}>
                  {formatNumber(rules, "asset", 1)}
                </Tag>
              </Space>
            </Col>
          </Row>
          <Typography.Text type="secondary">
            序号超过设定位数时自动扩展，不截断。前缀支持中英文、数字、下划线和短横线，最长
            12 个字符。
          </Typography.Text>
        </Card>
        <Card title="类型与前缀">
          <Table
            rowKey="key"
            pagination={false}
            size="middle"
            dataSource={numberingTypes.map(([key, label]) => ({ key, label }))}
            columns={[
              { title: "记录类型", dataIndex: "label", width: "30%" },
              {
                title: "自定义前缀",
                width: "40%",
                render: (_, row) => (
                  <Form.Item
                    name={["prefixes", row.key]}
                    style={{ marginBottom: 0 }}
                    rules={[
                      { required: true, message: "请填写前缀" },
                      {
                        pattern: /^[\p{L}\p{N}][\p{L}\p{N}_-]{0,11}$/u,
                        message:
                          "仅限文字、数字、下划线和短横线，以文字或数字开头",
                      },
                    ]}
                  >
                    <Input
                      aria-label={`${row.label}编号前缀`}
                      maxLength={12}
                      style={{ maxWidth: 240 }}
                    />
                  </Form.Item>
                ),
              },
              {
                title: "编号预览",
                render: (_, row) => (
                  <Typography.Text code>
                    {formatNumber(rules, row.key, 1)}
                  </Typography.Text>
                ),
              },
            ]}
          />
        </Card>
      </Form>
    </>
  );
}
