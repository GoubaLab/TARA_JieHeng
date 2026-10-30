import React from "react";
import {
  Alert,
  Card,
  Form,
  Input,
  Select,
  Space,
  Table,
  Tabs,
  Typography,
} from "antd";
import { effectiveMethod, dimensionKeys } from "./method-config.js";
import {
  impactGuidance,
  projectImpactCriteria,
  potentialFactors,
  potentialResult,
  attackVectors,
} from "./tara-guidance.js";
const { Text, Paragraph } = Typography;
export function ImpactFields({ ws, form, name = "impact", disabled = false }) {
  const method = effectiveMethod(ws.method);
  return (
    <Space orientation="vertical" size="middle" style={{ width: "100%" }}>
      <Alert
        type="info"
        showIcon
        title="分别判断 S、F、O、P，并记录使用情景和证据"
        description="参考 ISO/SAE 21434:2021 §15.5 与附录 F。附录 F 是分级示例；项目采用的准则与标准参考分别显示。财务金额、权重及聚合公式由项目确定。"
      />
      {dimensionKeys.map((key, i) => {
        const reference = impactGuidance[key],
          custom = projectImpactCriteria(ws, key);
        return (
          <Card
            size="small"
            key={key}
            title={`${"SFOP"[i]} · ${method.impact_dimensions[key]}`}
          >
            <Form.Item
              name={[name, key]}
              label="影响等级"
              rules={[{ required: true, message: "请结合下方准则选择等级" }]}
            >
              <Select
                disabled={disabled}
                placeholder="选择等级并核对判断准则"
                options={method.impact_levels.map((l, value) => ({
                  value,
                  label: `${l.label}${method.impact_aggregation === "coefficient_bands" ? ` · ${method.impact_scale[value]}` : ""}`,
                  description:
                    custom?.rows[value]?.description || reference.rows[value],
                }))}
                optionRender={(o) => (
                  <div style={{ whiteSpace: "normal" }}>
                    <Text strong>{o.label}</Text>
                    <div>{o.data.description}</div>
                  </div>
                )}
              />
            </Form.Item>
            <Tabs
              size="small"
              items={[
                ...(custom
                  ? [
                      {
                        key: "project",
                        label: "项目判定准则",
                        children: (
                          <>
                            <Paragraph type="secondary">
                              {custom.source}
                            </Paragraph>
                            <Table
                              size="small"
                              pagination={false}
                              rowKey="level"
                              dataSource={custom.rows.map((r, level) => ({
                                ...r,
                                level,
                              }))}
                              columns={[
                                {
                                  title: "等级",
                                  dataIndex: "label",
                                  width: 90,
                                },
                                { title: "判定条件", dataIndex: "description" },
                              ]}
                            />
                          </>
                        ),
                      },
                    ]
                  : []),
                {
                  key: "standard",
                  label: "21434 分级参考",
                  children: (
                    <>
                      <Paragraph type="secondary">
                        ISO/SAE 21434:2021 · {reference.source}（释义）
                      </Paragraph>
                      <Table
                        size="small"
                        pagination={false}
                        rowKey="level"
                        dataSource={reference.rows.map(
                          (description, level) => ({
                            level,
                            label: method.impact_levels[level].label,
                            description,
                          }),
                        )}
                        columns={[
                          { title: "等级", dataIndex: "label", width: 90 },
                          { title: "判断条件", dataIndex: "description" },
                        ]}
                      />
                    </>
                  ),
                },
              ]}
            />
          </Card>
        );
      })}
      <Form.Item
        name={[name, "rationale"]}
        label="影响评级依据"
        rules={[
          {
            required: true,
            whitespace: true,
            message: "请说明四类影响的判断依据",
          },
        ]}
      >
        <Input.TextArea
          disabled={disabled}
          rows={4}
          maxLength={6000}
          placeholder="说明受影响对象、运行情景、不利后果，以及各维度为何符合所选等级；引用相应需求、分析或证据。"
        />
      </Form.Item>
    </Space>
  );
}
export function FeasibilityFields({ form, ws }) {
  const method = effectiveMethod(ws.method),
    chosen = Form.useWatch("method", form),
    potential = Form.useWatch("potential", form),
    vector = Form.useWatch("attack_vector", form);
  const result = potentialResult(potential, method),
    thresholds = method.attack_potential_thresholds || [14, 20, 25];
  return (
    <>
      <Alert
        type="info"
        showIcon
        className="section-card"
        title="每条攻击路径必须完成攻击可行性评级"
        description="ISO/SAE 21434:2021 §15.7。选择方法、填写条件和依据，系统计算等级；未完成的路径保持待评估，不能形成风险处置结论。"
      />
      <Form.Item
        name="method"
        label="评级方法"
        rules={[{ required: true, message: "请选择评级方法" }]}
      >
        <Select
          options={[
            {
              value: "attack_potential",
              label: "攻击潜力 · 五因子（附录 G.2）",
            },
            { value: "attack_vector", label: "主要攻击向量（附录 G.4）" },
          ]}
        />
      </Form.Item>
      {chosen === "attack_potential" && (
        <>
          <Paragraph>
            AP 下限：中 {thresholds[0]} / 低 {thresholds[1]} / 很低{" "}
            {thresholds[2]}。
            {JSON.stringify(thresholds) === JSON.stringify([14, 20, 25])
              ? "采用附录 G 表 G.7 示例分档。"
              : "采用当前项目分档；附录 G 表 G.7 示例为 14 / 20 / 25。"}
          </Paragraph>
          {potentialFactors.map((f) => (
            <Card
              key={f.key}
              size="small"
              className="section-card"
              title={f.label}
            >
              <Form.Item
                name={["potential", f.key]}
                label="判定档位"
                rules={[
                  { required: true, message: "请选择符合攻击条件的档位" },
                ]}
                extra={`${f.description} 参考附录 G 表 ${f.source}。`}
              >
                <Select
                  placeholder="按条件选择，分值自动带入"
                  options={f.options.map(([value, label]) => ({
                    value,
                    label: `${label} · ${value} 分`,
                  }))}
                  optionRender={(o) => (
                    <div style={{ whiteSpace: "normal" }}>{o.label}</div>
                  )}
                />
              </Form.Item>
            </Card>
          ))}
          <Alert
            className="section-card"
            type={result ? "success" : "warning"}
            title={
              result
                ? `AP = ${potentialFactors.map((f) => potential[f.key]).join(" + ")} = ${result.total}；攻击可行性：${method.feasibility_levels[result.level - 1].label}`
                : "待评估：请完成五项档位选择"
            }
          />
        </>
      )}
      {chosen === "attack_vector" && (
        <>
          <Form.Item
            name="attack_vector"
            label="主要攻击向量"
            rules={[{ required: true, message: "请选择主要攻击向量" }]}
          >
            <Select
              options={attackVectors}
              optionRender={(o) => (
                <div style={{ whiteSpace: "normal" }}>
                  <Text strong>{o.label}</Text>
                  <div>{o.data.description}</div>
                </div>
              )}
            />
          </Form.Item>
          <Paragraph type="secondary">
            附录 G 表 G.9
            的示例映射。请结合路径实际访问条件判断，不能仅根据通信协议名称推断。
          </Paragraph>
          {vector && (
            <Alert
              className="section-card"
              type="success"
              title={`攻击可行性：${method.feasibility_levels[attackVectors.find((v) => v.value === vector).level - 1].label}`}
            />
          )}
        </>
      )}
      <Form.Item
        name="rationale"
        label="可行性判定依据"
        rules={[
          {
            required: true,
            whitespace: true,
            message: "请记录攻击条件、所需资源和证据依据",
          },
        ]}
      >
        <Input.TextArea
          rows={4}
          maxLength={6000}
          placeholder="解释所选档位及攻击前提，列出资料、验证或分析依据。"
        />
      </Form.Item>
    </>
  );
}
