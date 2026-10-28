#!/usr/bin/env python3
"""Collect bounded CISA KEV snapshots into review-only knowledge candidates.

Network access is opt-in. This tool never connects to an application database.
State records emitted or unchanged CVE hashes, not a global import offset.
"""
from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import html
import json
import os
from pathlib import Path
import re
import sys
import tempfile
import unicodedata
import urllib.error
import urllib.parse
import urllib.request

KEV_URL = 'https://www.cisa.gov/sites/default/files/feeds/known_exploited_vulnerabilities.json'
CATALOG_URL = 'https://www.cisa.gov/known-exploited-vulnerabilities-catalog'
LICENSE_URL = 'https://github.com/cisagov/kev-data/blob/develop/LICENSE'
MAX_BYTES = 5 * 1024 * 1024
MAX_ENTRY_BATCH_BYTES = 1_750_000
MAX_PACK_BYTES = 1_900_000  # Below the API HTTP body limit, including notices and metadata.
CVE_RE = re.compile(r'CVE-[0-9]{4}-[0-9]{4,19}\Z')
CWE_RE = re.compile(r'CWE-[0-9]{1,10}\Z')
HASH_RE = re.compile(r'[0-9a-f]{64}\Z')
REQUIRED_FIELDS = {
    'cveID': 28, 'vendorProject': 256, 'product': 512,
    'vulnerabilityName': 512, 'dateAdded': 10, 'shortDescription': 12000,
    'requiredAction': 12000, 'dueDate': 10,
}
OPTIONAL_FIELDS = {'notes': 12000, 'knownRansomwareCampaignUse': 32, 'forensicTriage': 16}
STATE_SCHEMA = 'jieheng.kev-cursor/1'


class ValidationError(ValueError):
    """The source or cursor is incomplete or unsafe to treat as a snapshot."""


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def _no_duplicate_keys(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValidationError('JSON contains duplicate object keys')
        result[key] = value
    return result


def _invalid_constant(value):
    raise ValidationError('JSON contains a non-finite number')


def decode_json(raw: bytes):
    if len(raw) > MAX_BYTES:
        raise ValidationError('JSON exceeds the 5 MB limit')
    try:
        return json.loads(raw.decode('utf-8-sig'), object_pairs_hook=_no_duplicate_keys,
                          parse_constant=_invalid_constant)
    except (UnicodeError, json.JSONDecodeError, RecursionError) as exc:
        raise ValidationError('Expected bounded UTF-8 JSON') from exc


def read_json(path: Path):
    with path.open('rb') as stream:
        return decode_json(stream.read(MAX_BYTES + 1))


def fetch_kev():
    request = urllib.request.Request(KEV_URL, headers={
        'Accept': 'application/json', 'Accept-Encoding': 'identity',
        'User-Agent': 'JieHeng-Public-Knowledge/1',
    })
    opener = urllib.request.build_opener(NoRedirect())
    try:
        with opener.open(request, timeout=25) as response:
            if response.geturl() != KEV_URL or response.status != 200:
                raise ValidationError('Unexpected source URL or response status')
            if response.headers.get('Content-Encoding', 'identity').lower() != 'identity':
                raise ValidationError('Compressed responses are not accepted')
            content_type = response.headers.get('Content-Type', '').split(';', 1)[0].strip().lower()
            if content_type != 'application/json' and not content_type.endswith('+json'):
                raise ValidationError('The official source did not return JSON')
            length = response.headers.get('Content-Length')
            if length is not None:
                try:
                    size = int(length)
                except ValueError as exc:
                    raise ValidationError('Invalid response length') from exc
                if size < 0 or size > MAX_BYTES:
                    raise ValidationError('Response exceeds the 5 MB limit')
            return decode_json(response.read(MAX_BYTES + 1))
    except urllib.error.HTTPError as exc:
        if 300 <= exc.code < 400:
            raise ValidationError('The official source redirected; redirects are not followed') from exc
        raise ValidationError(f'Official source returned HTTP {exc.code}') from exc
    except urllib.error.URLError as exc:
        raise ValidationError('Could not read the official source; cursor unchanged') from exc


def text_field(value, field, maximum, *, empty=False):
    if not isinstance(value, str) or len(value) > maximum:
        raise ValidationError(f'Invalid string or length: {field}')
    if any(ord(char) < 32 and char not in '\t\r\n' for char in value):
        raise ValidationError(f'Control character in {field}')
    result = unicodedata.normalize('NFC', value).strip()
    if not result and not empty:
        raise ValidationError(f'Empty field: {field}')
    return result


def checked_url(value):
    if not isinstance(value, str) or len(value) > 2048 or any(char.isspace() for char in value):
        raise ValidationError('Invalid evidence URL')
    try:
        parsed = urllib.parse.urlsplit(value)
        if (parsed.scheme not in ('https', 'http') or not parsed.hostname
                or parsed.username is not None or parsed.password is not None
                or parsed.port is not None and not 1 <= parsed.port <= 65535
                or '\\' in value):
            raise ValidationError('Evidence URLs must be HTTP(S) without credentials')
    except ValueError as exc:
        raise ValidationError('Invalid evidence URL') from exc
    return value


def check_embedded_urls(value):
    # Notes can include prose and multiple links. Links are validated but never fetched.
    for match in re.finditer(r'[A-Za-z][A-Za-z0-9+.-]*://[^\s<>"\[\]]+', value):
        checked_url(match.group(0).rstrip('.,;'))


def valid_date(value, field):
    if not re.fullmatch(r'[0-9]{4}-[0-9]{2}-[0-9]{2}', value):
        raise ValidationError(f'Invalid date: {field}')
    try:
        dt.date.fromisoformat(value)
    except ValueError as exc:
        raise ValidationError(f'Invalid date: {field}') from exc


def valid_timestamp(value, field):
    match = re.fullmatch(
        r'(?P<base>[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2})'
        r'(?:\.[0-9]+)?(?P<offset>Z|[+-][0-9]{2}:[0-9]{2})', value)
    if not match:
        raise ValidationError(f'Invalid timestamp: {field}')
    try:
        # Older Python accepts only certain fractional precisions in fromisoformat.
        # Validate the calendar/time separately; retain the exact source value.
        dt.datetime.strptime(match.group('base'), '%Y-%m-%dT%H:%M:%S')
        offset = match.group('offset')
        if offset != 'Z' and (int(offset[1:3]) > 23 or int(offset[4:6]) > 59):
            raise ValueError('Invalid UTC offset')
    except ValueError as exc:
        raise ValidationError(f'Invalid timestamp: {field}') from exc


def validate_feed(feed):
    if not isinstance(feed, dict):
        raise ValidationError('Catalog must be an object')
    version = text_field(feed.get('catalogVersion'), 'catalogVersion', 64)
    if not re.fullmatch(r'[A-Za-z0-9._-]+', version):
        raise ValidationError('Invalid catalogVersion')
    released = text_field(feed.get('dateReleased'), 'dateReleased', 64)
    valid_timestamp(released, 'dateReleased')
    records = feed.get('vulnerabilities')
    count = feed.get('count')
    if (type(count) is not int or not isinstance(records, list) or count != len(records)
            or count < 1):
        # An empty snapshot must not be interpreted as a mass withdrawal.
        raise ValidationError('Catalog count must match a non-empty vulnerabilities array')
    normalized = {}
    for record in records:
        if not isinstance(record, dict):
            raise ValidationError('Each vulnerability must be an object')
        unknown = set(record) - REQUIRED_FIELDS.keys() - OPTIONAL_FIELDS.keys() - {'cwes'}
        if unknown:
            raise ValidationError('New KEV fields require adapter review')
        result = {}
        for field, maximum in REQUIRED_FIELDS.items():
            result[field] = text_field(record.get(field), field, maximum)
        for field, maximum in OPTIONAL_FIELDS.items():
            if field in record:
                result[field] = text_field(record[field], field, maximum, empty=field == 'notes')
        cve = result['cveID']
        if not CVE_RE.fullmatch(cve) or cve in normalized:
            raise ValidationError('Invalid or duplicate CVE ID')
        valid_date(result['dateAdded'], 'dateAdded')
        valid_date(result['dueDate'], 'dueDate')
        if 'knownRansomwareCampaignUse' in result and result['knownRansomwareCampaignUse'] not in ('Known', 'Unknown'):
            raise ValidationError('Unknown ransomware-use value requires adapter review')
        if 'forensicTriage' in result and result['forensicTriage'] not in ('Yes', 'No'):
            raise ValidationError('Unknown forensic-triage value requires adapter review')
        cwes = record.get('cwes', [])
        if not isinstance(cwes, list) or len(cwes) > 64 or any(not isinstance(v, str) or not CWE_RE.fullmatch(v) for v in cwes):
            raise ValidationError('Invalid CWE IDs')
        result['cwes'] = sorted(set(cwes))
        for value in result.values():
            if isinstance(value, str):
                check_embedded_urls(value)
        normalized[cve] = result
    return {'catalogVersion': version, 'dateReleased': released, 'records': normalized}


def validate_state(state):
    if not isinstance(state, dict) or state.get('schema') != STATE_SCHEMA:
        raise ValidationError('Unsupported cursor schema')
    records = state.get('records')
    if not isinstance(records, dict):
        raise ValidationError('Invalid cursor records')
    for cve, value in records.items():
        if (not CVE_RE.fullmatch(cve) or not isinstance(value, dict)
                or not isinstance(value.get('hash'), str) or not HASH_RE.fullmatch(value['hash'])):
            raise ValidationError('Invalid cursor record')
        if ('missing_reported_hash' in value and
                (not isinstance(value['missing_reported_hash'], str) or not HASH_RE.fullmatch(value['missing_reported_hash']))):
            raise ValidationError('Invalid missing-notice cursor')
    return state


def content_hash(record):
    canonical = json.dumps(record, ensure_ascii=False, sort_keys=True, separators=(',', ':'))
    return hashlib.sha256(canonical.encode('utf-8')).hexdigest()


def escaped(value):
    # Source text remains inert Markdown data even when it contains HTML or links.
    value = html.escape(str(value), quote=True)
    return re.sub(r'([\\`*{}_\[\]()#+.!|>~-])', r'\\\1', value)


def make_entry(record, catalog, captured_at, digest):
    cve = record['cveID']
    facts = '\n'.join([
        f"- CVE：{cve}",
        f"- 厂商 / 项目：{escaped(record['vendorProject'])}",
        f"- 产品：{escaped(record['product'])}",
        f"- 目录名称：{escaped(record['vulnerabilityName'])}",
        f"- 加入日期：{record['dateAdded']}",
        f"- 目录短描述：{escaped(record['shortDescription'])}",
        f"- 目录要求的处置：{escaped(record['requiredAction'])}",
        f"- 目录中的期限：{record['dueDate']}（仅记录源值，不自动作为本项目合规期限）",
        f"- 勒索软件利用字段：{escaped(record.get('knownRansomwareCampaignUse', '未提供'))}（Unknown 不等于没有利用）",
        f"- 取证分流字段：{escaped(record.get('forensicTriage', '未提供'))}（源指令语境，需单独判断适用性）",
        '- CWE：' + (', '.join(record['cwes']) or '未提供'),
    ])
    content = '\n\n'.join([
        '### 已知被利用目录事实\n' + facts,
        '### 尚未分析适用条件\n本条仅为 CISA KEV 公开目录采集素材，待人工审核且默认停用。目录收录说明存在已知利用证据，不代表当前项目或所有同类产品受影响。',
        '### 需人工补充\n核对具体产品、组件及版本/回补、部署配置、攻击前提和入口可达性；补充适用与不适用依据、防护措施、验证方法与结果。未完成这些内容前不得用来自动判定威胁、风险或合规。',
        '### 来源附注（未访问第三方链接）\n' + (escaped(record.get('notes', '')) or '无'),
        '### 来源与版本\n' + '\n'.join([
            f'- 官方目录：{CATALOG_URL}', f'- 固定数据源：{KEV_URL}',
            f"- catalogVersion：{catalog['catalogVersion']}",
            f'- 规范化记录 SHA-256：{digest}',
            f'- 数据许可：CISA KEV，CC0 1.0 Universal；{LICENSE_URL}',
            '- CC0 不覆盖第三方链接内容，不授权使用 CISA 标志，也不表示 CISA 背书。',
        ]),
    ])
    if len(content) >= 120000:
        raise ValidationError('Rendered candidate content exceeds the import limit')
    return {
        'id': 'cisa-kev.' + cve.lower(),
        'name': (f'{cve} · {record["vulnerabilityName"]}')[:200],
        'source': CATALOG_URL, 'edition': catalog['catalogVersion'],
        'tags': ['CISA KEV', '公开素材', '待人工分析'], 'content': content,
        'evidence_urls': [CATALOG_URL, KEV_URL, LICENSE_URL], 'cve_ids': [cve],
    }


def collect(feed, state, limit=25, captured_at=None):
    if type(limit) is not int or not 1 <= limit <= 50:
        raise ValidationError('limit must be between 1 and 50')
    catalog = validate_feed(feed)
    state = validate_state(state)
    captured_at = captured_at or dt.datetime.now(dt.timezone.utc).isoformat(timespec='seconds').replace('+00:00', 'Z')
    valid_timestamp(captured_at, 'captured_at')
    old = state['records']
    next_records = {key: dict(value) for key, value in old.items()}
    entries, unchanged, pending = [], 0, 0
    for cve, record in sorted(catalog['records'].items()):
        digest = content_hash(record)
        if cve in next_records:
            # Reappearance ends this absence episode; a later absence needs review again.
            next_records[cve].pop('missing_reported_hash', None)
        if old.get(cve, {}).get('hash') == digest:
            unchanged += 1
            next_records[cve]['last_seen_catalog'] = catalog['catalogVersion']
            continue
        if len(entries) >= limit:
            pending += 1
            continue
        entry = make_entry(record, catalog, captured_at, digest)
        if len(encoded(entries + [entry])) > MAX_ENTRY_BATCH_BYTES:
            pending += 1
            continue
        entries.append(entry)
        next_records[cve] = {'hash': digest, 'last_output_catalog': catalog['catalogVersion'],
                             'last_seen_catalog': catalog['catalogVersion']}
    removed = sorted(set(old) - set(catalog['records']))
    review_required, pending_reviews = [], 0
    for cve in removed:
        if old[cve].get('missing_reported_hash') == old[cve]['hash']:
            continue
        notice = {'id': 'cisa-kev.' + cve.lower(), 'cve_id': cve,
                  'reason': 'missing_from_snapshot', 'previous_hash': old[cve]['hash'],
                  'catalogVersion': catalog['catalogVersion'],
                  'action': '来源快照未包含旧条目：需人工核对是否撤回；保留原知识和历史引用，不自动删除。'}
        if len(review_required) >= 100 or len(encoded(review_required + [notice])) > 60 * 1024:
            pending_reviews += 1
            continue
        review_required.append(notice)
        # Only emitted notices advance this independent review queue. Original facts remain.
        next_records[cve]['missing_reported_hash'] = old[cve]['hash']
    metadata = {
        'source_url': KEV_URL, 'catalogVersion': catalog['catalogVersion'],
        'dateReleased': catalog['dateReleased'], 'captured_at': captured_at,
        'license': {'name': 'CC0-1.0', 'url': LICENSE_URL,
                    'notice': 'CISA KEV data only; third-party linked material is not covered. No endorsement.'},
        'snapshot_sha256': content_hash(catalog),
        'review_status': 'pending', 'enabled': False,
        'counts': {'snapshot': len(catalog['records']), 'emitted': len(entries),
                   'unchanged': unchanged, 'pending': pending, 'review_required': len(review_required),
                   'pending_reviews': pending_reviews, 'missing_total': len(removed)},
    }
    pack = {'schema': 'jieheng.knowledge-pack/1', 'pack_id': 'cisa-kev-candidates',
            'version': catalog['catalogVersion'], 'metadata': metadata,
            'entries': entries, 'review_required': review_required}
    if len(encoded(pack)) > MAX_PACK_BYTES:
        raise ValidationError('Candidate pack exceeds the HTTP import budget')
    next_state = {'schema': STATE_SCHEMA, 'source_url': KEV_URL,
                  'last_observed_catalog': catalog['catalogVersion'],
                  'captured_at': captured_at, 'records': next_records}
    return pack, next_state


def encoded(value):
    return (json.dumps(value, ensure_ascii=False, indent=2) + '\n').encode('utf-8')


def prepare_file(path, payload):
    path.parent.mkdir(parents=True, exist_ok=True)
    descriptor, name = tempfile.mkstemp(prefix='.' + path.name + '.', dir=path.parent)
    temporary = Path(name)
    try:
        with os.fdopen(descriptor, 'wb') as stream:
            stream.write(payload)
            stream.flush()
            os.fsync(stream.fileno())
    except BaseException:
        temporary.unlink(missing_ok=True)
        raise
    return temporary


def persist(output_path, state_path, pack, state):
    # Write output first, then checkpoint. A failed checkpoint may replay a batch,
    # but cannot skip records that were never durably emitted.
    if output_path.exists():
        raise ValidationError('Output already exists; choose a new batch filename to preserve candidates')
    output_tmp = state_tmp = None
    try:
        output_tmp = prepare_file(output_path, encoded(pack))
        state_tmp = prepare_file(state_path, encoded(state))
        os.link(output_tmp, output_path)  # Atomic creation, without overwriting a prior batch.
        os.replace(state_tmp, state_path)
    finally:
        if output_tmp is not None:
            output_tmp.unlink(missing_ok=True)
        if state_tmp is not None:
            state_tmp.unlink(missing_ok=True)


def main(argv=None):
    parser = argparse.ArgumentParser(description='Collect review-only CISA KEV candidates, without importing them into a project.')
    source = parser.add_mutually_exclusive_group(required=True)
    source.add_argument('--input', type=Path, help='Offline official-format JSON snapshot (maximum 5 MB)')
    source.add_argument('--fetch-kev', action='store_true', help='Explicitly fetch the fixed official CISA HTTPS feed; no redirects')
    parser.add_argument('--output', required=True, type=Path, help='New knowledge-pack filename; existing output is never overwritten')
    parser.add_argument('--state', required=True, type=Path, help='Local per-CVE cursor JSON')
    parser.add_argument('--limit', type=int, default=25, help='Maximum new/modified candidates (1-50; default 25)')
    args = parser.parse_args(argv)
    try:
        output_path, state_path = args.output.resolve(), args.state.resolve()
        input_path = args.input.resolve() if args.input else None
        if output_path == state_path or input_path in (output_path, state_path):
            raise ValidationError('Input, output and state must be different files')
        if output_path.exists():
            raise ValidationError('Output already exists; choose a new batch filename')
        state = read_json(state_path) if state_path.exists() else {'schema': STATE_SCHEMA, 'records': {}}
        validate_state(state)
        if not 1 <= args.limit <= 50:
            raise ValidationError('limit must be between 1 and 50')
        feed = read_json(input_path) if input_path else fetch_kev()
        pack, next_state = collect(feed, state, args.limit)
        pack['metadata']['collection_mode'] = 'offline-input' if input_path else 'official-https-fetch'
        pack['metadata']['origin_note'] = ('Offline input is caller-supplied; its origin has not been independently authenticated.'
                                            if input_path else 'Fetched from the fixed official HTTPS endpoint without redirects.')
        persist(output_path, state_path, pack, next_state)
        print(json.dumps({'output': str(args.output), **pack['metadata']['counts']}, ensure_ascii=False))
        return 0
    except (ValidationError, OSError, ValueError) as exc:
        print(f'Collection failed: {exc}', file=sys.stderr)
        return 1


if __name__ == '__main__':
    raise SystemExit(main())
