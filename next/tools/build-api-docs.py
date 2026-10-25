"""Build dependency-free local API docs from the committed OpenAPI contract."""
from pathlib import Path
import html,json
root=Path(__file__).resolve().parents[1]
spec=json.loads((root/'server/openapi.json').read_text())
esc=html.escape
sections=[]
for path,ops in spec['paths'].items():
 for method,op in ops.items():
  if method not in ('get','post','put','patch','delete'):continue
  section=f'<section><h2><code>{method.upper()} /api/v1{esc(path)}</code></h2><h3>{esc(op.get("summary",""))}</h3><p>{esc(op.get("description",""))}</p>'
  section+='<details><summary>参数、请求与响应</summary><pre>'+esc(json.dumps(op,ensure_ascii=False,indent=2))+'</pre></details></section>'
  sections.append(section)
for name,schema in spec['components']['schemas'].items():
 sections.append(f'<section id="schema-{esc(name)}"><details><summary>{esc(name)}</summary><pre>{esc(json.dumps(schema,ensure_ascii=False,indent=2))}</pre></details></section>')
page='''<!doctype html><html lang="zh-CN"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>界衡 API 文档</title><style>body{font:15px/1.7 system-ui,sans-serif;max-width:1100px;margin:40px auto;padding:0 24px;color:#172b4d;background:#f6f8fb}section{padding:20px;margin:16px 0;background:white;border:1px solid #e3e8ef;border-radius:8px}h2{font-size:18px;overflow-wrap:anywhere}pre{overflow:auto;background:#f6f8fb;padding:16px;font-size:13px}a{color:#1769ce}summary{cursor:pointer;font-weight:600}</style><h1>界衡 · 工作产品 API</h1><p>版本化接口：/api/v1。<a href="/api/v1/openapi.json">下载 OpenAPI 3.1.1</a> · <a href="/">返回应用</a></p><p>写入使用 X-Workspace-Client: jieheng 与 Idempotency-Key（8–100 个可见 ASCII 字符）。项目内写入另带当前 ETag 作为 If-Match。旧版本 412，缺少版本 428，同一幂等键不同内容 409，校验错误 422。错误为 application/problem+json，含 request_id。本地标识不是身份认证。</p><p>新增资源 201，替换资源 200。所有操作采用事务，运行数据库不随交付包分发。当前为单机工具；自有参考资料接入、流程步骤及报告二次开发请参阅 EXTENDING.md。</p>'''+''.join(sections)+'</html>'
(root/'server/docs.html').write_text(page)
print('Built docs for',sum(len(v) for v in spec['paths'].values()),'operations')
