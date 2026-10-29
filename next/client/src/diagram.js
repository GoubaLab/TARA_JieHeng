import { numberedGraph } from "./numbering.js";
import {
  routeConnections,
  attachmentAt,
  editableSegments,
  snapRoutePoint,
  flowWaypoint,
  flowSide,
} from "./connection-routing.js";
import { labelOffsetForDrag } from "./flow-label.js";
import {
  entityTypes,
  isEntity,
  isDiagramEndpoint,
  entityType,
  entityKindLabel,
  normalizedArchitecture,
  assignLegacyStore,
  nodeMarkup,
  labelMarkup,
  staticDiagramSvg,
  connectionHandles,
} from "./architecture-view.js";
import {
  osFamilies,
  blankOperatingSystem,
  operatingSystemForSave,
  communicationDetails,
  flowDashArray,
  flowColor,
} from "./communication.js";
/* Independent SVG editor. Reuses only the locally authored pure geometry planner. */
const clone = (v) => structuredClone(v);
const kinds = {
  entity: "控制实体（ECU）",
  device: "设备",
  application: "应用",
  cloud: "云服务",
  actor: "参与者",
  external: "外部实体",
  service: "服务实体",
  boundary: "信任边界",
  process_step: "处理过程",
  data_store: "数据存储",
};
const sides = ["top", "right", "bottom", "left"];
export class Diagram {
  constructor(root, graph, options) {
    this.root = root;
    this.options = options;
    this.esc = options.escape;
    this.graph = clone(graph);
    this.saved = JSON.stringify(graph);
    this.draftKey = options.draftKey;
    let recovered = false;
    try {
      const draft = JSON.parse(sessionStorage.getItem(this.draftKey));
      if (draft && draft.base === this.saved) {
        this.graph = draft.graph;
        recovered = true;
      }
    } catch {}
    this.undo = [];
    this.redo = [];
    this.selected = null;
    this.view = { x: -40, y: -40, w: 1120, h: 660 };
    this.routes = [];
    this.alive = true;
    this.flowMotion = !window.matchMedia?.("(prefers-reduced-motion: reduce)")
      .matches;
    try {
      const preference = sessionStorage.getItem(this.draftKey + ".flow-motion");
      if (preference !== null) this.flowMotion = preference === "on";
    } catch {}
    root.innerHTML = `<div class="toolbar" role="toolbar" aria-label="架构绘图工具"><button class="btn small" data-dfd="view-mode" title="实体建模或展开内部 DFD">${graph.view_mode === "dfd" ? "展开 DFD" : "实体视图"}</button><select name="new-entity-type" aria-label="新增实体类型" style="width:auto;min-width:100px">${Object.entries(
      {
        ...entityTypes,
        ...(graph.view_mode === "dfd"
          ? { process_step: "处理过程", data_store: "数据存储" }
          : {}),
      },
    )
      .map(([key, label]) => `<option value="${key}">${label}</option>`)
      .join(
        "",
      )}</select><button class="btn small" data-dfd="add-selected-entity">＋ 添加节点</button><button class="btn small" data-dfd="add-boundary">＋ 信任边界</button><span class="separator"></span><button class="btn small" data-dfd="undo" title="撤销">↶</button><button class="btn small" data-dfd="redo" title="重做">↷</button><button class="btn small" data-dfd="fit">适合画布</button><button class="btn small" data-dfd="lanes" aria-pressed="${graph.equal_spacing}">通道间距</button><button class="btn small" data-dfd="motion" aria-pressed="true" title="显示数据流方向动画">流动效果</button><button class="btn small" data-dfd="numbers" aria-pressed="${graph.show_numbers !== false}">显示编号</button><button class="btn small" data-dfd="focus">专注画布</button><button class="btn small" data-dfd="export-static">导出静态图</button><span class="spacer"></span><button class="btn primary small" data-dfd="save" title="保存架构并自动同步资产清单">保存架构</button></div><details style="padding:8px 12px;border-bottom:1px solid #e5e9f0;color:#687281;font-size:12px"><summary style="cursor:pointer">图形与通信图例</summary><p style="margin:8px 0 0">白色卡片：设备；灰色卡片：外部实体；浅蓝色：处理过程或软件；浅绿色：数据存储；蓝色虚线框：信任边界。</p><p style="margin:8px 0 0">实线：有线或未分类；虚线：无线；点线：逻辑连接。协议与无线图标显示在线路标签中。红色表示强调的跨界流，浅灰色表示内部流，棕色表示电能流。箭头表示流向，跨线桥表示交叉但不相连。选中连线后，拖动两端圆点换边、拖动方点调整线路。</p></details><div class="canvas-layout"><div class="canvas-stage"><svg role="group" aria-label="系统数据流图" tabindex="0"></svg></div><aside class="canvas-inspector"></aside></div><div class="canvas-hint"><span class="hint-text">选中连线后拖动圆点换边、方点调整线路；拖动标签改位置。空白处拖动平移，Ctrl + 滚轮缩放。</span><span class="save-state"></span></div>`;
    this.svg = root.querySelector("svg");
    this.inspector = root.querySelector(".canvas-inspector");
    this.hint = root.querySelector(".hint-text");
    root.addEventListener("click", (this.click = (e) => this.onClick(e)));
    this.svg.addEventListener(
      "pointerdown",
      (this.down = (e) => this.pointerDown(e)),
    );
    window.addEventListener(
      "pointermove",
      (this.move = (e) => this.pointerMove(e)),
    );
    window.addEventListener("pointerup", (this.up = (e) => this.pointerUp(e)));
    window.addEventListener(
      "pointercancel",
      (this.cancel = (e) => this.cancelInteraction(e)),
    );
    window.addEventListener(
      "blur",
      (this.blur = () => this.cancelInteraction()),
    );
    this.svg.addEventListener(
      "wheel",
      (this.wheel = (e) => {
        if (!e.ctrlKey && !e.metaKey) return;
        e.preventDefault();
        const p = this.point(e),
          f = e.deltaY > 0 ? 1.12 : 0.89;
        this.view = {
          x: p.x - (p.x - this.view.x) * f,
          y: p.y - (p.y - this.view.y) * f,
          w: Math.max(280, Math.min(4000, this.view.w * f)),
          h: Math.max(165, Math.min(2400, this.view.h * f)),
        };
        this.render();
      }),
      { passive: false },
    );
    this.svg.addEventListener(
      "keydown",
      (this.key = (e) => {
        if (e.key === "Escape") {
          e.preventDefault();
          this.cancelInteraction();
          return;
        }
        if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "z") {
          e.preventDefault();
          this.cancelInteraction();
          this.history(e.shiftKey ? "redo" : "undo");
        }
      }),
    );
    this.layout();
    if (this.graph.nodes.length) this.fit();
    this.render();
    this.inspect();
    if (recovered) this.message("已恢复此工作区尚未保存的画布草稿。");
  }
  get dirty() {
    return JSON.stringify(this.graph) !== this.saved;
  }
  destroy() {
    if (!this.alive) return;
    this.alive = false;
    this.cancelInteraction();
    this.root.removeEventListener("click", this.click);
    this.svg.removeEventListener("pointerdown", this.down);
    this.svg.removeEventListener("wheel", this.wheel);
    this.svg.removeEventListener("keydown", this.key);
    window.removeEventListener("pointermove", this.move);
    window.removeEventListener("pointerup", this.up);
    window.removeEventListener("pointercancel", this.cancel);
    window.removeEventListener("blur", this.blur);
  }
  cancelInteraction(event) {
    if (event && this.pointerId !== event.pointerId) return;
    const drag = this.drag;
    this.drag = null;
    const pointerId = this.pointerId;
    this.pointerId = null;
    if (pointerId != null && this.svg.hasPointerCapture(pointerId))
      this.svg.releasePointerCapture(pointerId);
    if (
      drag?.type === "move" ||
      drag?.type === "label" ||
      drag?.type === "endpoint" ||
      drag?.type === "route"
    )
      this.graph = clone(drag.before);
    if (drag?.type === "pan") this.view = clone(drag.view);
    if (drag && ["endpoint", "route", "move"].includes(drag.type))
      this.layout();
    this.selected = null;
    if (this.alive) {
      this.render();
      this.inspect();
    }
  }
  point(e) {
    const m = this.svg.getScreenCTM();
    if (!m) return { x: 0, y: 0 };
    return new DOMPoint(e.clientX, e.clientY).matrixTransform(m.inverse());
  }
  message(text, error = false) {
    this.hint.textContent = text;
    this.root.querySelector(".canvas-hint").classList.toggle("error", error);
  }
  layout() {
    const result = routeConnections(this.graph);
    if (result.conflicts.length) {
      this.message(
        "连接空间不足，请调整实体间距或连接边。布局未更改。",
        true,
      );
      return false;
    }
    this.routes = result.routes;
    return true;
  }
  commit(before, reroute = true) {
    if (JSON.stringify(before) === JSON.stringify(this.graph)) return;
    if (reroute && !this.layout()) {
      this.graph = before;
      this.layout();
      this.render();
      this.inspect();
      return;
    }
    this.undo.push(before);
    if (this.undo.length > 60) this.undo.shift();
    this.redo = [];
    this.draft();
    this.render();
    this.inspect();
  }
  draft() {
    try {
      if (this.dirty)
        sessionStorage.setItem(
          this.draftKey,
          JSON.stringify({ base: this.saved, graph: this.graph }),
        );
      else sessionStorage.removeItem(this.draftKey);
    } catch {
      this.message("浏览器无法暂存草稿，请及时保存架构。", true);
    }
  }
  history(direction) {
    const from = this[direction],
      to = this[direction === "undo" ? "redo" : "undo"];
    if (!from.length) return;
    to.push(clone(this.graph));
    this.graph = from.pop();
    this.selected = null;
    this.layout();
    this.draft();
    this.render();
    this.inspect();
  }
  fit() {
    const n = [
      ...this.graph.nodes,
      ...this.routes.flatMap((route) =>
        route.points.map((point) => ({ ...point, width: 0, height: 0 })),
      ),
    ];
    if (!n.length) return;
    const x = Math.min(...n.map((n) => n.x)) - 65,
      y = Math.min(...n.map((n) => n.y)) - 65,
      right = Math.max(...n.map((n) => n.x + n.width)) + 65,
      bottom = Math.max(...n.map((n) => n.y + n.height)) + 65;
    const aspect =
      (this.svg.clientWidth || 850) / (this.svg.clientHeight || 540);
    const w = Math.max(right - x, (bottom - y) * aspect, 600);
    this.view = { x, y, w, h: w / aspect };
    this.render();
  }
  async onClick(e) {
    if (!this.alive) return;
    const b = e.target.closest("[data-dfd]");
    if (!b) return;
    const action = b.dataset.dfd;
    if (action === "save") {
      const submitted = normalizedArchitecture(this.graph),
        serialized = JSON.stringify(submitted);
      b.disabled = true;
      try {
        await this.options.onSave(submitted);
        // A React refresh may already have replaced this instance. Never clear a newer draft.
        try {
          const draft = JSON.parse(sessionStorage.getItem(this.draftKey));
          if (
            draft &&
            draft.base === this.saved &&
            JSON.stringify(draft.graph) === serialized
          )
            sessionStorage.removeItem(this.draftKey);
        } catch {}
        if (this.alive) {
          this.saved = serialized;
          this.draft();
        }
      } catch (err) {
        if (this.alive) this.message(err.message, true);
      } finally {
        if (this.alive) {
          b.disabled = false;
          this.render();
        }
      }
      return;
    }
    if (action === "undo" || action === "redo") {
      this.history(action);
      return;
    }
    if (action === "motion") {
      this.flowMotion = !this.flowMotion;
      try {
        sessionStorage.setItem(
          this.draftKey + ".flow-motion",
          this.flowMotion ? "on" : "off",
        );
      } catch {}
      this.render();
      return;
    }
    if (action === "focus") {
      this.root.classList.toggle("canvas-focused");
      this.fit();
      return;
    }
    if (action === "numbers") {
      const before = clone(this.graph);
      this.graph.show_numbers = this.graph.show_numbers === false;
      this.commit(before, false);
      return;
    }
    if (action === "view-mode") {
      const before = clone(this.graph);
      if (
        this.graph.view_mode === "dfd" &&
        this.graph.nodes.some((n) =>
          ["process_step", "data_store"].includes(n.kind),
        )
      ) {
        this.message(
          "当前含独立处理过程或数据存储，请保留展开 DFD 模式。",
          true,
        );
        return;
      }
      this.graph.view_mode =
        this.graph.view_mode === "dfd" ? "entities" : "dfd";
      this.commit(before, false);
      return;
    }
    if (action === "export-static") {
      let output;
      try {
        output = staticDiagramSvg(
          numberedGraph(this.graph, this.options?.getCodes?.(this.graph)),
          this.routes,
          this.bridgePath.bind(this),
        );
      } catch (error) {
        this.message(error.message, true);
        return;
      }
      const blob = new Blob([output], { type: "image/svg+xml;charset=utf-8" }),
        url = URL.createObjectURL(blob),
        link = document.createElement("a");
      link.href = url;
      link.download = "实体与数据流-静态图.svg";
      link.click();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
      this.message(
        "已导出纯图 SVG，保留当前通道间距与标签；无动画或附加明细。",
      );
      return;
    }
    if (action === "fit") {
      this.fit();
      return;
    }
    if (action === "lanes") {
      const before = clone(this.graph);
      this.graph.equal_spacing = !this.graph.equal_spacing;
      this.commit(before);
      return;
    }
    if (
      [
        ...Object.keys(kinds).map((kind) => "add-" + kind),
        "add-selected-entity",
      ].includes(action)
    ) {
      const selectedKind =
          action === "add-selected-entity"
            ? this.root.querySelector("[name=new-entity-type]").value
            : action.slice(4),
        kind = selectedKind === "ecu" ? "entity" : selectedKind,
        before = clone(this.graph);
      const node = {
        id: "node_" + crypto.randomUUID(),
        name: kinds[kind],
        kind: ["boundary", "process_step", "data_store"].includes(kind)
          ? kind
          : "entity",
        ...(["boundary", "process_step", "data_store"].includes(kind)
          ? {}
          : {
              entityType: kind === "entity" ? "ecu" : kind,
              children: [],
              attributes: [],
            }),
        x:
          Math.round(
            (this.view.x + 70 + (this.graph.nodes.length % 3) * 280) / 20,
          ) * 20,
        y:
          Math.round(
            (this.view.y + 80 + Math.floor(this.graph.nodes.length / 3) * 180) /
              20,
          ) * 20,
        width: kind === "boundary" ? 480 : 240,
        height: kind === "boundary" ? 300 : 110,
        description: "",
        reference: null,
      };
      this.graph.nodes.push(node);
      this.selected = node.id;
      this.commit(before);
      return;
    }
    if (
      [
        "apply-node",
        "add-attribute",
        "remove-attribute",
        "add-software",
        "add-store",
        "remove-child",
        "assign-store",
      ].includes(action)
    ) {
      const node = this.graph.nodes.find((n) => n.id === this.selected);
      if (!node) return;
      const before = clone(this.graph);
      try {
        const value = this.readNodeEditor(node, {
          skipAttributeIndex:
            action === "remove-attribute" ? Number(b.dataset.index) : -1,
          skipChildId: action === "remove-child" ? b.dataset.child : null,
        });
        Object.assign(node, value);
        if (action === "add-attribute") {
          if (node.attributes.length >= 50)
            throw new Error("每个实体最多维护 50 项属性。");
          let index = 1,
            name = "新属性";
          while (node.attributes.some((attribute) => attribute.name === name))
            name = "新属性 " + ++index;
          node.attributes.push({ name, value: "" });
        }
        if (action === "add-software" || action === "add-store") {
          if (node.children.length >= 100)
            throw new Error("每个实体最多维护 100 项内部组成。");
          node.children.push({
            id: "child_" + crypto.randomUUID(),
            kind: action === "add-software" ? "software" : "store",
            name: action === "add-software" ? "新软件" : "新数据存储",
            description: "",
            reference: null,
          });
          node.width = Math.max(node.width, 240);
          node.height = Math.max(node.height, 84 + node.children.length * 26);
        }
        if (action === "assign-store") {
          const parentId = this.inspector.querySelector(
            "[name=store-parent]",
          ).value;
          this.graph = assignLegacyStore(this.graph, node.id, parentId);
          this.selected = parentId;
        }
        this.commit(before);
        return;
      } catch (error) {
        this.graph = before;
        this.message(error.message, true);
        return;
      }
    }
    if (["tidy-route", "reset-route", "apply-flow-sides"].includes(action)) {
      const flow = this.graph.flows.find((f) => f.id === this.selected);
      if (!flow) return;
      const before = clone(this.graph);
      if (action === "apply-flow-sides") {
        const source = this.inspector.querySelector(
            "[name=flow-source-side]",
          ).value,
          target = this.inspector.querySelector(
            "[name=flow-target-side]",
          ).value;
        if ([source, target].some((side) => side && !sides.includes(side)))
          return this.message("请选择有效的连接边。", true);
        flow.sourceSide = source || null;
        flow.targetSide = target || null;
      } else {
        // An explicit empty layout suppresses legacy wireless waypoint fallback.
        flow.route_layout = {};
        if (action === "reset-route") flow.sourceSide = flow.targetSide = null;
      }
      this.commit(before);
      if (JSON.stringify(this.graph) !== JSON.stringify(before))
        this.message(
          action === "tidy-route"
            ? "已整理连线，保留已选连接边。"
            : action === "reset-route"
              ? "已恢复自动择边与布线。"
              : "已应用连接边。",
        );
      return;
    }
    if (action === "center-label") {
      const flow = this.graph.flows.find((f) => f.id === this.selected);
      if (!flow) return;
      const before = clone(this.graph);
      flow.labelOffset = { x: 0, y: 0 };
      this.commit(before, false);
      return;
    }
    if (action === "edit-flow") {
      const flow = this.graph.flows.find((f) => f.id === this.selected);
      this.options.onFlow(flow, (values) => {
        if (!this.alive) throw new Error("架构已更新，请重新打开数据流编辑。");
        const before = clone(this.graph);
        Object.assign(flow, values);
        this.commit(before);
      });
      return;
    }
    if (action === "remove") {
      const before = clone(this.graph),
        id = this.selected;
      this.graph.nodes = this.graph.nodes.filter((n) => n.id !== id);
      this.graph.flows = this.graph.flows.filter(
        (f) => f.id !== id && f.source !== id && f.target !== id,
      );
      this.selected = null;
      this.commit(before);
      return;
    }
  }
  pointerDown(e) {
    if (!this.alive || e.button !== 0 || this.pointerId != null) return;
    e.preventDefault();
    this.svg.focus({ preventScroll: true });
    this.pointerId = e.pointerId;
    this.svg.setPointerCapture(e.pointerId);
    const p = this.point(e),
      port = e.target.closest("[data-port]"),
      node = e.target.closest("[data-node]"),
      flow = e.target.closest("[data-flow]"),
      label = e.target.closest("[data-flow-label]"),
      endpoint = e.target.closest("[data-flow-end]"),
      segment = e.target.closest("[data-flow-segment]");
    if (endpoint || segment) {
      const f = this.graph.flows.find((f) => f.id === flow?.dataset.flow),
        route = this.routes.find((r) => r.id === f?.id);
      if (!f || !route) return;
      this.selected = f.id;
      const part =
        segment &&
        editableSegments(route.points).find(
          (s) => s.index === Number(segment.dataset.flowSegment),
        );
      this.drag = {
        type: endpoint ? "endpoint" : "route",
        id: f.id,
        start: p,
        role: endpoint?.dataset.flowEnd,
        part,
        before: clone(this.graph),
        routes: clone(this.routes),
      };
      this.render();
      this.inspect();
      return;
    }
    if (label) {
      const f = this.graph.flows.find((f) => f.id === label.dataset.flowLabel);
      if (!f) return;
      this.selected = f.id;
      this.drag = {
        type: "label",
        id: f.id,
        start: p,
        offset: clone(f.labelOffset || { x: 0, y: 0 }),
        before: clone(this.graph),
      };
      this.render();
      this.inspect();
      return;
    }
    if (port) {
      const source = this.graph.nodes.find((n) => n.id === node?.dataset.node);
      if (!isDiagramEndpoint(source, this.graph)) {
        this.message("数据流只能从实体连接到另一个实体。", true);
        return;
      }
      this.drag = {
        type: "connect",
        source: node.dataset.node,
        side: port.dataset.port,
        start: p,
        current: p,
      };
      e.preventDefault();
      return;
    }
    if (node) {
      this.selected = node.dataset.node;
      const n = this.graph.nodes.find((n) => n.id === this.selected);
      this.drag = {
        type: "move",
        id: n.id,
        start: p,
        x: n.x,
        y: n.y,
        before: clone(this.graph),
      };
      this.render();
      this.inspect();
      e.preventDefault();
      return;
    }
    if (flow) {
      this.selected = flow.dataset.flow;
      this.render();
      this.inspect();
      return;
    }
    this.selected = null;
    this.drag = { type: "pan", start: p, view: clone(this.view) };
    this.render();
    this.inspect();
  }
  pointerMove(e) {
    if (!this.alive || !this.drag || e.pointerId !== this.pointerId) return;
    const d = this.drag,
      p = this.point(e);
    if (d.type === "endpoint" || d.type === "route") {
      this.graph = clone(d.before);
      this.routes = clone(d.routes);
      d.invalid = false;
      const f = this.graph.flows.find((f) => f.id === d.id);
      if (Math.hypot(p.x - d.start.x, p.y - d.start.y) < 3) {
        d.attachment = null;
        this.render();
        return;
      }
      if (d.type === "endpoint") {
        const node = this.graph.nodes.find((n) => n.id === f[d.role]);
        d.attachment = attachmentAt(node, p);
        if (!d.attachment) {
          d.invalid = true;
          this.message("请将圆点放到原实体的另一侧；连接对象保持不变。");
          this.render();
          return;
        }
        const original = d.routes.find((r) => r.id === f.id);
        if (original[d.role]?.side === d.attachment.side) {
          this.render();
          return;
        }
        f[d.role + "Side"] = d.attachment.side;
        const other = d.role === "source" ? "target" : "source";
        f[other + "Side"] = original[other].side;
      } else {
        if (!d.part) return;
        const anchor = { ...d.part.point };
        anchor[d.part.axis] += p[d.part.axis] - d.start[d.part.axis];
        if (Math.abs(p[d.part.axis] - d.start[d.part.axis]) < 3) {
          this.render();
          return;
        }
        const waypoint = snapRoutePoint(anchor),
          originalRoute = d.routes.find((r) => r.id === f.id);
        for (const endpoint of [
          originalRoute.points[0],
          originalRoute.points.at(-1),
        ])
          if (Math.abs(waypoint[d.part.axis] - endpoint[d.part.axis]) <= 6)
            waypoint[d.part.axis] = endpoint[d.part.axis];
        f.route_layout = { waypoint };
        // Moving a segment must not silently move its ports to different sides.
        const original = d.routes.find((r) => r.id === f.id);
        f.sourceSide = original.source.side;
        f.targetSide = original.target.side;
      }
      const candidate = routeConnections(this.graph);
      if (candidate.conflicts.length) {
        this.graph = clone(d.before);
        d.invalid = true;
        this.message("这个位置无法保持连通和避障，松手将保留原线路。", true);
      } else {
        this.routes = candidate.routes;
        this.message(
          d.type === "endpoint"
            ? "松手确认连接边；数据流方向与连接对象不变。"
            : "线路已吸附并保持直角；松手确认，Esc 取消。",
        );
      }
      this.render();
      return;
    }
    if (d.type === "label") {
      const f = this.graph.flows.find((f) => f.id === d.id);
      const offset = labelOffsetForDrag(d.offset, d.start, p);
      if (
        (p.x === d.start.x && p.y === d.start.y) ||
        (offset.x === d.offset.x && offset.y === d.offset.y)
      ) {
        const original = d.before.flows.find((flow) => flow.id === d.id);
        if (Object.hasOwn(original, "labelOffset"))
          f.labelOffset = clone(original.labelOffset);
        else delete f.labelOffset;
      } else f.labelOffset = offset;
      this.render();
      return;
    }
    if (d.type === "connect") {
      d.current = p;
      this.render();
      return;
    }
    if (d.type === "move") {
      const n = this.graph.nodes.find((n) => n.id === d.id);
      n.x = Math.round((d.x + p.x - d.start.x) / 10) * 10;
      n.y = Math.round((d.y + p.y - d.start.y) / 10) * 10;
      this.render(true);
      return;
    }
    if (d.type === "pan") {
      this.view.x += d.start.x - p.x;
      this.view.y += d.start.y - p.y;
      this.render();
    }
  }
  pointerUp(e) {
    if (!this.alive || e.pointerId !== this.pointerId) return;
    this.pointerId = null;
    if (this.svg.hasPointerCapture(e.pointerId))
      this.svg.releasePointerCapture(e.pointerId);
    if (!this.drag) return;
    const d = this.drag;
    this.drag = null;
    if (d.type === "endpoint" || d.type === "route") {
      if (d.invalid) {
        this.graph = d.before;
        this.routes = d.routes;
        this.render();
        this.inspect();
        return;
      }
      this.commit(d.before);
      if (JSON.stringify(this.graph) !== JSON.stringify(d.before))
        this.message(
          d.type === "endpoint"
            ? "连接边已调整，连接对象与流向保持不变。"
            : "线路已调整并对齐；可点击整理连线清除手工绕行。",
        );
      this.render();
      this.inspect();
      return;
    }
    if (d.type === "label") {
      this.commit(d.before, false);
      return;
    }
    if (d.type === "move") {
      this.commit(d.before);
      return;
    }
    if (d.type === "connect") {
      const target = document
        .elementFromPoint(e.clientX, e.clientY)
        ?.closest("[data-node]");
      if (!target || target.dataset.node === d.source) return this.render();
      const node = this.graph.nodes.find((n) => n.id === target.dataset.node);
      if (!isDiagramEndpoint(node, this.graph)) {
        this.message(
          "数据流只能连接不同实体；内部软件、存储和信任边界不作端点。",
          true,
        );
        return this.render();
      }
      const port = document
        .elementFromPoint(e.clientX, e.clientY)
        ?.closest("[data-port]");
      const f = {
        id: "flow_" + crypto.randomUUID(),
        name: "",
        source: d.source,
        target: node.id,
        sourceSide: d.side,
        targetSide: port?.dataset.port || null,
        protocol: "",
        description: "",
      };
      this.render();
      this.options.onFlow({ ...f, id: null }, (values) => {
        if (!this.alive) throw new Error("架构已更新，请重新连接数据流。");
        const before = clone(this.graph);
        this.graph.flows.push({ ...f, ...values });
        this.selected = f.id;
        this.commit(before);
      });
    }
  }
  bridgePath(route, routes) {
    const pts = route.points;
    if (!pts.length) return "";
    let path = `M ${pts[0].x} ${pts[0].y}`;
    for (let i = 1; i < pts.length; i++) {
      const a = pts[i - 1],
        b = pts[i];
      if (Math.abs(a.y - b.y) < 0.001) {
        const crossings = [];
        for (const other of routes) {
          if (other.id === route.id) continue;
          for (let j = 1; j < other.points.length; j++) {
            const c = other.points[j - 1],
              d = other.points[j];
            if (
              Math.abs(c.x - d.x) < 0.001 &&
              c.x > Math.min(a.x, b.x) + 8 &&
              c.x < Math.max(a.x, b.x) - 8 &&
              a.y > Math.min(c.y, d.y) + 1 &&
              a.y < Math.max(c.y, d.y) - 1
            )
              crossings.push(c.x);
          }
        }
        const dir = b.x > a.x ? 1 : -1;
        for (const x of [...new Set(crossings)].sort((x, y) => (x - y) * dir)) {
          path += ` L ${x - dir * 4} ${a.y} Q ${x} ${a.y - 8} ${x + dir * 4} ${a.y}`;
        }
      }
      path += ` L ${b.x} ${b.y}`;
    }
    return path;
  }
  labelMarkup(route) {
    const flow = (this.displayGraph || this.graph).flows.find(
      (f) => f.id === route.id,
    );
    if (!flow) return "";
    return labelMarkup(flow, route, {
      selected: this.selected === flow.id,
      dfd: this.graph.view_mode === "dfd",
      dragging: this.drag?.type === "label" && this.drag.id === flow.id,
    });
  }
  render(moving = false) {
    this.displayGraph = numberedGraph(
      this.graph,
      this.options?.getCodes?.(this.graph),
    );
    const e = this.esc;
    if (moving) {
      const candidate = routeConnections(this.graph);
      if (!candidate.conflicts.length) this.routes = candidate.routes;
    }
    const routes = this.routes;
    this.svg.setAttribute(
      "viewBox",
      `${this.view.x} ${this.view.y} ${this.view.w} ${this.view.h}`,
    );
    const renderNode = (node) =>
      nodeMarkup(
        this.displayGraph.nodes.find((n) => n.id === node.id) || node,
        {
          selected: this.selected === node.id,
          dfd: this.graph.view_mode === "dfd",
        },
      );
    this.svg.innerHTML = `<defs><pattern id="dots" width="20" height="20" patternUnits="userSpaceOnUse"><circle cx="1" cy="1" r=".8" fill="#d9d9d9"/></pattern><marker id="arrowhead" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse"><path d="M0 0L10 5L0 10Z" fill="#8c8c8c"/></marker>${["normal", "critical", "internal", "energy"].map((kind) => `<marker id="arrowhead-${kind}" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse"><path d="M0 0L10 5L0 10Z" fill="${flowColor(kind === "energy" ? { channel_kind: kind } : { emphasis: kind })}"/></marker>`).join("")}</defs><rect x="${this.view.x}" y="${this.view.y}" width="${this.view.w}" height="${this.view.h}" fill="url(#dots)"/>${this.graph.nodes
      .filter((n) => n.kind === "boundary")
      .map(renderNode)
      .join("")}${routes
      .map((r) => {
        const f = this.graph.flows.find((f) => f.id === r.id);
        if (!f) return "";
        const dash = flowDashArray(f);
        return `<g data-flow="${e(r.id)}"><path d="${this.bridgePath(r, routes)}" fill="none" stroke="transparent" stroke-width="14"/><path class="flow-line ${this.selected === r.id ? "selected" : ""}" style="stroke:${this.selected === r.id ? "#1677ff" : flowColor(f)};stroke-width:${this.graph.view_mode === "dfd" ? 2.5 : 1.6}" d="${this.bridgePath(r, routes)}" ${dash ? `stroke-dasharray="${dash}"` : ""} marker-end="url(#arrowhead-${f.channel_kind === "energy" ? "energy" : f.emphasis || "normal"})"/>${this.flowMotion ? `<path class="flow-pulse" d="${this.bridgePath(r, routes)}" aria-hidden="true"/>` : ""}</g>`;
      })
      .join("")}${this.graph.nodes
      .filter((n) => n.kind !== "boundary")
      .map(renderNode)
      .join(
        "",
      )}${routes.map((r) => this.labelMarkup(r)).join("")}${connectionHandles(
      this.graph.flows.find((f) => f.id === this.selected),
      routes.find((r) => r.id === this.selected),
    )}${this.drag?.type === "endpoint" && this.drag.attachment ? `<circle cx="${this.drag.attachment.point.x}" cy="${this.drag.attachment.point.y}" r="11" fill="none" stroke="${this.drag.invalid ? "#ff4d4f" : "#1677ff"}" stroke-width="2" stroke-dasharray="3 3" pointer-events="none"/>` : ""}${this.drag?.type === "connect" ? `<path d="M ${this.drag.start.x} ${this.drag.start.y} L ${this.drag.current.x} ${this.drag.current.y}" stroke="#1677ff" stroke-width="2" stroke-dasharray="5 4" pointer-events="none"/>` : ""}`;
    const modeButton = this.root.querySelector("[data-dfd=view-mode]");
    if (modeButton)
      modeButton.textContent =
        this.graph.view_mode === "dfd" ? "展开 DFD" : "实体视图";
    const numberButton = this.root.querySelector("[data-dfd=numbers]");
    if (numberButton) {
      numberButton.setAttribute(
        "aria-pressed",
        this.graph.show_numbers !== false,
      );
      numberButton.classList.toggle(
        "primary",
        this.graph.show_numbers !== false,
      );
    }
    const palette = this.root.querySelector("[name=new-entity-type]");
    if (palette && this.paletteMode !== (this.graph.view_mode || "entities")) {
      const value = palette.value;
      palette.innerHTML = Object.entries({
        ...entityTypes,
        ...(this.graph.view_mode === "dfd"
          ? { process_step: "处理过程", data_store: "数据存储" }
          : {}),
      })
        .map(([key, label]) => `<option value="${key}">${label}</option>`)
        .join("");
      if ([...(palette.options || [])].some((o) => o.value === value))
        palette.value = value;
      this.paletteMode = this.graph.view_mode || "entities";
    }
    this.root.querySelector("[data-dfd=undo]").disabled = !this.undo.length;
    this.root.querySelector("[data-dfd=redo]").disabled = !this.redo.length;
    const lanes = this.root.querySelector("[data-dfd=lanes]");
    lanes.setAttribute("aria-pressed", this.graph.equal_spacing);
    lanes.classList.toggle("primary", this.graph.equal_spacing);
    const motion = this.root.querySelector("[data-dfd=motion]");
    motion.setAttribute("aria-pressed", this.flowMotion);
    motion.classList.toggle("primary", this.flowMotion);
    this.root.querySelector(".save-state").textContent =
      (this.dirty ? "未保存更改 · " : "已保存 · ") +
      `${this.graph.nodes.filter((n) => isDiagramEndpoint(n, this.graph)).length} 个节点 / ${this.graph.flows.length} 条连接${this.graph.nodes.some((n) => n.kind === "store") ? " · 有存储待归属" : ""}`;
  }
  readNodeEditor(node, { skipAttributeIndex = -1, skipChildId = null } = {}) {
    const field = (name) => this.inspector.querySelector(`[name=${name}]`),
      value = clone(node);
    value.name = field("node-name").value.trim();
    value.description = field("node-description").value;
    value.width = Number(field("node-width").value);
    value.height = Number(field("node-height").value);
    if (!value.name) throw new Error("请填写实体名称。");
    if (
      !Number.isFinite(value.width) ||
      !Number.isFinite(value.height) ||
      value.width < 60 ||
      value.width > 3000 ||
      value.height < 40 ||
      value.height > 3000
    )
      throw new Error("尺寸超出范围：宽 60–3000，高 40–3000。");
    if (isEntity(node)) {
      value.kind = "entity";
      value.entityType = field("entity-type").value;
      const osFields = field("os-family");
      if (osFields) {
        const os = operatingSystemForSave(
          {
            family: osFields.value,
            ...Object.fromEntries(
              ["name", "version", "architecture", "notes"].map((key) => [
                key,
                field("os-" + key).value,
              ]),
            ),
          },
          node.operating_system,
        );
        if (os !== undefined) value.operating_system = os;
      }
      value.attributes = Array.from(
        this.inspector.querySelectorAll("[data-attribute-row]"),
      )
        .filter(
          (row) => Number(row.dataset.attributeRow) !== skipAttributeIndex,
        )
        .map((row) => ({
          name: row.querySelector("[name=attribute-name]").value.trim(),
          value: row.querySelector("[name=attribute-value]").value,
        }));
      if (value.attributes.some((attr) => !attr.name))
        throw new Error("请填写属性名称，或移除空白属性。");
      if (
        new Set(value.attributes.map((attr) => attr.name.toLocaleLowerCase()))
          .size !== value.attributes.length
      )
        throw new Error("同一实体的属性名称不能重复。");
      value.children = Array.from(
        this.inspector.querySelectorAll("[data-child-row]"),
      )
        .filter((row) => row.dataset.childRow !== skipChildId)
        .map((row) => {
          const original = (node.children || []).find(
            (child) => child.id === row.dataset.childRow,
          );
          return {
            ...original,
            name: row.querySelector("[name=child-name]").value.trim(),
            description: row.querySelector("[name=child-description]").value,
          };
        });
      if (value.children.some((child) => !child.name))
        throw new Error("请填写内部软件或数据存储的名称。");
    }
    return value;
  }
  inspectNode(node) {
    const e = this.esc,
      entity = isEntity(node),
      legacyStore = node.kind === "store";
    const base = `<p class="eyebrow">${node.kind === "boundary" ? "信任边界" : legacyStore ? "待归属记录" : "实体"}</p><h3>${e(entityKindLabel(node))}</h3><div class="field"><label for="node-name">名称</label><input id="node-name" name="node-name" value="${e(node.name)}" maxlength="200"></div>${
      entity
        ? `<div class="field"><label for="entity-type">实体类型</label><select id="entity-type" name="entity-type">${Object.entries(
            entityTypes,
          )
            .map(
              ([key, title]) =>
                `<option value="${key}" ${entityType(node) === key ? "selected" : ""}>${title}</option>`,
            )
            .join("")}</select></div>`
        : ""
    }<div class="form-grid"><div class="field"><label for="node-width">宽度</label><input id="node-width" name="node-width" type="number" min="60" max="3000" value="${node.width}"></div><div class="field"><label for="node-height">高度</label><input id="node-height" name="node-height" type="number" min="40" max="3000" value="${node.height}"></div></div><div class="field"><label for="node-description">说明</label><textarea id="node-description" name="node-description" maxlength="4000">${e(node.description)}</textarea></div>`;
    const attributes = entity
      ? `<section class="entity-inspector-section"><h3>实体属性</h3><p>记录型号、硬件版本、部署位置等属性。</p>${(node.attributes || []).map((attr, index) => `<div class="entity-attribute-row" data-attribute-row="${index}"><input name="attribute-name" aria-label="属性 ${index + 1} 名称" placeholder="属性名称" value="${e(attr.name)}" maxlength="200"><input name="attribute-value" aria-label="属性 ${index + 1} 内容" placeholder="属性内容" value="${e(attr.value)}" maxlength="2000"><button class="btn small danger" data-dfd="remove-attribute" data-index="${index}" aria-label="移除属性 ${index + 1}">×</button></div>`).join("")}<button class="btn small" data-dfd="add-attribute">＋ 添加属性</button></section>`
      : "";
    const os = { ...blankOperatingSystem(), ...node.operating_system };
    const operatingSystem = entity
      ? `<details class="entity-inspector-section" ${node.operating_system ? "open" : ""}><summary style="cursor:pointer;font-weight:600;margin-bottom:10px">操作系统资料</summary><div class="field"><label for="os-family">系统类别</label><select id="os-family" name="os-family">${Object.entries(
          osFamilies,
        )
          .map(
            ([key, label]) =>
              `<option value="${key}" ${os.family === key ? "selected" : ""}>${label}</option>`,
          )
          .join("")}</select></div>${[
          ["name", "系统名称", 200, "例如：FreeRTOS、Ubuntu"],
          ["version", "版本", 200, "例如：24.04 LTS"],
          ["architecture", "处理器架构", 100, "例如：ARM64、x86_64"],
        ]
          .map(
            ([key, label, max, placeholder]) =>
              `<div class="field"><label for="os-${key}">${label}</label><input id="os-${key}" name="os-${key}" value="${e(os[key])}" maxlength="${max}" placeholder="${placeholder}"></div>`,
          )
          .join(
            "",
          )}<div class="field"><label for="os-notes">备注</label><textarea id="os-notes" name="os-notes" rows="3" maxlength="4000" placeholder="补充实时性、虚拟化或多个系统的部署情况">${e(os.notes)}</textarea></div><p>按实际部署填写；无操作系统的设备可选择裸机。</p></details>`
      : "";
    const children = entity
      ? `<section class="entity-inspector-section"><h3>内部组成 <span class="entity-count">${(node.children || []).length}</span></h3><p>软件与数据存储归属当前实体；数据流连接实体之间。</p>${(node.children || []).map((child, index) => `<div class="entity-child-editor" data-child-row="${e(child.id)}"><div class="entity-child-heading"><strong>${child.kind === "store" ? "数据存储" : "软件"}</strong><button class="btn small danger" data-dfd="remove-child" data-child="${e(child.id)}" aria-label="移除内部组成 ${index + 1}">移除</button></div><label>名称<input name="child-name" aria-label="内部组成 ${index + 1} 名称" value="${e(child.name)}" maxlength="200"></label><label>说明<textarea name="child-description" aria-label="内部组成 ${index + 1} 说明" rows="2" maxlength="4000">${e(child.description)}</textarea></label></div>`).join("")}<div class="entity-child-actions"><button class="btn small" data-dfd="add-software">＋ 软件</button><button class="btn small" data-dfd="add-store">＋ 数据存储</button></div></section>`
      : "";
    const migration = legacyStore
      ? `<section class="entity-inspector-section"><h3>设置所属实体</h3><p>此数据存储尚未归属实体。请选择所属实体；归属后保留原标识、描述与引用，同时转接相关的数据流和资产关联。</p><div class="field"><label for="store-parent">所属实体</label><select id="store-parent" name="store-parent"><option value="">请选择实体</option>${this.graph.nodes
          .filter(isEntity)
          .map(
            (parent) =>
              `<option value="${e(parent.id)}">${e(parent.name)}</option>`,
          )
          .join(
            "",
          )}</select></div><button class="btn small primary" data-dfd="assign-store">归入实体内部</button></section>`
      : "";
    this.inspector.innerHTML =
      base +
      operatingSystem +
      attributes +
      children +
      `<button class="btn small primary" data-dfd="apply-node">应用修改</button>` +
      migration +
      `<div class="section-gap"><button class="btn small danger" data-dfd="remove">${node.kind === "boundary" ? "移除边界" : legacyStore ? "移除数据存储" : "移除实体"}</button></div><p>修改和移除可在保存前撤销。已关联资产的实体需先解除关联。</p>`;
  }
  inspect() {
    const e = this.esc,
      n = this.graph.nodes.find((n) => n.id === this.selected),
      f = this.graph.flows.find((f) => f.id === this.selected);
    if (n) {
      this.inspectNode(n);
      return;
    }
    if (f) {
      this.inspector.innerHTML = `<p class="eyebrow">DATA FLOW</p><h3>${e(f.name)}</h3>${communicationDetails(
        f,
      )
        .slice(0, 2)
        .map(
          (line) =>
            `<p style="white-space:pre-wrap;overflow-wrap:anywhere">${e(line)}</p>`,
        )
        .join("")}
      <button class="btn small" data-dfd="edit-flow">编辑数据流</button>
      <div class="section-gap"><h3>连接与布线</h3><p>${e(this.graph.nodes.find((n) => n.id === f.source)?.name || f.source)} → ${e(this.graph.nodes.find((n) => n.id === f.target)?.name || f.target)}</p>
      <p>圆点：拖到所属实体另一侧换边。<br>方点：拖动线段，自动吸附并保持直角。<br>只改变绘图位置，不改变连接对象与流向。</p>
      <div class="form-grid">${["source", "target"]
        .map(
          (role) =>
            `<div class="field"><label for="flow-${role}-side">${role === "source" ? "发送端出线" : "接收端入线"}</label><select id="flow-${role}-side" name="flow-${role}-side">${Object.entries(
              {
                "": "自动择短",
                top: "上侧",
                right: "右侧",
                bottom: "下侧",
                left: "左侧",
              },
            )
              .map(
                ([key, title]) =>
                  `<option value="${key}" ${(flowSide(f, role) || "") === key ? "selected" : ""}>${title}</option>`,
              )
              .join("")}</select></div>`,
        )
        .join("")}</div>
      <button class="btn small" data-dfd="apply-flow-sides">应用连接边</button>
      <p>${flowWaypoint(f) ? "已手动调整途经位置" : "自动避障布线"}</p>
      <div class="connection-actions"><button class="btn small" data-dfd="tidy-route" title="清除手工途经点，保留已选连接边，重新寻找简洁路线">整理连线</button><button class="btn small" data-dfd="reset-route" title="同时恢复自动选择连接边和路线">恢复自动</button></div>
      <p>整理连线会清除手工绕行，保留连接边；恢复自动会重新选择连接边。调整支持撤销，按 Esc 取消拖动。</p></div>
      <div class="section-gap"><h3>标签位置</h3><p>${f.labelOffset?.x || f.labelOffset?.y ? "已手动调整 · 跟随线路移动" : "默认居中"}</p><button class="btn small" data-dfd="center-label" ${!f.labelOffset?.x && !f.labelOffset?.y ? "disabled" : ""}>恢复居中</button><p>协议和无线图标随标签一起移动。</p></div>
      <details class="section-gap"><summary>通信详情</summary>${communicationDetails(
        f,
      )
        .map((line) => `<p>${e(line)}</p>`)
        .join(
          "",
        )}</details><div class="section-gap"><button class="btn small danger" data-dfd="remove">移除数据流</button></div>`;
      return;
    }
    if (this.graph.view_mode === "dfd") {
      this.inspector.innerHTML =
        '<p class="eyebrow">EXPANDED DFD</p><h3>展开数据流图</h3><p>外部实体、硬件接口、处理过程与数据存储分层展示；边界框用于表示信任域。</p><p>拖动节点调整位置，拖动四侧连接点创建连接。选中连接可修改方向接点、走线和标签。</p><p>红色：安全关键流；灰色：内部流；棕色：电能流。电能流不自动生成传输数据资产。</p><p>显示编号可切换自动编号前缀；原图编号可保留在名称中。专注画布可展开绘图区。</p><div class="notice">导出图始终静态。图形不替代威胁、影响及风险评估。</div>';
      return;
    }
    this.inspector.innerHTML =
      '<p class="eyebrow">系统架构</p><h3>构建系统视图</h3><p>在工具栏选择设备、应用、云服务、参与者等实体类型。软件和数据存储在实体内部维护。</p><p>选中实体可维护操作系统、属性与内部组成。拖动四侧连接点到另一实体即可创建数据流，再设置有线、无线或逻辑连接及协议。</p><div class="notice green">画布支持流向动画，导出使用静态图。</div><p>所有数据流的标签默认居中，可拖动调整；无线直接以虚线和协议图标标识。选中连线后拖动圆点换边、方点调整线路。<br>Ctrl + 滚轮缩放<br>空白处拖动平移<br>Ctrl / ⌘ + Z 撤销</p>';
  }
}
