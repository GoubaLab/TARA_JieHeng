import { useCallback, useEffect, useRef, useState } from "react";
import { request } from "./api.js";
export function useResource(path) {
  const [data, setData] = useState(null),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(true),
    generation = useRef(0);
  const refresh = useCallback(async () => {
    const n = ++generation.current;
    setLoading(true);
    try {
      const value = await request(path);
      if (n === generation.current) {
        setData(value);
        setError("");
      }
    } catch (e) {
      if (n === generation.current) setError(e.message);
    } finally {
      if (n === generation.current) setLoading(false);
    }
  }, [path]);
  useEffect(() => {
    setData(null);
    refresh();
    return () => {
      generation.current++;
    };
  }, [refresh]);
  return { data, error, loading, refresh };
}
export function mutateResource(
  path,
  body,
  revision,
  method = "POST",
  key = crypto.randomUUID(),
) {
  return request(path, {
    method,
    headers: {
      "Idempotency-Key": key,
      ...(revision == null ? {} : { "If-Match": `"${revision}"` }),
    },
    body: JSON.stringify(body),
  });
}
export function downloadJSON(value, name) {
  const url = URL.createObjectURL(
    new Blob([JSON.stringify(value, null, 2)], { type: "application/json" }),
  );
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  a.click();
  setTimeout(() => URL.revokeObjectURL(url), 500);
}
export function pickFields(value, names) {
  return Object.fromEntries(
    names.filter((k) => value[k] !== undefined).map((k) => [k, value[k]]),
  );
}
export const providerProtocols = {
  openai_chat: "OpenAI 兼容 Chat",
  openai_responses: "OpenAI Responses",
  anthropic: "Anthropic Messages",
  ollama: "Ollama",
};
export const roleLabels = {
  admin: "管理员",
  reviewer: "评审人员",
  analyst: "分析人员",
  viewer: "只读成员",
};
export const purposeLabels = {
  diagram: "图片建模",
  analysis: "全流程分析",
  review: "质量复核",
  knowledge_index: "知识索引",
};
export const taskStates = {
  queued: ["等待执行", "default"],
  running: ["执行中", "processing"],
  ready: ["待审阅", "blue"],
  needs_revision: ["需要修订", "warning"],
  failed: ["执行失败", "error"],
  interrupted: ["已中断", "warning"],
  cancelled: ["已取消", "default"],
  adopted: ["已采用", "success"],
  completed: ["已完成", "success"],
};
