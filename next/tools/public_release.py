"""Prepare an audited source-only staging tree. This tool never creates or pushes a Git repository."""
from pathlib import Path, PurePosixPath
import argparse
import ast
import datetime
import hashlib
import json
import os
import re
import shutil
import struct
import zlib
import tempfile
import unicodedata

PROJECT = Path(__file__).resolve().parents[2]
BUILDERS = {'build_gb44495.py', 'build_gb44496.py', 'build_gbt44464.py', 'build_un_bundles.py', 'build_r156_amendment.py'}
TOOLS = {'build-go.sh', 'build-api-docs.py', 'public_release.py', 'test_public_release.py', 'collect_public_knowledge.py', 'test_public_knowledge.py'}
BLOCKED_PARTS = {'.git', '.work', '.runtime', '.venv', 'node_modules', '__pycache__', 'var', 'bin', 'web', 'dist', 'private', 'backups', 'retired-tests', 'source-snapshot', 'testdata', 'fixtures'}
BLOCKED_SUFFIXES = {'.pdf', '.docx', '.xlsx', '.zip', '.tar', '.gz', '.tgz', '.sqlite', '.sqlite3', '.db', '.pyc', '.log'}
SCREENSHOTS = {'next/docs/screenshots/architecture.png', 'next/docs/screenshots/assets.png', 'next/docs/screenshots/numbering.png', 'next/docs/screenshots/quality-center.jpg', 'next/docs/screenshots/model-services.jpg', 'next/docs/screenshots/automation-center.jpg', 'next/docs/screenshots/attack-path-presets.jpg', 'next/docs/screenshots/project-management.jpg'}
SOURCE_SUFFIXES = {'.go', '.json', '.html', '.js', '.jsx', '.mjs', '.cjs', '.ts', '.tsx', '.css', '.svg', '.txt', '.md', '.mod', '.sum', '.sh', '.py'}
IDENTIFIERS = re.compile(rb'\b(?:systems|losses|patterns|techniques|controls|relationships|checks|methods|criteria|collections|gb|r155):[A-Za-z0-9_-]+')
# Assembled from parts so this audit pattern cannot match the staging tool's own source.
LOCAL_OR_CREDENTIAL = '|'.join(
    [re.escape(root) + r'/[^/\s]+/' for root in ('/Users', '/home')]
    + [r'(?:ghp_|github_pat_)[A-Za-z0-9_]{20,}']
)
EMPTY_MANIFEST = {'schema_version': 1, 'verified_at': None, 'documents': []}
PUBLIC_README = '''# 界衡 · 威胁建模平台

以 ISO/SAE 21434 的工作产品和追溯关系为基础，面向汽车、物联网、机器人及其他软硬件产品的本地威胁建模工作台。使用 **Go + SQLite** 后端、**React + Ant Design** 前端，支持从系统架构、资产和威胁分析，到风险处置、需求验证和静态报告的完整流程。

## 主要能力

| 环节 | 能力 |
| --- | --- |
| 项目管理 | 维护项目名称与范围说明，记录修订与审计；项目基本信息与分析工作产品独立管理 |
| 相关项与 DFD | 实体模式或展开 DFD；信任边界、内部软件/存储、操作系统、数据流与电能流；有线/无线协议标注 |
| 布线与图形 | 拖动端点换边、调整线段和标签、网格整理；无线链路以虚线及协议标识呈现；流动效果仅用于画布，导出保持静态 |
| 资产识别 | 保存架构时同步生成资产、保留人工修改和风险关联；导入项目支持维护已有资产目录 |
| TARA 工作产品 | 独立损害场景、安全目标；多资产/多属性威胁关联、攻击路径、CAL、需求及接受条件 |
| 风险方法 | SFOP 最大值、权重或系数法；直接用系数总分计算风险，或先分级再查矩阵；支持五因子攻击潜力 |
| 处置与验证 | 区分残余风险目标和已验证结果；目标、需求或评估依据变化后，旧证据和处置需重新复核 |
| 法规引用 | 本地条款目录、版本定位和勾选引用；公开仓库初始为空，由使用者接入资料 |
| 工作簿与报告 | 约定格式 XLSX 预览与原子导入、原文件保留；当前工作产品、来源快照和审计记录导出 |
| 企业模型 | 四种协议、用途模型、加密凭据、显式备用链、健康与用量、无密钥配置交换 |
| 辅助分析 | 图片转原生 DFD、全流程草案、质量复核；来源追溯、人工采用、可恢复 |
| 自动分析 | 8 条内置确定性规则、命中理由与前提、人工采用、去重和还原点，无需模型 |
| 攻击路径 | 16 类预置模板、按资产查看匹配线索、批量加入可编辑草稿；保留模板版本与项目路径编号 |
| 企业知识 | 8 个公开漏洞分析案例、版本化待复核导入、CISA KEV 增量收集工具；语义和关键词检索 |
| 内网交付 | 账号角色、项目成员、确定性交付检查、还原点与独立备份恢复 |
| 开发接口 | 版本化资源 API、OpenAPI、并发修订检查、幂等写入和统一错误格式 |

平台不替代工程判断或标准符合性评审；算法由项目确认，不能把某一套评分矩阵视为标准强制值。

## 界面预览

以下界面使用演示项目。

项目列表汇总资产、风险场景与评审基线，支持直接维护项目基本信息。

![项目管理](next/docs/screenshots/project-management.jpg)

实体间统一布线，协议直接标在连线上；端点、线段和标签可手工调整。

![实体与数据流](next/docs/screenshots/architecture.png)

保存架构后自动生成资产；已有目录可保留原编号和人工维护内容。

![自动资产清单](next/docs/screenshots/assets.png)

“项目设置 → 编号规则”支持按类型配置前缀、分隔符与数字位数，画布、清单和报告统一显示。

![自定义编号规则](next/docs/screenshots/numbering.png)

模型服务支持统一配置与连接检查；交付检查汇总工作产品完整性和待处理事项。图中的模型连接为演示测试服务。

![模型服务](next/docs/screenshots/model-services.jpg)

![交付检查](next/docs/screenshots/quality-center.jpg)

不依赖模型的规则分析，审阅后加入待评估工作产品。

![自动分析](next/docs/screenshots/automation-center.jpg)

按资产条件筛选攻击路径模板，补充项目事实后继续评级；已添加模板会标记并避免重复。

![攻击路径预置](next/docs/screenshots/attack-path-presets.jpg)

## 从源码运行

准备 Go（版本见 `next/server/go.mod`）、Node.js 22.12 或更新的兼容版本，以及 npm。在仓库根目录执行：

```sh
./next/tools/build-go.sh
npm ci --prefix next/client
npm run build --prefix next/client
./start.sh
```

打开 http://127.0.0.1:8790 。默认数据文件为 `next/var/assurance.sqlite3`，服务只监听回环地址。需要独立端口或数据库时直接启动：

```sh
./next/bin/jieheng --root . --database next/var/demo.sqlite3 --port 8791
```

源码仓库不附运行数据库、项目报告、本机二进制或前端构建产物。法规目录初始为空，可直接进行建模与开发测试。按 [法规接入说明](next/data/regulations/README.md) 在本地接入有权使用的资料；应用不会自动联网获取正文。

## 企业内网与辅助分析

企业模式通过同机 HTTPS 反向代理提供登录及项目权限；本机模式仍可独立使用。接入企业模型后，可从图片生成原生 DFD，结合自有知识生成全流程分析草案，审阅采用后继续人工验证和报告交付。模型输出以草案形式采用，验证与签署需在相应流程中独立记录。详见 [企业部署](next/ENTERPRISE_DEPLOYMENT.md) 和 [模型服务与辅助分析](next/AI_ASSISTANCE.md)。

## TARA 导入与导出

创建空白项目，在“TARA 工作产品”中选择 XLSX，检查预览后导入。导入会核对编号、引用关系和原始评分；失败时整体回滚，不覆盖已有工作产品。损害场景、目标、需求和处置成为可编辑记录，同时保留完整来源表页及字节一致的原文件。

当前提供 `tara-chinese-v1` 约定版式适配器，**不支持任意 Excel 自动识别**。表头、编号、计算方法和大小限制见 [TARA 工作簿接入](next/TARA_IMPORT.md)。导入后的残余风险仍是设计目标，确认和签署状态按实际记录展示。

标准 XLSX 导出包含 17 张基础工作表；使用扩展 TARA 工作产品时增加 6 张追溯表，导入项目再追加全部来源表页快照。采用 AI 时按需增加辅助分析追溯及待确认建议表；采用确定性规则时增加自动化分析追溯表。当前资产元数据和编辑结果进入工作产品表，原表快照保持不变。报告图形为静态；来源公式作为文本保留，不执行。见 [Excel 导出格式](next/EXCEL_EXPORT.md)。

## 开发与验证

```sh
(cd next/server && go test -race ./... && go vet ./...)
node --test next/tests/*.mjs next/tests/*.cjs
python3 next/tools/test_public_release.py
```

前端布线测试读取构建产物，请先完成前端构建。Go 测试使用临时数据库和独立构造的工作簿、法规数据。GitHub Actions 执行后端、前端和公开源码范围检查。

- [工作流设计参考](next/PRODUCT_RESEARCH.md)
- [攻击路径模板](next/ATTACK_PATH_TEMPLATES.md)
- [确定性自动分析](next/AUTOMATION.md) · [公开知识积累](next/PUBLIC_KNOWLEDGE.md)
- [功能和开发说明](next/README.md)
- [领域与模块](next/ARCHITECTURE.md)
- [版本化 API](next/API_CONVENTIONS.md) · [OpenAPI](next/server/openapi.json)
- [扩展流程、算法和报告](next/EXTENDING.md)
- [后端运行与数据兼容](next/GO_MIGRATION.md)

## 部署范围与扩展

当前版本支持单企业、单实例内网部署和本机工作台。尚未提供 SSO/MFA、合格电子签名、审批流、分布式高可用或大规模向量服务。集成测试使用固定响应服务验证协议和流程；实际识图与分析质量取决于配置的模型，应使用部署环境中的代表性样例评估。跨行业流程、额外工作簿版式和专用报告可按扩展文档二次开发。接口服务不应直接作为公网多租户服务部署。

源码仓库包含应用代码、测试、公开案例分析、演示截图和开发文档。运行数据库、项目资料和法规正文由部署方管理，不随源码分发。仓库目前未选定开源许可证；公开可见不表示已授予额外的使用、修改或再分发许可，第三方依赖遵循各自许可证。
'''
REGULATIONS_README = '''# 本地法规资料接入

本公开源码版本不附带任何法规或标准正文、PDF、Formex ZIP、用户上传资料或预生成条款库。`manifest.json` 与 `manifest.sample.json` 均为可运行的空目录；空库不会生成示例法规或隐式联网下载。

## 目录契约

`manifest.json` 使用 `schema_version: 1`，`documents` 为数组。无资料时保持空数组。接入文档时，条目须包含独立 `id`、`code`、`title`、`edition`、`source_url`、`coverage` 和 `clause_count`。正文列表通过 `clauses_file` 引用目录内文件，并填写其 `clauses_sha256`；原件通过 `artifacts` 声明目录内路径、媒体类型与 SHA-256。

覆盖状态仅有 `metadata_only`、`partial`、`complete_for_edition`。仅元数据不可包含可引用正文；完整版本必须有正文和 `coverage_evidence`。不得把未取得、未核对或仅部分提取的材料声明为完整版本。

条款列表每项至少包含 `key`（`reg:{document_id}:{locator}`）、`document_id`、`number`、`title`、`body` 和 `parent`。父级必须属于同一版本且不存在循环。物理页码用 `page` / `page_end`，印刷页码用 `print_page`；多原件版本可使用 `source_file` 与 `source_pages` 明确定位。具体约束见 `server/internal/assurance/regulations.go` 与原创测试。

版本或修改单使用独立文档 ID。不要覆盖原始版本，也不要根据引用自动判断符合性。加载器校验所有声明文件的路径、摘要及条款关系；缺失或损坏的已声明文件会报错，不会静默变成空库。

## 可选离线构建器

保留的 `build_*.py` 是针对指定版式与来源摘要编写的解析代码，正文从调用者自行准备的本地文件读取，未内嵌法规全文。Python 仅用于可选资料处理，不是服务运行依赖；相关构建器使用 `pypdf` 或 `pdfplumber`，按脚本导入要求安装。

GB 44495、GB 44496 构建器接收命令行 PDF 路径；GB/T 44464 构建器读取指定本地 PDF；UN 构建器需要其脚本列明的本地原件与工作目录。不同来源或版本应先独立核对，再调整解析器、摘要和覆盖证据，不能为通过校验直接替换摘要。构建器生成待核对条款与清单片段，不会自动更新运行目录。

接入资料前自行确认使用及分发权限。`sources/` 和生成的正文 JSON 已在公开仓库的忽略规则中排除；不要将个人、客户资料或未获分发许可的全文提交到公开仓库。
'''
CI = '''name: Source checks
on:
  push:
  pull_request:
permissions:
  contents: read
jobs:
  go:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: next/server/go.mod
          cache-dependency-path: next/server/go.sum
      - run: go test -race ./...
        working-directory: next/server
      - run: go vet ./...
        working-directory: next/server
  frontend:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '22'
          cache: npm
          cache-dependency-path: next/client/package-lock.json
      - run: npm ci --prefix next/client
      - run: npm run build --prefix next/client
      - run: node --test next/tests/*.mjs next/tests/*.cjs
  source-scope:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: python3 next/tools/test_public_release.py
      - run: python3 next/tools/test_public_knowledge.py
'''
IGNORE = '''.work/
.runtime/
.venv/
__pycache__/
*.pyc
.env
.env.*
**/node_modules/
next/var/
next/bin/
next/web/
dist/
output/
*.xlsx
*.docx
*.pdf
*.sqlite*
*.key
*.pem
*.p12
*.db
*.db-wal
*.db-shm
next/data/regulations/sources/
next/data/regulations/*.json
!next/data/regulations/manifest.json
!next/data/regulations/manifest.sample.json
'''


def source_allowed(name):
    path = PurePosixPath(name)
    if path.is_absolute() or '..' in path.parts or any(part in BLOCKED_PARTS for part in path.parts):
        return False
    if path.suffix.lower() in BLOCKED_SUFFIXES or path.name.startswith('.env') or path.name.endswith(('-wal', '-shm')):
        return False
    if name in SCREENSHOTS:
        return True
    if name in {'start.sh', 'start.py'}:
        return True
    if len(path.parts) < 2 or path.parts[0] != 'next':
        return False
    if len(path.parts) == 2:
        return path.suffix == '.md'
    group = path.parts[1]
    if group == 'data':
        if name in {'next/data/public-knowledge/seed.json', 'next/data/public-knowledge/README.md'}: return True
        return len(path.parts) == 4 and path.parts[2] == 'regulations' and path.name in BUILDERS
    if group == 'tools':
        return len(path.parts) == 3 and path.name in TOOLS
    if group == 'tests':
        return len(path.parts) == 3 and path.name.startswith('test_') and path.suffix in {'.mjs', '.cjs', '.js'}
    if group in {'server', 'client'}:
        return path.suffix.lower() in SOURCE_SUFFIXES or path.name == '.gitignore'
    return False


def selected_sources(project):
    files = []
    for name in ('start.sh', 'start.py'):
        p = project / name
        if p.is_file() and not p.is_symlink(): files.append(p)
    for directory, subdirs, names in os.walk(project / 'next', followlinks=False):
        subdirs[:] = sorted(name for name in subdirs if name not in BLOCKED_PARTS and not (Path(directory) / name).is_symlink())
        for name in sorted(names):
            p = Path(directory) / name
            if p.is_file() and not p.is_symlink() and source_allowed(p.relative_to(project).as_posix()): files.append(p)
    return sorted(set(files))


def normalized(text):
    return ''.join(unicodedata.normalize('NFKC', text).split())


def local_fingerprints(project):
    """Use local exclusion evidence without placing that evidence in the public staging tree."""
    file = project / 'next/tools/private-material-fingerprints.json'
    evidence = json.loads(file.read_text()) if file.exists() else {'file_sha256': [], 'identifier_sha256': [], 'normalized_text_windows': []}
    legal_windows = set()
    for source in (project / 'next/data/regulations').glob('*.json'):
        if source.name.startswith('manifest'): continue
        try: rows = json.loads(source.read_text())
        except (ValueError, OSError): continue
        if not isinstance(rows, list): continue
        for row in rows:
            if not isinstance(row, dict): continue
            body = normalized(str(row.get('body', '')))
            if len(body) >= 96:
                for sample in (body[:96], body[-96:]): legal_windows.add(hashlib.sha256(sample.encode()).hexdigest())
    return evidence, legal_windows


def transformed(name, data):
    if name.startswith('next/data/regulations/build_'):
        text = data.decode('utf-8')
        text = re.sub(r"^SOURCE = Path\([^\n]+\)$", "SOURCE = Path('provide-your-authorized-input.pdf')", text, flags=re.M)
        return text.encode()
    return data


def validate_jfif_screenshot(data):
    if len(data)>5_000_000: raise ValueError('Screenshot too large')
    offset, frame = 2, False
    while offset+4<=len(data):
        if data[offset]!=255: raise ValueError('Invalid JPEG segment')
        marker=data[offset+1];length=int.from_bytes(data[offset+2:offset+4],'big');end=offset+2+length
        if length<2 or end>len(data): raise ValueError('Invalid JPEG segment length')
        if marker==0xe0:
            if length!=16 or data[offset+4:offset+9]!=b'JFIF\0': raise ValueError('Unsupported JPEG metadata')
        elif marker==0xc0:
            if length<11 or not (0<int.from_bytes(data[offset+7:offset+9],'big')<=16000 and 0<int.from_bytes(data[offset+9:offset+11],'big')<=16000): raise ValueError('Invalid JPEG dimensions')
            frame=True
        elif marker==0xda:
            if not frame: raise ValueError('Missing JPEG frame')
            i=end
            while i<len(data):
                if data[i]!=255: i+=1;continue
                if i+1>=len(data): break
                next_marker=data[i+1]
                if next_marker==0 or 0xd0<=next_marker<=0xd7: i+=2;continue
                if next_marker==0xd9 and i+2==len(data): return
                raise ValueError('Unsupported JPEG scan or trailing metadata')
            raise ValueError('Incomplete JPEG image')
        elif marker not in (0xdb,0xc4,0xdd): raise ValueError('Screenshot contains unsupported metadata')
        offset=end
    raise ValueError('Incomplete JPEG screenshot')


def validate_screenshot(data):
    """Only reviewed PNG/JFIF screenshots without textual metadata are permitted."""
    if data.startswith(b'\xff\xd8'):
        return validate_jfif_screenshot(data)
    if not data.startswith(b'\x89PNG\r\n\x1a\n') or len(data) > 5_000_000:
        raise ValueError('Invalid public screenshot')
    offset, kinds = 8, []
    while offset + 12 <= len(data):
        length = struct.unpack('>I', data[offset:offset+4])[0]
        end = offset + 12 + length
        kind = data[offset+4:offset+8]
        if end > len(data) or kind not in {b'IHDR', b'IDAT', b'IEND', b'sRGB', b'gAMA', b'cHRM', b'pHYs'}:
            raise ValueError('Screenshot contains unsupported metadata or invalid chunks')
        if zlib.crc32(data[offset+4:end-4]) != struct.unpack('>I', data[end-4:end])[0]:
            raise ValueError('Screenshot checksum mismatch')
        kinds.append(kind); offset = end
        if kind == b'IEND': break
    if offset != len(data) or not kinds or kinds[0] != b'IHDR' or kinds[-1] != b'IEND' or b'IDAT' not in kinds:
        raise ValueError('Incomplete public screenshot')


def audit(stage, evidence, legal_windows):
    generated = {'README.md', '.gitignore', '.github/workflows/ci.yml', '.public-release-stage.json', 'next/data/regulations/README.md', 'next/data/regulations/manifest.json', 'next/data/regulations/manifest.sample.json'}
    identifiers, files = set(evidence['identifier_sha256']), set(evidence['file_sha256'])
    windows = {}
    for row in evidence['normalized_text_windows']:
        windows.setdefault(row['characters'], set()).add(row['sha256'])
    if legal_windows: windows.setdefault(96, set()).update(legal_windows)
    records = []
    for p in sorted(stage.rglob('*')):
        if p.is_symlink(): raise ValueError('Symlink is not allowed in public staging')
        if not p.is_file(): continue
        name = p.relative_to(stage).as_posix()
        if name not in generated and not source_allowed(name): raise ValueError('Excluded public path: ' + name)
        data = p.read_bytes(); digest = hashlib.sha256(data).hexdigest()
        if digest in files: raise ValueError('Excluded source fingerprint: ' + name)
        if any(hashlib.sha256(match.group()).hexdigest() in identifiers for match in IDENTIFIERS.finditer(data)):
            raise ValueError('Excluded identifier fingerprint: ' + name)
        if name in SCREENSHOTS:
            validate_screenshot(data)
            records.append({'path': name, 'sha256': digest, 'bytes': len(data)})
            continue
        text = data.decode('utf-8')
        if re.search(LOCAL_OR_CREDENTIAL, text):
            raise ValueError('Local path or credential-shaped content: ' + name)
        clean = normalized(text)
        for length, hashes in windows.items():
            for i in range(max(0, len(clean)-length+1)):
                if hashlib.sha256(clean[i:i+length].encode()).hexdigest() in hashes:
                    raise ValueError('Excluded text fingerprint: ' + name)
        if name in {'next/data/regulations/manifest.json', 'next/data/regulations/manifest.sample.json'} and json.loads(text) != EMPTY_MANIFEST:
            raise ValueError('Public regulation manifest must be empty')
        if p.suffix == '.py': ast.parse(text, filename=name)
        records.append({'path': name, 'sha256': digest, 'bytes': len(data)})
    return records


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', type=Path, default=PROJECT / '.work/public-release')
    parser.add_argument('--refresh', action='store_true', help='Replace only a prior audited staging tree without Git history; preserve it as a sibling backup')
    args = parser.parse_args(argv)
    output = args.output.resolve()
    if output == PROJECT or PROJECT.is_relative_to(output): raise ValueError('Output must not replace the working project')
    if output.exists() and (not args.refresh or not (output / '.public-release-stage.json').is_file() or (output / '.git').exists()):
        raise ValueError('Output exists; refresh is allowed only for generated staging without .git')
    evidence, legal_windows = local_fingerprints(PROJECT)
    output.parent.mkdir(parents=True, exist_ok=True)
    temporary = Path(tempfile.mkdtemp(prefix='public-release-build-', dir=output.parent))
    try:
        for source in selected_sources(PROJECT):
            name = source.relative_to(PROJECT).as_posix(); target = temporary / name
            target.parent.mkdir(parents=True, exist_ok=True); target.write_bytes(transformed(name, source.read_bytes()))
            if source.stat().st_mode & 0o111: target.chmod(0o755)
        generated = {'README.md': PUBLIC_README, '.gitignore': IGNORE, '.github/workflows/ci.yml': CI, 'next/data/regulations/README.md': REGULATIONS_README,
                     'next/data/regulations/manifest.json': json.dumps(EMPTY_MANIFEST, indent=2)+'\n', 'next/data/regulations/manifest.sample.json': json.dumps(EMPTY_MANIFEST, indent=2)+'\n',
                     '.public-release-stage.json': json.dumps({'generator':'next/tools/public_release.py','scope':'source-only','regulation_documents':0}, indent=2)+'\n'}
        for name, content in generated.items():
            target=temporary/name;target.parent.mkdir(parents=True,exist_ok=True);target.write_text(content)
        records = audit(temporary, evidence, legal_windows)
        if output.exists():
            stamp=datetime.datetime.now(datetime.timezone.utc).strftime('%Y%m%dT%H%M%S%fZ');output.rename(output.with_name(output.name+'.previous-'+stamp))
        temporary.rename(output)
    finally:
        if temporary.exists(): shutil.rmtree(temporary)
    report={'scope':'source-only','created_at':datetime.datetime.now(datetime.timezone.utc).isoformat(),'files':records,'regulation_documents':0,'binary_artifacts':0,'screenshot_artifacts':sum(r['path'] in SCREENSHOTS for r in records),'contains_private_fingerprint_manifest':False,'network_operations':False,'published':False}
    audit_file=output.with_suffix('.audit.json');audit_file.write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'staging':str(output),'audit':str(audit_file),'files':len(records),'regulation_documents':0,'published':False}))


if __name__ == '__main__': main()
