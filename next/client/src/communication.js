// Display conventions for architecture diagrams, not protocol security claims.
export const media = {
  unspecified: "未分类",
  wired: "有线",
  wireless: "无线",
  logical: "逻辑连接",
};
export const ranges = {
  unspecified: "未指定",
  near_field: "近场",
  local: "局域",
  wide_area: "广域",
};
export const osFamilies = {
  unknown: "未指定",
  bare_metal: "裸机 / 无操作系统",
  rtos: "实时操作系统（RTOS）",
  linux: "Linux",
  qnx: "QNX",
  windows: "Windows",
  android: "Android",
  ios: "iOS",
  macos: "macOS",
  other: "其他",
};
export const securityProperties = {
  confidentiality: "机密性",
  integrity: "完整性",
  availability: "可用性",
  authenticity: "真实性",
  authorization: "授权",
  non_repudiation: "不可否认性",
};
export const purposes = ["指令", "请求", "响应", "遥测", "日志", "升级"];
export const technologies = {
  wired: [
    "FlexRay",
    "CAN",
    "CAN FD",
    "DoIP",
    "Ethernet",
    "UART",
    "JTAG",
    "SWD/DAP",
    "USB",
    "LIN",
    "SPI",
    "I2C",
    "RS-485",
  ],
  wireless: [
    "NFC",
    "Wi-Fi",
    "Bluetooth/BLE",
    "UWB",
    "RFID",
    "Zigbee",
    "Thread",
    "LoRa",
    "蜂窝",
  ],
  logical: [],
  unspecified: [],
};
const legacyNames = [
  ["wireless", "Bluetooth/BLE", /\b(?:bluetooth|ble)\b|蓝牙/i],
  ["wireless", "Wi-Fi", /\bwi[ -]?fi\b|无线局域网/i],
  ["wireless", "NFC", /\bnfc\b/i],
  ["wireless", "UWB", /\buwb\b/i],
  ["wireless", "RFID", /\brfid\b/i],
  ["wireless", "Zigbee", /\bzigbee\b/i],
  ["wireless", "Thread", /^thread$/i],
  ["wireless", "LoRa", /\blora(?:wan)?\b/i],
  ["wireless", "蜂窝", /\b(?:cellular|lte|[345]g)\b|蜂窝/i],
  ["wired", "CAN FD", /\bcan[ _-]?fd\b/i],
  ["wired", "CAN", /\bcan\b/i],
  ["wired", "DoIP", /\bdoip\b/i],
  ["wired", "Ethernet", /\bethernet\b|以太网/i],
  ...[
    "FlexRay",
    "UART",
    "JTAG",
    "SWD/DAP",
    "USB",
    "LIN",
    "SPI",
    "I2C",
    "RS-485",
  ].map((name) => [
    "wired",
    name,
    new RegExp(
      `\\b${name === "SWD/DAP" ? "(?:swd|dap)" : name === "RS-485" ? "rs[ -]?485" : name}\\b`,
      "i",
    ),
  ]),
];
export function blankCommunication() {
  return {
    medium: "unspecified",
    technology: "",
    range: "unspecified",
    purpose: "",
    security_properties: [],
  };
}
export function blankOperatingSystem() {
  return {
    family: "unknown",
    name: "",
    version: "",
    architecture: "",
    notes: "",
  };
}
export function hasOperatingSystem(value) {
  return (
    !!value &&
    ((value.family && value.family !== "unknown") ||
      ["name", "version", "architecture", "notes"].some((key) => value[key]))
  );
}
export function operatingSystemSummary(value) {
  if (!hasOperatingSystem(value)) return "";
  return (
    [
      value.name ||
        (value.family !== "unknown" ? osFamilies[value.family] : ""),
      value.version,
    ]
      .filter(Boolean)
      .join(" ") || "已记录操作系统资料"
  );
}
export function communicationForDisplay(flow) {
  if (flow.communication)
    return { ...blankCommunication(), ...flow.communication, inferred: false };
  const known = legacyNames.find(([, , expression]) =>
    expression.test(flow.protocol || ""),
  );
  return {
    ...blankCommunication(),
    ...(known ? { medium: known[0], technology: known[1] } : {}),
    inferred: !!known,
  };
}
// Keep an absent optional object absent when users only edit another field.
export function communicationForSave(value, original) {
  const next = { ...blankCommunication(), ...value };
  next.security_properties = [...new Set(next.security_properties || [])];
  const empty =
    next.medium === "unspecified" &&
    !next.technology &&
    next.range === "unspecified" &&
    !next.purpose &&
    !next.security_properties.length;
  if (empty && !original) return undefined;
  if (
    original &&
    JSON.stringify(next) ===
      JSON.stringify({ ...blankCommunication(), ...original })
  )
    return structuredClone(original);
  return next;
}
export function operatingSystemForSave(value, original) {
  const next = { ...blankOperatingSystem(), ...value };
  if (!hasOperatingSystem(next) && !original) return undefined;
  if (
    original &&
    JSON.stringify(next) ===
      JSON.stringify({ ...blankOperatingSystem(), ...original })
  )
    return structuredClone(original);
  return next;
}
export function communicationCaption(flow) {
  const value = communicationForDisplay(flow);
  const parts = [value.technology, flow.protocol].filter(Boolean);
  return (
    [...new Set(parts)].join(" · ") ||
    (value.medium !== "unspecified" ? media[value.medium] : "")
  );
}
export function communicationDetails(flow) {
  const value = communicationForDisplay(flow);
  const details = [
    `传输方式：${media[value.medium] || value.medium}${value.inferred ? "（根据原协议识别，仅用于展示）" : ""}`,
    `通信技术：${value.technology || "未指定"}`,
    `覆盖范围：${ranges[value.range] || value.range}`,
    `上层协议 / 通道：${flow.protocol || "未指定"}`,
    `用途：${value.purpose || "未指定"}`,
    `安全属性：${value.security_properties?.length ? value.security_properties.map((key) => securityProperties[key] || key).join("、") : "未指定"}`,
  ];
  if (flow.description) details.push("传输内容与约束：" + flow.description);
  return details;
}
export const isWirelessFlow = (flow) =>
  communicationForDisplay(flow).medium === "wireless";

export function flowDashArray(flow) {
  if (flow.channel_kind === "energy") return "";
  const medium = communicationForDisplay(flow).medium;
  return medium === "wireless" ? "7 5" : medium === "logical" ? "2 4" : "";
}
// Neutral diagram symbols drawn here; they are not official protocol logos.
export function wirelessBadge(flow, x = 0, y = 0) {
  const value = communicationForDisplay(flow);
  if (value.medium !== "wireless") return "";
  const name = value.technology.toLowerCase();
  const icon =
    name === "nfc"
      ? '<text x="0" y="3" text-anchor="middle" font-size="7" fill="#536176">NFC</text>'
      : /bluetooth|ble/.test(name)
        ? '<text x="0" y="3" text-anchor="middle" font-size="8" fill="#536176">BT</text>'
        : '<path d="M-7-2Q0-8 7-2M-4 1Q0-3 4 1" fill="none" stroke="#536176" stroke-width="1.3"/><circle cx="0" cy="4" r="1.4" fill="#536176"/>';
  return `<g class="communication-badge" transform="translate(${x} ${y})"><rect x="-11" y="-10" width="22" height="20" rx="4" fill="#f5f7fa" stroke="#c9d2df"/>${icon}</g>`;
}

export function flowColor(flow) {
  return flow?.channel_kind === "energy"
    ? "#ad6800"
    : flow?.emphasis === "critical"
      ? "#b42318"
      : flow?.emphasis === "internal"
        ? "#9aa4b2"
        : "#687281";
}
