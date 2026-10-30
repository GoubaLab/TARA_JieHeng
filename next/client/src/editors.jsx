import AttackPathTemplatePicker from "./AttackPathTemplatePicker.jsx";
import { appendTemplatePaths } from "./attack-path-templates.js";
import { TaraFields } from "./TaraFields.jsx";
import React, { useState, useEffect, useRef } from "react";
import {
  Drawer,
  Modal,
  Form,
  Input,
  Select,
  Button,
  Space,
  Typography,
  Alert,
  Row,
  Col,
  InputNumber,
  Divider,
  Card,
  Table,
  Tag,
  Descriptions,
  Empty,
  Spin,
  Collapse,
  Checkbox,
} from "antd";
import {
  PlusOutlined,
  DeleteOutlined,
  SearchOutlined,
  DatabaseOutlined,
} from "@ant-design/icons";
import { useWorkbench, Risk } from "./Workbench.jsx";
import {
  request,
  navigate,
  properties,
  propOptions,
  gradeOptions,
  feasibilityOptions,
  treatments,
  impacts,
  feasibility,
} from "./api.js";
import {
  domains,
  dimensionKeys,
  dimensionCodes,
  effectiveMethod,
  coefficientMaximum,
  levelOptions,
} from "./method-config.js";
const { Text, Paragraph } = Typography;
const { TextArea } = Input;
const required = [{ required: true, whitespace: true, message: "请填写此项" }];
const chosen = [{ required: true, message: "请选择此项" }];
const copy = (v) => structuredClone(v);
const pick = (v, keys) =>
  Object.fromEntries(
    keys.filter((k) => v[k] !== undefined).map((k) => [k, v[k]]),
  );
const blankPath = () => ({
  id: "path_" + crypto.randomUUID(),
  name: "",
  steps: "",
  feasibility: null,
  rationale: "",
  reference: null,
});
const titles = {
  item: "编辑相关项定义",
  method: "配置评估方法",
  asset: "编辑资产",
  scenario: "编辑风险场景",
  decision: "记录处置决策",
  requirement: "编辑安全需求",
  evidence: "新增验证记录",
  baseline: "建立评审基线",
};
const commands = {
  item: "define-item",
  method: "adopt-method",
  asset: "record-asset",
  scenario: "record-scenario",
  decision: "decide-treatment",
  requirement: "specify-requirement",
  evidence: "record-evidence",
  baseline: "seal-baseline",
};
export function ProjectCreator({ open, onClose }) {
  const { reload, message } = useWorkbench();
  const [form] = Form.useForm();
  const creationAttempt = useRef(null);
  const [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  useEffect(() => {
    if (open) {
      form.resetFields();
      creationAttempt.current = null;
      setError("");
    }
  }, [open]);
  const create = async () => {
    try {
      const values = await form.validateFields();
      setBusy(true);
      setError("");
      const body = JSON.stringify(values);
      if (creationAttempt.current?.body !== body)
        creationAttempt.current = { body, key: crypto.randomUUID() };
      const ws = await request("/workspaces", {
        method: "POST",
        headers: { "Idempotency-Key": creationAttempt.current.key },
        body,
      });
      await reload();
      message.success("项目已创建，请填写相关项定义");
      onClose();
      navigate("definition", ws.id);
    } catch (e) {
      if (!e.errorFields) setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <Modal
      title="新建威胁建模项目"
      open={open}
      onCancel={() => !busy && onClose()}
      onOk={create}
      confirmLoading={busy}
      okText="创建项目"
      cancelText="取消"
      destroyOnHidden
    >
      <Paragraph type="secondary">
        从相关项定义开始，逐步建立资产、风险场景和验证记录。各工作区可自由切换。
      </Paragraph>
      {error && (
        <Alert type="error" title={error} showIcon className="section-card" />
      )}
      <Form form={form} layout="vertical" initialValues={{ description: "" }}>
        <Form.Item name="title" label="项目名称" rules={required}>
          <Input
            placeholder="例如：设备控制系统、云服务或车载系统"
            maxLength={200}
            autoFocus
          />
        </Form.Item>
        <Form.Item name="description" label="项目说明">
          <TextArea
            rows={3}
            placeholder="填写产品、系统范围和分析目的"
            maxLength={4000}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}
export function EntityEditor({ editor, onClose }) {
  const {
    ws,
    catalog,
    review,
    save,
    message,
    modal,
    setPicker,
    setRegulationPicker,
    setReference,
  } = useWorkbench();
  const [form] = Form.useForm();
  const [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const kind = editor?.kind;
  const configuredMethod = effectiveMethod(ws?.method);
  const gradeOptions = levelOptions(configuredMethod.impact_levels).map(
    (o, i) => ({
      ...o,
      label:
        configuredMethod.impact_aggregation === "coefficient_bands" &&
        configuredMethod.impact_scale
          ? `${configuredMethod.impact_scale[i]} · ${configuredMethod.impact_levels[i].label}`
          : o.label,
    }),
  );
  const damageIds = Form.useWatch("damage_ids", form);
  const feasibilityOptions = levelOptions(
    configuredMethod.feasibility_levels,
    1,
  );
  const initialValues = useRef("");
  const [pathOpen, setPathOpen] = useState([]);
  const [pathTemplatesOpen, setPathTemplatesOpen] = useState(false);
  const pathValues = Form.useWatch("paths", form);
  const pathProperty = Form.useWatch("property", form);
  const assetId = Form.useWatch("asset_id", form);
  const treatment = Form.useWatch("treatment", form);
  const selectedGoalIDs = Form.useWatch("goal_ids", form) || [];
  const scenarioId = Form.useWatch("scenario_id", form);
  const [labels, setLabels] = useState({});
  useEffect(() => {
    if (!editor || !ws) return;
    setError("");
    setLabels({});
    form.resetFields();
    let value = {};
    if (kind === "item")
      value = { ...copy(ws.item), domain: ws.item.domain || "automotive" };
    if (kind === "method") value = copy(ws.method);
    if (kind === "asset") {
      value = editor.id
        ? copy(ws.assets.find((v) => v.id === editor.id))
        : {
            name: editor.source?.title || "",
            description: editor.source?.description || "",
            properties: ["integrity", "availability"],
            node_ids: [],
            reference: editor.source?.key || null,
          };
      if (editor.source)
        setLabels({ [editor.source.key]: editor.source.title });
    }
    if (kind === "scenario")
      value = editor.id
        ? copy(ws.scenarios.find((v) => v.id === editor.id))
        : {
            title: "",
            asset_id: editor.assetId || ws.assets[0]?.id,
            property: (ws.assets.find(
              (a) => a.id === (editor.assetId || ws.assets[0]?.id),
            )?.properties || [])[0],
            damage: "",
            threat: "",
            impact: {
              safety: null,
              financial: null,
              operational: null,
              privacy: null,
              rationale: "",
            },
            paths: [],
            references: [],
          };
    if (kind === "decision")
      value = pick(
        ws.decisions.find((v) => v.scenario_id === editor.id) ||
          ws.treatment_drafts?.find((v) => v.scenario_id === editor.id) || {
            scenario_id: editor.id,
            treatment: "reduce",
            owner: ws.item.owner,
            rationale: "",
            goal: "",
            claim: "",
            residual_status: "target",
            residual_impact: null,
            residual_feasibility: null,
            residual_rationale: "",
          },
        [
          "scenario_id",
          "treatment",
          "rationale",
          "owner",
          "goal",
          "claim",
          "residual_impact",
          "residual_feasibility",
          "residual_rationale",
          "controls",
          "goal_ids",
          "cal",
          "acceptance",
          "acceptance_rationale",
          "customer_confirmation",
          "confirmed_at",
          "declarant",
          "residual_status",
        ],
      );
    if (kind === "requirement")
      value = editor.id
        ? copy(ws.requirements.find((v) => v.id === editor.id))
        : {
            scenario_id: editor.scenarioId || ws.scenarios[0]?.id,
            statement:
              editor.source?.statement || editor.source?.description || "",
            allocation: "",
            verification: "",
            standard_refs: [],
            reference: editor.source?.key || null,
          };
    if (kind === "evidence")
      value = {
        requirement_id: editor.id,
        result: "pending",
        reference: "",
        summary: "",
        reviewer: ws.item.owner,
      };
    if (kind === "baseline")
      value = {
        name: "评审基线 R" + ws.revision,
        reviewer: ws.item.owner,
        note: "",
      };
    if (kind === "requirement" && !value.scenario_ids?.length)
      value.scenario_ids = [value.scenario_id].filter(Boolean);
    form.setFieldsValue(value);
    initialValues.current = JSON.stringify(form.getFieldsValue(true));
    setPathOpen([]);
    setPathTemplatesOpen(false);
    let active = true;
    const refs = [
      ...new Set(
        [value.reference, ...(value.standard_refs || [])].filter(Boolean),
      ),
    ];
    Promise.allSettled(
      refs.map((key) => request("/references/" + encodeURIComponent(key))),
    ).then((results) => {
      if (active)
        setLabels((previous) => ({
          ...previous,
          ...Object.fromEntries(
            results
              .filter((r) => r.status === "fulfilled")
              .map((r) => [r.value.key, r.value.title]),
          ),
        }));
    });
    return () => {
      active = false;
    };
  }, [editor, ws?.id]);
  const close = () => {
    if (busy) return;
    if (JSON.stringify(form.getFieldsValue(true)) !== initialValues.current)
      modal.confirm({
        title: "放弃未保存的修改？",
        content: "已保存的项目记录不受影响。",
        okText: "放弃修改",
        cancelText: "继续编辑",
        onOk: onClose,
      });
    else onClose();
  };
  const submit = async () => {
    try {
      await form.validateFields();
      const values = form.getFieldsValue(true);
      setBusy(true);
      setError("");
      let payload = values;
      if (kind === "asset")
        payload = {
          ...pick(values, [
            "name",
            "description",
            "node_ids",
            "source_code",
            "asset_type",
            "group",
            "function_refs",
            "properties",
            "reference",
          ]),
          ...(editor.id ? { id: editor.id } : {}),
        };
      if (kind === "scenario") {
        payload = {
          ...pick(values, [
            "title",
            "asset_id",
            "property",
            "damage",
            "threat",
            "impact",
            "paths",
            "references",
            "source_code",
            "asset_ids",
            "damage_ids",
            "properties",
            "stride",
            "cause",
            "cal",
          ]),
          ...(editor.id ? { id: editor.id } : {}),
        };
        payload.impact = {
          safety: null,
          financial: null,
          operational: null,
          privacy: null,
          rationale: "",
          ...payload.impact,
        };
        for (const key of ["safety", "financial", "operational", "privacy"])
          payload.impact[key] ??= null;
        payload.paths = (payload.paths || []).map((p) => ({
          ...p,
          id: p.id || "path_" + crypto.randomUUID(),
          feasibility: p.feasibility ?? null,
          reference: p.reference ?? null,
          steps: p.steps || "",
          rationale: p.rationale || "",
        }));
        payload.references ||= [];
        if (payload.asset_ids)
          payload.asset_ids = [
            ...new Set([...payload.asset_ids, payload.asset_id]),
          ];
        if (payload.properties)
          payload.properties = [
            ...new Set([...payload.properties, payload.property]),
          ];
      }
      if (kind === "decision")
        payload = {
          ...pick(values, [
            "scenario_id",
            "treatment",
            "rationale",
            "owner",
            "goal",
            "claim",
            "residual_impact",
            "residual_feasibility",
            "residual_rationale",
            "controls",
            "goal_ids",
            "cal",
            "acceptance",
            "acceptance_rationale",
            "customer_confirmation",
            "confirmed_at",
            "declarant",
            "residual_status",
          ]),
          goal: values.goal || "",
          claim: values.claim || "",
          residual_impact: values.residual_impact ?? null,
          residual_feasibility: values.residual_feasibility ?? null,
        };
      if (kind === "requirement")
        payload = {
          ...pick(values, [
            "scenario_id",
            "statement",
            "allocation",
            "verification",
            "standard_refs",
            "source_code",
            "scenario_ids",
            "goal_ids",
            "cal",
            "customer_refs",
            "reference",
          ]),
          ...(editor.id ? { id: editor.id } : {}),
        };
      if (kind === "requirement") {
        payload.scenario_ids = [...new Set(payload.scenario_ids || [])];
        if (!payload.scenario_ids.includes(payload.scenario_id))
          payload.scenario_id = payload.scenario_ids[0];
      }
      await save(commands[kind], payload);
      message.success(
        kind === "baseline"
          ? "评审基线已建立"
          : kind === "evidence"
            ? "验证记录已追加"
            : "修改已保存",
      );
      onClose();
    } catch (e) {
      if (e.errorFields) {
        const paths = e.errorFields
          .filter((f) => f.name[0] === "paths")
          .map((f) => String(f.name[1]));
        if (paths.length)
          setPathOpen((current) => [...new Set([...current, ...paths])]);
        setError("请补全表单中标记的必填项后再保存。");
        setTimeout(
          () => form.scrollToField(e.errorFields[0].name, { focus: true }),
          0,
        );
      } else setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  const chooseReference = (field, category, multiple = false) =>
    setPicker({
      category,
      multiple,
      onSelect: (rows) => {
        setLabels((l) => ({
          ...l,
          ...Object.fromEntries(rows.map((r) => [r.key, r.title])),
        }));
        form.setFieldValue(
          field,
          multiple
            ? [
                ...new Set([
                  ...(form.getFieldValue(field) || []),
                  ...rows.map((r) => r.key),
                ]),
              ]
            : rows[0]?.key || null,
        );
      },
    });
  const referenceField = (field, label, category, multiple = false) => {
    if (!catalog?.categories.length && !form.getFieldValue(field)) return null;
    return (
      <Form.Item label={label}>
        <Space orientation="vertical" style={{ width: "100%" }}>
          <Form.Item name={field} noStyle>
            <Select
              mode={multiple ? "multiple" : undefined}
              open={false}
              allowClear
              placeholder={
                catalog?.categories.length
                  ? "从参考资料中选择"
                  : "未配置可选参考资料"
              }
              options={Object.entries(labels).map(([value, label]) => ({
                value,
                label,
              }))}
              labelRender={(p) => (
                <span>{labels[p.value] || "已关联参考条目"}</span>
              )}
            />
          </Form.Item>
          {catalog?.categories.length > 0 && (
            <Button
              size="small"
              icon={<DatabaseOutlined />}
              onClick={() => chooseReference(field, category, multiple)}
            >
              浏览并选择
            </Button>
          )}
        </Space>
      </Form.Item>
    );
  };
  const selectedScenario = ws?.scenarios.find(
    (s) => s.id === (kind === "decision" ? editor?.id : scenarioId),
  );
  const selectedEvaluation = review?.evaluations.find(
    (e) => e.scenario_id === editor?.id,
  );
  const requirement = ws?.requirements.find((r) => r.id === editor?.id);
  return (
    <Drawer
      title={titles[kind] || "编辑工作产品"}
      open={!!editor}
      onClose={close}
      size={kind === "scenario" || kind === "method" ? 900 : 760}
      destroyOnHidden
      maskClosable={!busy}
      footer={
        <div className="editor-footer">
          <Text type="secondary">
            {kind === "evidence"
              ? "验证记录会追加保留，不覆盖历史结果。"
              : "保存后更新项目修订并自动检查待办。"}
          </Text>
          <Space>
            <Button onClick={close} disabled={busy}>
              取消
            </Button>
            <Button type="primary" onClick={submit} loading={busy}>
              {kind === "method"
                ? "确认并应用方法"
                : kind === "baseline"
                  ? "建立基线"
                  : "保存"}
            </Button>
          </Space>
        </div>
      }
    >
      {editor && ws && (
        <>
          <Alert
            type="error"
            showIcon
            title={error}
            style={{ display: error ? "block" : "none", marginBottom: 16 }}
          />
          <Form form={form} layout="vertical" requiredMark>
            <TaraFields kind={kind} ws={ws} />
            {kind === "item" && (
              <>
                <Form.Item
                  name="domain"
                  label="应用领域"
                  extra="领域用于区分分析上下文；不会自动代选法规或改变评分方法。"
                >
                  <Select
                    options={Object.entries(domains).map(([value, label]) => ({
                      value,
                      label,
                    }))}
                  />
                </Form.Item>
                <Form.Item
                  name="purpose"
                  label="功能与预期用途"
                  rules={required}
                >
                  <TextArea
                    rows={3}
                    placeholder="系统提供哪些功能、适用哪些用户和业务场景"
                    maxLength={10000}
                  />
                </Form.Item>
                <Form.Item
                  name="boundary"
                  label="系统边界与外部接口"
                  rules={required}
                >
                  <TextArea
                    rows={3}
                    placeholder="填写分析范围、外部交互及排除项"
                    maxLength={10000}
                  />
                </Form.Item>
                <Form.Item name="environment" label="运行环境" rules={required}>
                  <TextArea rows={3} maxLength={10000} />
                </Form.Item>
                <Form.Item
                  name="assumptions"
                  label="假设与依赖"
                  rules={required}
                >
                  <TextArea rows={3} maxLength={10000} />
                </Form.Item>
                <Row gutter={16}>
                  <Col span={12}>
                    <Form.Item name="owner" label="项目负责人" rules={required}>
                      <Input maxLength={200} />
                    </Form.Item>
                  </Col>
                  <Col span={12}>
                    <Form.Item
                      name="lifecycle"
                      label="生命周期阶段"
                      rules={chosen}
                    >
                      <Select
                        options={[
                          "概念阶段",
                          "产品开发阶段",
                          "生产阶段",
                          "运行与维护阶段",
                          "退役阶段",
                        ].map((value) => ({ value, label: value }))}
                      />
                    </Form.Item>
                  </Col>
                </Row>
              </>
            )}
            {kind === "method" && (
              <>
                <Alert
                  type="info"
                  showIcon
                  title="项目评估方法"
                  description="使用前请确认组织适用的评估方法。方法变更后，相关处置决策需要复核。"
                  className="section-card"
                />
                <Form.Item name="name" label="方法名称" rules={required}>
                  <Input maxLength={200} />
                </Form.Item>
                <Form.Item
                  name="rationale"
                  label="方法说明与适用依据"
                  rules={required}
                >
                  <TextArea rows={4} maxLength={6000} />
                </Form.Item>
                <div className="form-section-title">风险矩阵</div>
                <Paragraph type="secondary">
                  风险等级为 1–5；影响或攻击可行性增加时，风险等级不能降低。
                </Paragraph>
                <div className="matrix-editor">
                  <div>影响 / 可行性</div>
                  {feasibility.map((v) => (
                    <div key={v}>{v}</div>
                  ))}
                  {impacts.map((v, i) => (
                    <React.Fragment key={v}>
                      <div>{v}</div>
                      {feasibility.map((f, j) => (
                        <Form.Item
                          key={j}
                          name={["matrix", i, j]}
                          rules={[
                            { required: true, message: "填写等级" },
                            {
                              type: "integer",
                              min: 1,
                              max: 5,
                              message: "整数 1–5",
                            },
                          ]}
                        >
                          <InputNumber
                            aria-label={`${v}影响、${f}可行性的风险等级`}
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
              </>
            )}
            {kind === "asset" && (
              <>
                {ws.assets.find((a) => a.id === editor.id)
                  ?.architecture_source && (
                  <Alert
                    className="section-card"
                    type={
                      ws.assets.find((a) => a.id === editor.id)
                        .architecture_source.status === "missing"
                        ? "warning"
                        : "info"
                    }
                    showIcon
                    title={
                      ws.assets.find((a) => a.id === editor.id)
                        .architecture_source.status === "missing"
                        ? "架构来源已移除，资产和已有风险分析已保留"
                        : "此资产由系统架构自动识别"
                    }
                    description="名称、说明和安全属性可直接修改；后续保存架构会保留这些手工修改。"
                  />
                )}
                <Form.Item name="name" label="资产名称" rules={required}>
                  <Input maxLength={200} placeholder="例如：高压控制指令" />
                </Form.Item>
                <Form.Item name="description" label="资产说明">
                  <TextArea rows={3} maxLength={6000} />
                </Form.Item>
                <Form.Item
                  name="properties"
                  label="需要保护的安全属性"
                  rules={[
                    {
                      required: true,
                      type: "array",
                      min: 1,
                      message: "至少选择一项安全属性",
                    },
                  ]}
                >
                  <Checkbox.Group options={propOptions} />
                </Form.Item>
                <Form.Item
                  name="node_ids"
                  label="关联实体"
                  extra="将资产关联到所属实体。软件和数据存储在实体内部维护。"
                >
                  <Select
                    mode="multiple"
                    placeholder="选择实体"
                    options={ws.architecture.nodes
                      .filter((n) => n.kind !== "boundary")
                      .map((n) => ({
                        value: n.id,
                        label:
                          n.kind === "store"
                            ? n.name + "（待归属实体）"
                            : n.name,
                        disabled:
                          n.kind === "store" &&
                          ws.architecture.view_mode !== "dfd",
                      }))}
                  />
                </Form.Item>
                {!ws.architecture.nodes.length && (
                  <Alert
                    type="info"
                    title="目前没有实体。可以先保存资产，再在系统架构中补充实体。"
                    className="section-card"
                  />
                )}
                {referenceField("reference", "参考组件知识", "systems")}
              </>
            )}
            {kind === "decision" && (
              <>
                <Alert
                  className="section-card"
                  type={selectedEvaluation?.decision_stale ? "warning" : "info"}
                  showIcon
                  title={selectedScenario?.title}
                  description={
                    selectedEvaluation?.decision_stale
                      ? "输入发生变化，原决策已失效。请重新核对处置与残余风险。"
                      : "记录处置策略、负责人及残余风险的独立评估。"
                  }
                />
                <Form.Item name="scenario_id" hidden>
                  <Input />
                </Form.Item>
                <Form.Item name="treatment" label="处置策略" rules={chosen}>
                  <Select
                    options={Object.entries(treatments).map(
                      ([value, label]) => ({ value, label }),
                    )}
                  />
                </Form.Item>
                <Form.Item name="rationale" label="处置依据" rules={required}>
                  <TextArea rows={3} maxLength={8000} />
                </Form.Item>
                <Form.Item name="owner" label="处置负责人" rules={required}>
                  <Input maxLength={200} />
                </Form.Item>
                {treatment === "reduce" ? (
                  <Form.Item
                    name="goal"
                    label="网络安全目标"
                    rules={selectedGoalIDs.length ? [] : required}
                    extra={
                      selectedGoalIDs.length
                        ? "已关联上方安全目标，保存时自动带入目标内容，无需重复填写。"
                        : "可先记录目标说明，再在安全目标页建立独立工作产品。"
                    }
                  >
                    <TextArea rows={3} maxLength={8000} />
                  </Form.Item>
                ) : (
                  <Form.Item name="claim" label="网络安全声明" rules={required}>
                    <TextArea
                      rows={3}
                      maxLength={8000}
                      placeholder="说明选择该策略的条件、责任和限制"
                    />
                  </Form.Item>
                )}
                <Divider>残余风险评估</Divider>
                <Paragraph type="secondary">
                  根据措施实施后的情况重新评估。未确认时留空，不沿用初始风险分值。
                </Paragraph>
                <Row gutter={16}>
                  <Col span={12}>
                    <Form.Item
                      name="residual_impact"
                      label={
                        configuredMethod.impact_aggregation ===
                        "coefficient_sum"
                          ? "残余 SFOP 总分"
                          : "残余影响"
                      }
                      extra={
                        configuredMethod.impact_aggregation ===
                        "coefficient_sum"
                          ? "按同一组系数独立评估后的总分，可保留小数。"
                          : undefined
                      }
                    >
                      {configuredMethod.impact_aggregation ===
                      "coefficient_sum" ? (
                        <InputNumber
                          min={0}
                          max={coefficientMaximum(configuredMethod)}
                          step={0.1}
                          style={{ width: "100%" }}
                          placeholder="待评估"
                        />
                      ) : (
                        <Select
                          options={gradeOptions}
                          allowClear
                          placeholder="待评估"
                        />
                      )}
                    </Form.Item>
                  </Col>
                  <Col span={12}>
                    <Form.Item
                      name="residual_feasibility"
                      label="残余攻击可行性"
                    >
                      <Select
                        options={feasibilityOptions}
                        allowClear
                        placeholder="待评估"
                      />
                    </Form.Item>
                  </Col>
                </Row>
                <Form.Item name="residual_rationale" label="残余风险判定依据">
                  <TextArea rows={3} maxLength={8000} />
                </Form.Item>
              </>
            )}
            {kind === "requirement" && (
              <>
                <Form.Item name="scenario_id" hidden>
                  <Select
                    showSearch
                    optionFilterProp="label"
                    options={ws.scenarios.map((s) => ({
                      value: s.id,
                      label: s.title,
                    }))}
                  />
                </Form.Item>
                <Form.Item
                  name="statement"
                  label="网络安全需求"
                  rules={required}
                >
                  <TextArea
                    rows={4}
                    maxLength={10000}
                    placeholder="使用可实现、可验证的表述"
                  />
                </Form.Item>
                <Form.Item name="allocation" label="分配对象" rules={required}>
                  <Input
                    maxLength={500}
                    placeholder="负责实现需求的实体、软件或团队"
                  />
                </Form.Item>
                <Form.Item
                  name="verification"
                  label="验证方法与通过准则"
                  rules={required}
                >
                  <TextArea
                    rows={4}
                    maxLength={8000}
                    placeholder="说明如何测试、检查哪些结果以及什么情况下通过"
                  />
                </Form.Item>
                {referenceField("reference", "参考控制措施", "controls")}
                <Form.Item label="引用法规条款">
                  <Form.Item name="standard_refs" noStyle>
                    <Select
                      mode="multiple"
                      open={false}
                      allowClear
                      placeholder="勾选正式版本的条款"
                      options={Object.entries(labels).map(([value, label]) => ({
                        value,
                        label,
                      }))}
                      style={{ width: "100%" }}
                    />
                  </Form.Item>
                  <Button
                    style={{ marginTop: 10 }}
                    icon={<DatabaseOutlined />}
                    onClick={() =>
                      setRegulationPicker({
                        initialRefs: form.getFieldValue("standard_refs") || [],
                        onSelect: (keys, records) => {
                          form.setFieldValue("standard_refs", keys);
                          setLabels((old) => ({
                            ...old,
                            ...Object.fromEntries(
                              Object.values(records).map((row) => [
                                row.key,
                                `${row.document_title || row.document_id} · ${row.edition} · ${row.number}`,
                              ]),
                            ),
                          }));
                        },
                      })
                    }
                  >
                    浏览法规并勾选
                  </Button>
                </Form.Item>
                <Text type="secondary">
                  引用记录包含文档版本与条号；失效引用需要核对或替换。
                </Text>
              </>
            )}
            {kind === "evidence" && (
              <>
                <Alert
                  showIcon
                  type="info"
                  title="验证对象"
                  description={requirement?.statement}
                  className="section-card"
                />
                <Descriptions
                  size="small"
                  column={1}
                  items={[
                    {
                      key: "a",
                      label: "分配对象",
                      children: requirement?.allocation,
                    },
                    {
                      key: "v",
                      label: "验证准则",
                      children: requirement?.verification,
                    },
                  ]}
                  className="section-card"
                />
                <Form.Item name="requirement_id" hidden>
                  <Input />
                </Form.Item>
                <Form.Item name="result" label="验证结果" rules={chosen}>
                  <Select
                    options={[
                      { value: "pending", label: "待验证" },
                      { value: "pass", label: "通过" },
                      { value: "fail", label: "未通过" },
                    ]}
                  />
                </Form.Item>
                <Form.Item
                  name="reference"
                  label="证据引用"
                  rules={required}
                  extra="填写报告编号、受控文件位置或测试记录编号。"
                >
                  <Input maxLength={2000} />
                </Form.Item>
                <Form.Item
                  name="summary"
                  label="验证过程与结论"
                  rules={required}
                >
                  <TextArea rows={4} maxLength={10000} />
                </Form.Item>
                <Form.Item name="reviewer" label="验证人员" rules={required}>
                  <Input maxLength={200} />
                </Form.Item>
              </>
            )}
            {kind === "baseline" && (
              <>
                <Alert
                  type="success"
                  showIcon
                  title="当前完整性检查通过"
                  description={`将保留修订 ${ws.revision} 的完整工作产品快照，后续修改不会覆盖此基线。`}
                  className="section-card"
                />
                <Form.Item name="name" label="基线名称" rules={required}>
                  <Input maxLength={200} />
                </Form.Item>
                <Form.Item name="reviewer" label="评审人" rules={required}>
                  <Input maxLength={200} />
                </Form.Item>
                <Form.Item name="note" label="评审说明" rules={required}>
                  <TextArea rows={4} maxLength={6000} />
                </Form.Item>
              </>
            )}
          </Form>
        </>
      )}
    </Drawer>
  );
}
export function ReferencePicker({ config, onClose }) {
  const { message } = useWorkbench();
  const [items, setItems] = useState([]),
    [total, setTotal] = useState(0),
    [page, setPage] = useState(1),
    [q, setQ] = useState(""),
    [search, setSearch] = useState(""),
    [selected, setSelected] = useState([]),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  useEffect(() => {
    if (config) {
      setPage(1);
      setQ("");
      setSearch("");
      setSelected([]);
    }
  }, [config]);
  useEffect(() => {
    if (!config) return;
    let active = true;
    setBusy(true);
    setError("");
    request(
      `/catalog/${config.category}/entries?q=${encodeURIComponent(search)}&offset=${(page - 1) * 10}&limit=10`,
    )
      .then((v) => {
        if (active) {
          setItems(v.items);
          setTotal(v.total);
        }
      })
      .catch((e) => {
        if (active) setError(e.message);
      })
      .finally(() => active && setBusy(false));
    return () => {
      active = false;
    };
  }, [config, search, page]);
  return (
    <Modal
      title="选择参考条目"
      open={!!config}
      width={960}
      okText={`选用${selected.length ? " " + selected.length + " 项" : ""}`}
      cancelText="取消"
      okButtonProps={{ disabled: !selected.length }}
      onCancel={onClose}
      onOk={() => {
        config.onSelect(selected);
        onClose();
      }}
    >
      <div className="table-toolbar">
        <Text type="secondary">选择后可继续按项目实际编辑。</Text>
        <Input.Search
          placeholder="搜索名称或内容"
          aria-label="搜索参考条目"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          onSearch={(v) => {
            setSearch(v);
            setPage(1);
          }}
          style={{ width: 340 }}
        />
      </div>
      {error && <Alert type="error" title={error} showIcon />}
      <Table
        loading={busy}
        rowKey="key"
        dataSource={items}
        size="small"
        rowSelection={{
          type: config?.multiple ? "checkbox" : "radio",
          selectedRowKeys: selected.map((r) => r.key),
          preserveSelectedRowKeys: true,
          onChange: (_, rows) => setSelected(rows.filter(Boolean)),
        }}
        pagination={{
          current: page,
          pageSize: 10,
          total,
          showSizeChanger: false,
          onChange: setPage,
          showTotal: (v) => `共 ${v} 条`,
        }}
        columns={[
          { title: "名称", dataIndex: "title", width: "34%" },
          {
            title: "内容摘要",
            dataIndex: "description",
            render: (v) => (
              <Paragraph ellipsis={{ rows: 3 }} style={{ margin: 0 }}>
                {v || "—"}
              </Paragraph>
            ),
          },
        ]}
        scroll={{ y: 420 }}
      />
    </Modal>
  );
}
export function ReferenceDetail({ referenceKey, onClose }) {
  const [entry, setEntry] = useState(null),
    [error, setError] = useState("");
  useEffect(() => {
    if (!referenceKey) return;
    let active = true;
    setEntry(null);
    setError("");
    request("/references/" + encodeURIComponent(referenceKey))
      .then((v) => active && setEntry(v))
      .catch((e) => active && setError(e.message));
    return () => {
      active = false;
    };
  }, [referenceKey]);
  return (
    <Drawer
      title="参考条目详情"
      open={!!referenceKey}
      onClose={onClose}
      size={760}
    >
      {error ? (
        <Alert type="error" title={error} />
      ) : entry ? (
        <>
          <Typography.Title level={4}>{entry.title}</Typography.Title>
          {entry.withdrawn && (
            <Alert
              type="warning"
              showIcon
              title="引用不可用"
              description="来源内容不可用，无法用于新增记录。请核对现有引用或选择其他来源。"
              className="section-card"
            />
          )}
          <Tag>{entry.key}</Tag>
          <Paragraph className="pre-wrap" style={{ marginTop: 16 }}>
            {entry.description}
          </Paragraph>
          <Descriptions
            className="reference-facts"
            bordered
            column={1}
            size="small"
            items={Object.entries(entry.facts || {}).map(([label, value]) => ({
              key: label,
              label,
              children:
                typeof value === "object" ? (
                  <pre>{JSON.stringify(value, null, 2)}</pre>
                ) : (
                  String(value ?? "—")
                ),
            }))}
          />
          <Divider />
          <Text type="secondary">
            {entry.withdrawn
              ? "该引用仅保留来源标识。"
              : "项目编辑不会修改此条目。"}
          </Text>
        </>
      ) : (
        <Spin />
      )}
    </Drawer>
  );
}
import { AutoComplete } from "antd";
import {
  media,
  ranges,
  technologies,
  purposes,
  securityProperties,
  blankCommunication,
  communicationForDisplay,
  communicationForSave,
} from "./communication.js";

export function FlowEditor({ config, onClose }) {
  const [form] = Form.useForm();
  const [error, setError] = useState("");
  const medium =
    Form.useWatch(["communication", "medium"], form) || "unspecified";
  const technology = Form.useWatch(["communication", "technology"], form) || "";
  const legacy = config ? communicationForDisplay(config.value) : null;
  useEffect(() => {
    if (config) {
      form.resetFields();
      form.setFieldsValue({
        ...pick(config.value, ["name", "protocol", "description"]),
        channel_kind: config.value.channel_kind || "data",
        emphasis: config.value.emphasis || "normal",
        show_label: config.value.show_label !== false,
        communication: {
          ...blankCommunication(),
          ...config.value.communication,
        },
      });
      setError("");
    }
  }, [config]);
  const confirm = async () => {
    try {
      const values = await form.validateFields();
      const communication = communicationForSave(
        values.communication,
        config.value.communication,
      );
      if (communication === undefined) delete values.communication;
      else values.communication = communication;
      for (const [key, fallback] of [
        ["channel_kind", "data"],
        ["emphasis", "normal"],
        ["show_label", true],
      ]) {
        if (values[key] === fallback && !Object.hasOwn(config.value, key))
          delete values[key];
      }
      config.done(values);
      onClose();
    } catch (e) {
      if (!e.errorFields) setError(e.message);
    }
  };
  return (
    <Modal
      title="数据流属性"
      open={!!config}
      onCancel={onClose}
      onOk={confirm}
      okText="应用到画布"
      cancelText="取消"
      width={680}
    >
      {error && <Alert type="error" title={error} className="section-card" />}
      <Form form={form} layout="vertical">
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="channel_kind" label="连接类别">
              <Select
                options={[
                  { value: "data", label: "数据流" },
                  { value: "energy", label: "电能流（不生成数据资产）" },
                ]}
              />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="emphasis" label="图中标识">
              <Select
                options={[
                  { value: "normal", label: "一般连接" },
                  { value: "critical", label: "安全关键流（红色）" },
                  { value: "internal", label: "内部数据流（灰色）" },
                ]}
              />
            </Form.Item>
          </Col>
        </Row>
        <Form.Item name="show_label" valuePropName="checked">
          <Checkbox>在画布显示连接名称</Checkbox>
        </Form.Item>
        <Form.Item name="name" label="数据流名称" rules={required}>
          <Input maxLength={200} />
        </Form.Item>
        {legacy?.inferred && (
          <Alert
            type="info"
            showIcon
            className="section-card"
            title={`原协议识别为${media[legacy.medium]} · ${legacy.technology}`}
            description="当前仅用于画布展示。可采用此分类，也可按实际连接填写；上层协议原文保持可编辑。"
            action={
              <Button
                size="small"
                onClick={() =>
                  form.setFieldsValue({
                    communication: {
                      medium: legacy.medium,
                      technology: legacy.technology,
                    },
                  })
                }
              >
                采用分类
              </Button>
            }
          />
        )}
        <Row gutter={16}>
          <Col span={10}>
            <Form.Item name={["communication", "medium"]} label="传输方式">
              <Select
                options={Object.entries(media).map(([value, label]) => ({
                  value,
                  label,
                }))}
              />
            </Form.Item>
          </Col>
          <Col span={14}>
            <Form.Item
              name={["communication", "technology"]}
              label="通信技术"
              extra={
                technology.toLowerCase() === "doip"
                  ? "DoIP 可运行在 Ethernet 上；可在此补充组合方式。"
                  : "选择常用技术，也可直接输入自定义名称。"
              }
            >
              <AutoComplete
                options={(technologies[medium] || []).map((value) => ({
                  value,
                }))}
                filterOption={(input, option) =>
                  option.value.toLowerCase().includes(input.toLowerCase())
                }
                placeholder={
                  medium === "wireless"
                    ? "例如：Wi-Fi、NFC、Bluetooth/BLE"
                    : medium === "wired"
                      ? "例如：Ethernet、CAN FD、UART"
                      : "选择传输方式后可用预置技术"
                }
              >
                <Input maxLength={200} />
              </AutoComplete>
            </Form.Item>
          </Col>
        </Row>
        <Row gutter={16}>
          <Col span={10}>
            <Form.Item
              name={["communication", "range"]}
              label="覆盖范围"
              extra="辅助描述，可按部署选择。Wi-Fi 与蓝牙通常属于局域无线。"
            >
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
              <AutoComplete
                options={purposes.map((value) => ({ value }))}
                placeholder="例如：请求、遥测；也可自定义"
              >
                <Input maxLength={4000} />
              </AutoComplete>
            </Form.Item>
          </Col>
        </Row>
        <Form.Item
          name="protocol"
          label="上层协议 / 通道"
          extra="与通信技术分开记录；例如 Ethernet 上的 HTTPS、CAN 上的 UDS。"
        >
          <Input
            maxLength={200}
            placeholder="例如：HTTPS、MQTT、UDS、自定义诊断通道"
          />
        </Form.Item>
        <Form.Item
          name={["communication", "security_properties"]}
          label="关注的安全属性"
          extra="按分析需要勾选，不会根据技术名称自动判断其安全能力。"
        >
          <Checkbox.Group
            options={Object.entries(securityProperties).map(
              ([value, label]) => ({ value, label }),
            )}
          />
        </Form.Item>
        <Form.Item name="description" label="传输内容与约束">
          <TextArea rows={3} maxLength={4000} />
        </Form.Item>
      </Form>
      <Text type="secondary">应用后请在系统架构中保存。</Text>
    </Modal>
  );
}
