import React from "react";
import { createRoot } from "react-dom/client";
import { ConfigProvider, App as AntApp } from "antd";
import zhCN from "antd/locale/zh_CN";
import Workbench from "./Workbench.jsx";
import "antd/dist/reset.css";
import "./styles.css";
createRoot(document.getElementById("root")).render(
  <ConfigProvider
    locale={zhCN}
    theme={{
      token: {
        colorPrimary: "#1677ff",
        colorBgLayout: "#f5f7fa",
        colorText: "#1f2329",
        colorTextSecondary: "#687281",
        fontSize: 14,
        borderRadius: 6,
        controlHeight: 34,
        fontFamily:
          '-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif',
      },
      components: {
        Layout: { headerBg: "#fff", siderBg: "#fff" },
        Menu: { itemBorderRadius: 4, itemHeight: 42 },
        Table: { headerBg: "#fafbfc", cellPaddingBlock: 14 },
        Card: { headerHeight: 52 },
      },
    }}
  >
    <AntApp>
      <Workbench />
    </AntApp>
  </ConfigProvider>,
);
