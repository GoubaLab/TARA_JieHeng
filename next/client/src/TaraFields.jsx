import React from "react";
import { FunctionReferences } from "./FunctionCatalog.jsx";
import { Alert, Col, Form, Input, InputNumber, Row, Select } from "antd";
import { propOptions } from "./api.js";
const options = (records, text) =>
  (records || []).map((r) => ({
    value: r.id,
    label: `${r.source_code || r.id} · ${r[text] || ""}`,
  }));
export function TaraFields({ kind, ws }) {
  const common = ["asset", "scenario", "requirement"].includes(kind);
  return (
    <>
      {common && (
        <Form.Item name="source_code" label="业务编号">
          <Input maxLength={100} />
        </Form.Item>
      )}
      {kind === "asset" && (
        <>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="asset_type" label="资产类型">
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="group" label="分组">
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="function_refs" label="关联功能">
            <FunctionReferences ws={ws} />
          </Form.Item>
        </>
      )}
      {kind === "scenario" && (
        <>
          <Form.Item
            name="asset_ids"
            label="全部关联资产"
            extra="可关联多个资产；下方主资产也会计入追溯。"
          >
            <Select
              mode="multiple"
              optionFilterProp="label"
              options={options(ws.assets, "name")}
            />
          </Form.Item>
          <Form.Item name="properties" label="全部受损安全属性">
            <Select mode="multiple" options={propOptions} />
          </Form.Item>
          <Form.Item
            name="damage_ids"
            label="引用损害场景"
            extra="引用后，影响取关联损害场景的最高等级；请在 TARA 工作产品中编辑损害后果和评级。"
          >
            <Select
              mode="multiple"
              optionFilterProp="label"
              options={options(ws.damage_scenarios, "description")}
            />
          </Form.Item>
          <Form.Item name="stride" label="STRIDE 分类">
            <Input />
          </Form.Item>
          <Form.Item name="cause" label="攻击原因">
            <Input.TextArea rows={3} />
          </Form.Item>
        </>
      )}
      {["requirement", "decision"].includes(kind) && (
        <Form.Item name="goal_ids" label="关联安全目标">
          <Select
            mode="multiple"
            optionFilterProp="label"
            options={options(ws.security_goals, "statement")}
          />
        </Form.Item>
      )}
      {kind === "requirement" && (
        <>
          <Form.Item name="scenario_ids" label="全部关联威胁场景">
            <Select
              mode="multiple"
              optionFilterProp="label"
              options={options(ws.scenarios, "title")}
            />
          </Form.Item>
          <Form.Item name="customer_refs" label="客户需求编号">
            <Select mode="tags" />
          </Form.Item>
        </>
      )}
      {["scenario", "requirement", "decision"].includes(kind) && (
        <Form.Item name="cal" label="CAL">
          <InputNumber min={1} max={4} precision={0} />
        </Form.Item>
      )}
      {kind === "decision" && (
        <>
          <Form.Item name="controls" label="控制措施">
            <Input.TextArea rows={4} />
          </Form.Item>
          <Form.Item name="acceptance" label="原接受结论／当前接受条件">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item name="acceptance_rationale" label="接受依据与前提">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="customer_confirmation" label="客户确认">
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="confirmed_at" label="确认日期">
                <Input placeholder="YYYY-MM-DD" />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="declarant" label="声明人">
            <Input />
          </Form.Item>
          <Form.Item name="residual_status" label="残余风险性质">
            <Select
              options={[
                { value: "target", label: "设计目标 · 待验证" },
                { value: "validated", label: "已验证并确认" },
              ]}
            />
          </Form.Item>
          <Alert
            type="info"
            title="设计目标须完成关联需求验证后才能改为已验证；客户确认和签署仍需实际完成。"
          />
        </>
      )}
    </>
  );
}
