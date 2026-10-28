import React, { createContext, useContext, useEffect, useState } from "react";
import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  Space,
  Spin,
  Typography,
} from "antd";
import { SafetyCertificateOutlined } from "@ant-design/icons";
import { request } from "./api.js";
const AuthContext = createContext({
  mode: "local",
  user: { role: "admin", name: "本地管理员" },
});
export const useAuth = () => useContext(AuthContext);
export default function AuthGate({ children }) {
  const [session, setSession] = useState(null),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const refresh = async () => {
    try {
      setSession(await request("/auth/session"));
      setError("");
    } catch (e) {
      setError(e.message);
    }
  };
  useEffect(() => {
    refresh();
    const expired = () => setSession({ mode: "enterprise", user: null });
    window.addEventListener("jieheng-session-expired", expired);
    return () => window.removeEventListener("jieheng-session-expired", expired);
  }, []);
  const login = async (values) => {
    setBusy(true);
    setError("");
    try {
      setSession(
        await request("/auth/sessions", {
          method: "POST",
          body: JSON.stringify(values),
        }),
      );
    } catch (e) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };
  const logout = async () => {
    await request("/auth/logout", { method: "POST", body: "{}" });
    setSession({ mode: "enterprise", user: null });
  };
  if (!session)
    return (
      <div className="login-screen">
        <Card>
          {error ? (
            <>
              <Alert type="error" title={error} />
              <Button onClick={refresh}>重新连接</Button>
            </>
          ) : (
            <Spin tip="正在连接工作台" />
          )}
        </Card>
      </div>
    );
  if (session.mode === "enterprise" && !session.user)
    return (
      <div className="login-screen">
        <Card className="login-card">
          <Space align="center">
            <SafetyCertificateOutlined
              style={{ fontSize: 32, color: "#1677ff" }}
            />
            <Typography.Title level={3}>界衡 · 企业工作台</Typography.Title>
          </Space>
          <Typography.Paragraph type="secondary">
            使用企业管理员分配的账号登录，项目内容保存在企业部署环境中。
          </Typography.Paragraph>
          {error && <Alert type="error" title={error} showIcon />}
          <Form layout="vertical" onFinish={login}>
            <Form.Item
              name="username"
              label="账号"
              rules={[{ required: true }]}
            >
              <Input autoComplete="username" maxLength={80} />
            </Form.Item>
            <Form.Item
              name="password"
              label="口令"
              rules={[{ required: true }]}
            >
              <Input.Password autoComplete="current-password" maxLength={72} />
            </Form.Item>
            <Button htmlType="submit" type="primary" loading={busy} block>
              登录工作台
            </Button>
          </Form>
        </Card>
      </div>
    );
  return (
    <AuthContext.Provider value={{ ...session, logout, refresh }}>
      {children}
    </AuthContext.Provider>
  );
}
