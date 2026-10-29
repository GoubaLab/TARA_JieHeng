import NumberingSettings from "./NumberingSettings.jsx";
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
  Tabs,
  Typography,
  Tag,
} from "antd";
import { SaveOutlined } from "@ant-design/icons";
import { PageTitle, Risk, useWorkbench } from "./Workbench.jsx";
import {
  dimensionKeys,
  dimensionCodes,
  effectiveMethod,
  methodErrors,
  previewRisk,
  previewScore,
  coefficientMaximum,
} from "./method-config.js";
const { Text, Paragraph } = Typography;
const required = [{ required: true, whitespace: true, message: "请填写此项" }];
function Levels({ field, rows, start }) {
  return (
    <div className="method-levels">
      {rows.map((_, i) => (
        <Row gutter={16} key={i} align="top">
          <Col xs={24} md={8}>
            <Form.Item
              name={[field, i, "label"]}
              label={`等级 ${i + start} · 名称`}
              rules={required}
            >
              <Input maxLength={100} />
            </Form.Item>
          </Col>
          <Col xs={24} md={16}>
            <Form.Item name={[field, i, "description"]} label="判定条件与示例">
              <Input.TextArea
                rows={2}
                maxLength={2000}
                placeholder="填写组织的量化阈值、适用条件或示例，供评估时参考"
              />
            </Form.Item>
          </Col>
        </Row>
      ))}
    </div>
  );
}
export function MethodPreview({ method }) {
  const m = effectiveMethod(method);
  const coefficient = m.impact_aggregation === "coefficient_sum";
  const samples = coefficient
    ? [0, 1, 2, 3].map((i) => ({
        value: Number((((coefficientMaximum(m) || 0) * i) / 3).toPrecision(12)),
        label: String(
          Number((((coefficientMaximum(m) || 0) * i) / 3).toPrecision(12)),
        ),
      }))
    : m.impact_levels.map((v, i) => ({ ...v, value: i }));
  return (
    <div className="matrix-preview">
      <div>{coefficient ? "总分 / 可行性" : "影响 / 可行性"}</div>
      {m.feasibility_levels.map((v, i) => (
        <div key={i}>{v.label}</div>
      ))}
      {samples.map((v, i) => (
        <React.Fragment key={i}>
          <div>{v.label}</div>
          {m.feasibility_levels.map((_, j) => (
            <div key={j}>
              <Risk level={previewRisk(m, v.value, j + 1)} method={m} />
            </div>
          ))}
        </React.Fragment>
      ))}
    </div>
  );
}
export default function AssessmentSettings() {
  return (
    <Tabs
      items={[
        { key: "method", label: "评估算法", children: <MethodSettings /> },
        {
          key: "numbering",
          label: "编号规则",
          children: <NumberingSettings />,
        },
      ]}
    />
  );
}
function MethodSettings() {
  const { ws, save, message } = useWorkbench();
  const [form] = Form.useForm();
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const watched = Form.useWatch([], { form, preserve: true });
  const method = effectiveMethod({ ...ws.method, ...watched });
  const coefficient = method.impact_aggregation === "coefficient_sum";
  const banded = method.impact_aggregation === "coefficient_bands";
  const [sampleImpact, setSampleImpact] = useState(5.2);
  const [sampleFeasibility, setSampleFeasibility] = useState(2);
  useEffect(() => {
    form.resetFields();
    form.setFieldsValue(effectiveMethod(ws.method));
    setError("");
  }, [ws.id, ws.revision]);
  const apply = async () => {
    try {
      await form.validateFields();
      const payload = effectiveMethod(form.getFieldsValue(true));
      const problems = methodErrors(payload);
      if (problems.length) {
        setError(problems.join(" "));
        return;
      }
      setBusy(true);
      setError("");
      await save("adopt-method", payload);
      message.success("评估算法已保存，风险与待办已重新计算");
    } catch (e) {
      if (!e.errorFields) setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  const total = dimensionKeys.reduce(
    (sum, key) => sum + (Number(method.impact_weights[key]) || 0),
    0,
  );
  return (
    <>
      <PageTitle
        title="评估算法"
        description="为当前项目配置影响维度、评分依据和风险算法。"
        extra={
          <Button
            type="primary"
            icon={<SaveOutlined />}
            loading={busy}
            onClick={apply}
          >
            保存并应用
          </Button>
        }
      />
      <Alert
        type="info"
        showIcon
        className="section-card"
        title="配置仅作用于当前项目"
        description="保存后重算风险并标记需复核的处置，冻结基线保持原方法。默认参数为配置示例，应按组织评估准则确认。"
      />
      {error && (
        <Alert type="error" showIcon title={error} className="section-card" />
      )}
      <Form
        form={form}
        layout="vertical"
        requiredMark="optional"
        initialValues={effectiveMethod(ws.method)}
      >
        <Row gutter={[16, 16]}>
          <Col xs={24} xl={16}>
            <Card title="方法与适用依据" className="section-card">
              <Form.Item name="name" label="方法名称" rules={required}>
                <Input maxLength={200} />
              </Form.Item>
              <Form.Item
                name="rationale"
                label="适用依据与说明"
                rules={required}
              >
                <Input.TextArea rows={3} maxLength={6000} />
              </Form.Item>
            </Card>
            <Card title="评分规则">
              <Tabs
                items={[
                  {
                    key: "impact",
                    label: "影响维度 · SFOP",
                    forceRender: true,
                    children: (
                      <>
                        <Form.Item
                          name="impact_aggregation"
                          label="综合影响算法"
                        >
                          <Select
                            onChange={(v) => {
                              if (v === "coefficient_bands")
                                form.setFieldsValue({
                                  risk_calculation: "matrix",
                                  impact_scale: method.impact_scale || [
                                    0, 10, 100, 1000,
                                  ],
                                  impact_thresholds:
                                    method.impact_thresholds || [20, 100, 1000],
                                  severe_impact_floor:
                                    method.severe_impact_floor ?? 2,
                                  attack_potential_thresholds:
                                    method.attack_potential_thresholds || [
                                      10, 14, 20,
                                    ],
                                });
                              if (v === "coefficient_sum")
                                form.setFieldValue(
                                  "risk_calculation",
                                  "product",
                                );
                            }}
                            options={[
                              {
                                value: "coefficient_bands",
                                label: "原始取值 × 系数 → 影响分档 → 风险矩阵",
                              },
                              {
                                value: "maximum",
                                label: "最高维度（保守评估）",
                              },
                              {
                                value: "weighted_mean",
                                label: "加权平均（向上取整）",
                              },
                              {
                                value: "coefficient_sum",
                                label: "系数求和（保留原始总分）",
                              },
                            ]}
                          />
                        </Form.Item>
                        <Paragraph type="secondary">
                          默认四维分别评为
                          0–3。最高值法取最大值；加权平均法归一化并向上取整；系数法直接求和，保留小数、不归一化。权重或系数为
                          0
                          的维度可留空。原始取值分档法使用下方独立量表，四维均需评定。
                        </Paragraph>
                        {dimensionKeys.map((key) => (
                          <Row gutter={16} key={key}>
                            <Col span={14}>
                              <Form.Item
                                name={["impact_dimensions", key]}
                                label={`维度 ${dimensionCodes[key]} · 显示名称`}
                                rules={required}
                              >
                                <Input maxLength={100} />
                              </Form.Item>
                            </Col>
                            <Col span={10}>
                              <Form.Item
                                name={[
                                  coefficient || banded
                                    ? "impact_coefficients"
                                    : "impact_weights",
                                  key,
                                ]}
                                label={`维度 ${dimensionCodes[key]} · ${coefficient || banded ? "乘数系数" : "相对权重"}`}
                                rules={[
                                  {
                                    required: true,
                                    type: "number",
                                    min: 0,
                                    max: 100,
                                    message: "填写 0–100 的有效数值",
                                  },
                                ]}
                              >
                                <InputNumber
                                  min={0}
                                  max={100}
                                  precision={
                                    coefficient || banded ? undefined : 0
                                  }
                                  step={coefficient || banded ? 0.1 : 1}
                                  disabled={
                                    method.impact_aggregation === "maximum"
                                  }
                                  style={{ width: "100%" }}
                                />
                              </Form.Item>
                            </Col>
                          </Row>
                        ))}
                        {method.impact_aggregation === "weighted_mean" && (
                          <Alert
                            type={total ? "info" : "error"}
                            title={`权重合计 ${total}；无需等于 100，系统会自动归一化。`}
                            className="section-card"
                          />
                        )}
                        {coefficient && (
                          <Alert
                            type="info"
                            className="section-card"
                            title={`SFOP = S×${method.impact_coefficients.safety} + F×${method.impact_coefficients.financial} + O×${method.impact_coefficients.operational} + P×${method.impact_coefficients.privacy}`}
                            description={`总分范围 0–${coefficientMaximum(method) ?? "待填写"}。风险分值 = SFOP总分 × 可行性；总分不加1、不归一化。`}
                          />
                        )}
                        {banded && (
                          <Card
                            size="small"
                            title="原始量表与分档阈值"
                            className="section-card"
                          >
                            <Row gutter={12}>
                              {[0, 1, 2, 3].map((i) => (
                                <Col span={6} key={i}>
                                  <Form.Item
                                    name={["impact_scale", i]}
                                    label={`等级 ${i} 原始值`}
                                    rules={[{ required: true }]}
                                  >
                                    <InputNumber min={0} max={1000000} />
                                  </Form.Item>
                                </Col>
                              ))}
                            </Row>
                            <Row gutter={12}>
                              {[0, 1, 2].map((i) => (
                                <Col span={8} key={i}>
                                  <Form.Item
                                    name={["impact_thresholds", i]}
                                    label={`进入等级 ${i + 1} 的下限`}
                                    rules={[{ required: true }]}
                                  >
                                    <InputNumber min={0} />
                                  </Form.Item>
                                </Col>
                              ))}
                            </Row>
                            <Form.Item
                              name="severe_impact_floor"
                              label="任一维度严重时的最低综合等级"
                            >
                              <Select
                                options={method.impact_levels.map(
                                  (v, value) => ({ value, label: v.label }),
                                )}
                              />
                            </Form.Item>
                            <Text type="secondary">
                              SFOP
                              原始值乘对应系数求和，按以上下限分档后查风险矩阵。边界值归入较高等级。
                            </Text>
                          </Card>
                        )}
                        <Levels
                          field="impact_levels"
                          rows={method.impact_levels}
                          start={0}
                        />
                      </>
                    ),
                  },
                  {
                    key: "feasibility",
                    label: "攻击可行性",
                    forceRender: true,
                    children: (
                      <>
                        <Paragraph type="secondary">
                          可行性由评估人员依据攻击条件和证据选择 1–4
                          级；数值越高代表越容易实现攻击。可在这里维护组织的分级准则。
                        </Paragraph>
                        {method.attack_potential_thresholds && (
                          <Card
                            size="small"
                            title="五因子攻击潜力分档"
                            className="section-card"
                          >
                            <Row gutter={12}>
                              {[0, 1, 2].map((i) => (
                                <Col span={8} key={i}>
                                  <Form.Item
                                    name={["attack_potential_thresholds", i]}
                                    label={`进入${["中", "低", "极低"][i]}可行性的 AP 下限`}
                                    rules={[{ required: true }]}
                                  >
                                    <InputNumber
                                      min={1}
                                      max={500}
                                      precision={0}
                                    />
                                  </Form.Item>
                                </Col>
                              ))}
                            </Row>
                            <Text type="secondary">
                              AP = ET + Ex + KoI + WoO + Eq；AP
                              越高，攻击越困难。
                            </Text>
                          </Card>
                        )}
                        <Levels
                          field="feasibility_levels"
                          rows={method.feasibility_levels}
                          start={1}
                        />
                      </>
                    ),
                  },
                  {
                    key: "risk",
                    label: "风险计算与分级",
                    forceRender: true,
                    children: (
                      <>
                        <Form.Item name="risk_calculation" label="风险算法">
                          <Select
                            options={[
                              {
                                value: "matrix",
                                label: "4 × 4 风险矩阵",
                                disabled: coefficient,
                              },
                              {
                                value: "product",
                                disabled: banded,
                                label: "影响 × 可行性（阈值分级）",
                              },
                            ]}
                          />
                        </Form.Item>
                        <div
                          style={{
                            display:
                              method.risk_calculation === "matrix"
                                ? "block"
                                : "none",
                          }}
                        >
                          <Paragraph type="secondary">
                            每格为 1–5
                            级风险；影响或可行性增加时，风险不能降低。
                          </Paragraph>
                          <div className="matrix-editor">
                            <div>影响 / 可行性</div>
                            {method.feasibility_levels.map((v, i) => (
                              <div key={i}>{v.label}</div>
                            ))}
                            {method.impact_levels.map((v, i) => (
                              <React.Fragment key={i}>
                                <div>{v.label}</div>
                                {method.feasibility_levels.map((f, j) => (
                                  <Form.Item
                                    key={j}
                                    name={["matrix", i, j]}
                                    rules={[
                                      {
                                        required: true,
                                        type: "integer",
                                        min: 1,
                                        max: 5,
                                        message: "整数 1–5",
                                      },
                                    ]}
                                  >
                                    <InputNumber
                                      aria-label={`${v.label}影响、${f.label}可行性的风险等级`}
                                      min={1}
                                      max={5}
                                      precision={0}
                                      style={{ width: "100%" }}
                                    />
                                  </Form.Item>
                                ))}
                              </React.Fragment>
                            ))}
                          </div>
                        </div>
                        <div
                          style={{
                            display:
                              method.risk_calculation === "product"
                                ? "block"
                                : "none",
                          }}
                        >
                          <Paragraph type="secondary">
                            {coefficient
                              ? `分值 = SFOP总分 × 可行性，范围 0–${(coefficientMaximum(method) || 0) * 4}，保留小数。`
                              : "分值 =（影响等级 + 1）× 可行性，范围 1–16。"}{" "}
                            分值小于等于第一个阈值为 1
                            级，之后依次分段，超过最后一个阈值为 5 级。4
                            个阈值必须递增；默认阈值只是组织示例。
                          </Paragraph>
                          <Row gutter={12}>
                            {method.product_thresholds.map((_, i) => (
                              <Col span={6} key={i}>
                                <Form.Item
                                  name={["product_thresholds", i]}
                                  label={`${i + 1} 级上限`}
                                  rules={[
                                    {
                                      required: true,
                                      type: coefficient ? "number" : "integer",
                                      min: coefficient ? 0 : 1,
                                      max: coefficient ? 4800 : 15,
                                      message: coefficient
                                        ? "数值 0–4800"
                                        : "整数 1–15",
                                    },
                                  ]}
                                >
                                  <InputNumber
                                    min={coefficient ? 0 : 1}
                                    max={coefficient ? 4800 : 15}
                                    precision={
                                      coefficient || banded ? undefined : 0
                                    }
                                    step={coefficient || banded ? 0.1 : 1}
                                    style={{ width: "100%" }}
                                  />
                                </Form.Item>
                              </Col>
                            ))}
                          </Row>
                        </div>
                        <Levels
                          field="risk_levels"
                          rows={method.risk_levels}
                          start={1}
                        />
                      </>
                    ),
                  },
                ]}
              />
            </Card>
          </Col>
          <Col xs={24} xl={8}>
            <Card
              title="当前配置预览"
              className="section-card"
              extra={
                <Tag color={ws.method_confirmed ? "success" : "warning"}>
                  {ws.method_confirmed ? "已确认" : "待确认"}
                </Tag>
              }
            >
              <Paragraph>
                <Text strong>
                  {coefficient
                    ? "系数直接求和 · 保留小数"
                    : banded
                      ? "原始取值 × 系数 → 影响分档"
                      : method.impact_aggregation === "weighted_mean"
                        ? "加权平均 → 向上取整"
                        : "取四维影响最高值"}
                </Text>
                <br />
                <Text type="secondary">
                  {method.risk_calculation === "product"
                    ? "乘积后按阈值分级"
                    : "按矩阵查得风险等级"}
                </Text>
              </Paragraph>
              <MethodPreview method={method} />
              {coefficient && (
                <div style={{ marginTop: 16 }}>
                  <Paragraph type="secondary">
                    上表为总分范围示意。可在此验证任意总分与可行性组合：
                  </Paragraph>
                  <Space wrap>
                    <InputNumber
                      aria-label="示例SFOP总分"
                      value={sampleImpact}
                      onChange={setSampleImpact}
                      min={0}
                      max={coefficientMaximum(method) || 0}
                      step={0.1}
                    />
                    <Select
                      aria-label="示例可行性"
                      value={sampleFeasibility}
                      onChange={setSampleFeasibility}
                      options={method.feasibility_levels.map((v, i) => ({
                        value: i + 1,
                        label: `${i + 1} · ${v.label}`,
                      }))}
                    />
                  </Space>
                  <Paragraph style={{ marginTop: 12 }}>
                    风险分值：
                    {previewRisk(method, sampleImpact, sampleFeasibility) ==
                    null
                      ? "待填写"
                      : previewScore(
                          method,
                          sampleImpact,
                          sampleFeasibility,
                        )}{" "}
                    <Risk
                      method={method}
                      level={previewRisk(
                        method,
                        sampleImpact,
                        sampleFeasibility,
                      )}
                    />
                  </Paragraph>
                </div>
              )}
              <Paragraph type="secondary" style={{ marginTop: 16 }}>
                预览随编辑变化；保存后才作用于项目。每条攻击路径独立计算，场景取最高风险。残余风险使用同一分级算法。
              </Paragraph>
            </Card>
            <Card title="算法扩展">
              <Paragraph>
                新增评分维度或算法需进行部署扩展。接口与数据结构见开发文档。
              </Paragraph>
              <Space wrap>
                <Button href="/api/v1/docs" target="_blank" rel="noreferrer">
                  接口文档
                </Button>
                <Text type="secondary">扩展指南见开发文档</Text>
              </Space>
            </Card>
          </Col>
        </Row>
      </Form>
    </>
  );
}
