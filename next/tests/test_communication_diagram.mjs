import test from "node:test";
import assert from "node:assert/strict";
import {
  blankCommunication, blankOperatingSystem, communicationForDisplay,
  communicationForSave, operatingSystemForSave, technologies, flowDashArray,
  wirelessBadge,
} from "../client/src/communication.js";
import { entityTypes, nodeMarkup, labelMarkup, staticDiagramSvg } from "../client/src/architecture-view.js";
import { Diagram } from "../client/src/diagram.js";

const entity = (id, x = 0) => ({ id, name: id, kind: "entity", entityType: "device", x, y: 0, width: 240, height: 130, description: "", attributes: [], children: [] });
const flow = (overrides = {}) => ({ id: "f", name: "状态上报", source: "device", target: "cloud", protocol: "", description: "", ...overrides });
const route = { id: "f", points: [{ x: 240, y: 65 }, { x: 400, y: 65 }] };

test("legacy protocol recognition is presentation only, does not classify range or assume security", () => {
  for (const [protocol, medium, technology] of [["CAN FD", "wired", "CAN FD"], ["DoIP", "wired", "DoIP"], ["BLE", "wireless", "Bluetooth/BLE"], ["Wi-Fi", "wireless", "Wi-Fi"], ["NFC", "wireless", "NFC"]]) {
    const original = flow({ protocol }), before = structuredClone(original), display = communicationForDisplay(original);
    assert.equal(display.medium, medium); assert.equal(display.technology, technology);
    assert.equal(display.range, "unspecified"); assert.deepEqual(display.security_properties, []);
    assert.equal(display.inferred, true); assert.deepEqual(original, before);
  }
  assert.equal(communicationForDisplay(flow({ protocol: "HTTPS" })).medium, "unspecified");
  assert.equal(communicationForDisplay(flow({ protocol: "CAN", communication: blankCommunication() })).inferred, false);
});

test("all built-in technologies are displayable without misclassifying wired buses as wireless", () => {
  for (const [medium, values] of Object.entries(technologies)) for (const technology of values) {
    const display = communicationForDisplay(flow({ protocol: technology }));
    assert.equal(display.medium, medium, technology);
    assert.equal(display.technology, technology, technology);
  }
  assert.equal(communicationForDisplay(flow({ protocol: "application_thread" })).medium, "unspecified");
});

test("saving unrelated fields preserves omitted or partial optional objects", () => {
  assert.equal(communicationForSave(blankCommunication()), undefined);
  assert.equal(operatingSystemForSave(blankOperatingSystem()), undefined);
  const communication = { medium: "wired", technology: "UART" }, os = { family: "linux", name: "Ubuntu" };
  assert.deepEqual(communicationForSave({ ...blankCommunication(), ...communication }, communication), communication);
  assert.deepEqual(operatingSystemForSave({ ...blankOperatingSystem(), ...os }, os), os);
  assert.deepEqual(communicationForSave({ security_properties: ["integrity", "integrity"] }).security_properties, ["integrity"]);
  assert.equal(operatingSystemForSave(blankOperatingSystem(), os).family, "unknown");
});

test("wired, wireless and logical rendering have distinct line conventions and neutral wireless badges", () => {
  assert.equal(flowDashArray(flow({ communication: { medium: "wired", technology: "FlexRay" } })), "");
  assert.equal(flowDashArray(flow({ communication: { medium: "logical" } })), "2 4");
  for (const technology of ["NFC", "Bluetooth/BLE", "Wi-Fi", "自定义无线"]) {
    const wireless = flow({ communication: { medium: "wireless", technology } });
    assert.equal(flowDashArray(wireless), "7 5");
    assert.match(wirelessBadge(wireless), /communication-badge/);
    assert.match(labelMarkup(wireless, route), /communication-badge/);
  }
  assert.equal(wirelessBadge(flow({ protocol: "CAN" })), "");
  assert.match(wirelessBadge(flow({ protocol: "NFC" })), />NFC<\/text>/);
  assert.match(wirelessBadge(flow({ protocol: "BLE" })), />BT<\/text>/);
});

test("expanded entity types retain ports and OS summary while children remain internal", () => {
  for (const type of ["device", "application", "cloud", "actor", "ecu", "external", "service"]) {
    const node = { ...entity(type), entityType: type, operating_system: { family: "linux", name: "Yocto", version: "5.0" }, children: [{ id: "sw", kind: "software", name: "Agent", description: "" }] };
    const svg = nodeMarkup(node);
    assert.match(svg, /OS · Yocto 5.0/); assert.ok(svg.includes(entityTypes[type]));
    assert.equal((svg.match(/data-port=/g) || []).length, 4); assert.match(svg, /Agent/);
  }
  assert.doesNotMatch(nodeMarkup({ ...entity("boundary"), kind: "boundary", operating_system: { name: "ignored" } }), /OS ·/);
});

test("static export preserves full OS and communication metadata while escaping text and omitting animation", () => {
  const device = { ...entity("device"), height: 80, operating_system: { family: "rtos", name: "系统 <A>", version: "v2.3", architecture: "ARM64", notes: "宿主与客体\n客体系统 QNX 8.0" } };
  const wireless = flow({ protocol: "MQTT/TLS", description: "遥测数据 & 校验", labelOffset: { x: 20, y: -12 }, communication: { medium: "wireless", technology: "Wi-Fi", range: "local", purpose: "遥测", security_properties: ["integrity", "authenticity"] } });
  const graph = { nodes: [device, { ...entity("cloud", 400), entityType: "cloud" }], flows: [wireless] }, before = structuredClone(graph);
  const svg = staticDiagramSvg(graph, [route]);
  for (const text of ["系统 &lt;A&gt;", "v2.3", "ARM64", "宿主与客体", "客体系统 QNX 8.0", "MQTT/TLS", "覆盖范围：局域", "用途：遥测", "安全属性：完整性、真实性", "遥测数据 &amp; 校验", "device → cloud"])
    assert.ok(svg.includes(text), text);
  assert.match(svg, /stroke-dasharray="7 5"/); assert.match(svg, /marker-end="url\(#static-arrow\)"/); assert.equal((svg.match(/communication-badge/g) || []).length, 1);
  assert.doesNotMatch(svg, /wireless-endpoint|W01/);
  assert.doesNotMatch(svg, /<animate|@keyframes|animation:|flow-pulse|data-port=|data-flow-label=|选中实体查看|<script|<image/i);
  assert.deepEqual(graph, before);
});

test("entity inspector keeps OS absent on unrelated edits and stores all explicitly entered fields", () => {
  const node = entity("node"), d = Object.create(Diagram.prototype);
  const fields = { "node-name": "设备", "node-description": "说明", "node-width": 240, "node-height": 130, "entity-type": "device", ...Object.fromEntries(Object.entries(blankOperatingSystem()).map(([key, value]) => ["os-" + key, value])) };
  d.inspector = { querySelector: (selector) => ({ value: fields[selector.slice(6, -1)] }), querySelectorAll: () => [] };
  assert.equal(Object.hasOwn(d.readNodeEditor(node), "operating_system"), false);
  Object.assign(fields, { "os-family": "linux", "os-name": "Debian", "os-version": "12", "os-architecture": "x86_64", "os-notes": "虚拟机" });
  assert.deepEqual(d.readNodeEditor(node).operating_system, { family: "linux", name: "Debian", version: "12", architecture: "x86_64", notes: "虚拟机" });
  assert.equal(Object.hasOwn(node, "operating_system"), false);
});
