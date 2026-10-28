"""Offline tests for the public KEV collector; all fixtures are synthetic."""
import copy
import importlib.util
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock
from contextlib import redirect_stdout, redirect_stderr

spec = importlib.util.spec_from_file_location('collect_public_knowledge', Path(__file__).with_name('collect_public_knowledge.py'))
collector = importlib.util.module_from_spec(spec)
spec.loader.exec_module(collector)


def record(number=1):
    return {
        'cveID': f'CVE-2024-{number:04}', 'vendorProject': 'Synthetic Vendor',
        'product': 'Synthetic Product', 'vulnerabilityName': 'Synthetic boundary issue',
        'dateAdded': '2026-10-01', 'shortDescription': 'Synthetic public catalog fact.',
        'requiredAction': 'Review the synthetic vendor update.', 'dueDate': '2026-10-31',
        'knownRansomwareCampaignUse': 'Unknown', 'forensicTriage': 'No',
        'notes': 'https://example.test/advisory', 'cwes': ['CWE-20'],
    }


def snapshot(*records):
    return {'catalogVersion': '2026.10.28', 'dateReleased': '2026-10-28T00:00:00Z',
            'count': len(records), 'vulnerabilities': list(records)}


def empty_state():
    return {'schema': collector.STATE_SCHEMA, 'records': {}}


class CollectorTests(unittest.TestCase):
    def collect(self, feed, state=None, limit=25):
        return collector.collect(feed, state or empty_state(), limit, '2026-10-28T01:00:00Z')

    def test_repeat_emits_nothing_and_retains_stable_id(self):
        feed = snapshot(record())
        first, state = self.collect(feed)
        second, later = self.collect(feed, state)
        self.assertEqual(first['entries'][0]['id'], 'cisa-kev.cve-2024-0001')
        self.assertEqual(second['entries'], [])
        self.assertEqual(second['metadata']['counts']['unchanged'], 1)
        self.assertEqual(state['records'], later['records'])
        self.assertFalse(first['metadata']['enabled'])
        self.assertEqual(first['metadata']['review_status'], 'pending')
        self.assertIn('尚未分析适用条件', first['entries'][0]['content'])

    def test_changed_fact_emits_same_id_and_new_hash(self):
        first, state = self.collect(snapshot(record()))
        updated = record()
        updated['requiredAction'] = 'A revised synthetic recommendation.'
        second, next_state = self.collect(snapshot(updated), state)
        self.assertEqual(first['entries'][0]['id'], second['entries'][0]['id'])
        self.assertNotEqual(state['records']['CVE-2024-0001']['hash'], next_state['records']['CVE-2024-0001']['hash'])

    def test_limit_does_not_advance_unemitted_records(self):
        feed = snapshot(record(3), record(1), record(2))
        first, state = self.collect(feed, limit=1)
        self.assertEqual(len(state['records']), 1)
        self.assertEqual(first['metadata']['counts']['pending'], 2)
        second, state = self.collect(feed, state, limit=1)
        third, state = self.collect(feed, state, limit=1)
        emitted = first['entries'] + second['entries'] + third['entries']
        self.assertEqual(len({entry['id'] for entry in emitted}), 3)
        self.assertEqual(len(state['records']), 3)
        self.assertEqual(third['metadata']['counts']['pending'], 0)
        fourth, _ = self.collect(feed, state)
        self.assertEqual(fourth['entries'], [])

    def test_changed_record_beyond_limit_is_not_lost(self):
        _, state = self.collect(snapshot(record(2), record(3)))
        updated = record(3)
        updated['shortDescription'] = 'Revised synthetic fact.'
        first, intermediate = self.collect(snapshot(record(1), record(2), updated), state, limit=1)
        self.assertEqual(intermediate['records']['CVE-2024-0003']['hash'], state['records']['CVE-2024-0003']['hash'])
        second, completed = self.collect(snapshot(record(1), record(2), updated), intermediate, limit=1)
        self.assertEqual(second['entries'][0]['cve_ids'], ['CVE-2024-0003'])
        self.assertNotEqual(completed['records']['CVE-2024-0003']['hash'], state['records']['CVE-2024-0003']['hash'])

    def test_missing_record_requires_review_without_deleting_cursor(self):
        _, state = self.collect(snapshot(record(1), record(2)))
        pack, next_state = self.collect(snapshot(record(2)), state)
        self.assertEqual(pack['entries'], [])
        self.assertEqual(pack['review_required'][0]['cve_id'], 'CVE-2024-0001')
        self.assertEqual(pack['review_required'][0]['reason'], 'missing_from_snapshot')
        self.assertIn('CVE-2024-0001', next_state['records'])
        with self.assertRaises(collector.ValidationError):
            self.collect(snapshot(), state)

    def test_missing_notice_queue_drains_without_deleting_or_repeating(self):
        state = empty_state()
        original = snapshot(*(record(i) for i in range(1, 152)))
        for _ in range(4):
            _, state = self.collect(original, state, limit=50)
        first, state = self.collect(snapshot(record(151)), state)
        self.assertEqual(first['entries'], [])
        self.assertEqual(len(first['review_required']), 100)
        self.assertEqual(first['metadata']['counts']['pending_reviews'], 50)
        self.assertLessEqual(len(collector.encoded(first['review_required'])), 60 * 1024)
        second, state = self.collect(snapshot(record(151)), state)
        self.assertEqual(len(second['review_required']), 50)
        self.assertEqual(second['metadata']['counts']['pending_reviews'], 0)
        all_ids = [n['cve_id'] for n in first['review_required'] + second['review_required']]
        self.assertEqual(len(set(all_ids)), 150)
        self.assertEqual(len(state['records']), 151)
        third, state = self.collect(snapshot(record(151)), state)
        self.assertEqual(third['review_required'], [])
        _, state = self.collect(snapshot(record(1), record(151)), state)
        fourth, _ = self.collect(snapshot(record(151)), state)
        self.assertEqual([n['cve_id'] for n in fourth['review_required']], ['CVE-2024-0001'])

    def test_import_contract_limits_and_display_name(self):
        item = record()
        item['vulnerabilityName'] = 'x' * 512
        pack, _ = self.collect(snapshot(item), limit=50)
        self.assertLessEqual(len(pack['entries'][0]['name']), 200)
        self.assertIn('x' * 512, pack['entries'][0]['content'])
        for limit in (0, 51, 1000):
            with self.assertRaises(collector.ValidationError):
                self.collect(snapshot(record()), limit=limit)

    def test_capture_metadata_does_not_change_entry_fingerprint(self):
        feed = snapshot(record())
        first, _ = collector.collect(feed, empty_state(), 25, '2026-10-28T01:00:00Z')
        later, _ = collector.collect(feed, empty_state(), 25, '2026-10-28T02:00:00Z')
        self.assertNotEqual(first['metadata']['captured_at'], later['metadata']['captured_at'])
        self.assertEqual(first['entries'], later['entries'])
        changed_metadata = copy.deepcopy(feed)
        changed_metadata['dateReleased'] = '2026-10-28T00:01:00Z'
        replay, _ = self.collect(changed_metadata)
        self.assertEqual(first['entries'], replay['entries'])

    def test_rendered_content_length_is_checked_before_cursor_write(self):
        item = record()
        for field in ('shortDescription', 'requiredAction', 'notes'):
            item[field] = '&' * 12000
        with self.assertRaises(collector.ValidationError):
            self.collect(snapshot(item))
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source, state_path, output = root/'input.json', root/'state.json', root/'output.json'
            source.write_text(json.dumps(snapshot(item)))
            original = json.dumps(empty_state()).encode()
            state_path.write_bytes(original)
            with redirect_stderr(io.StringIO()):
                result = collector.main(['--input', str(source), '--output', str(output), '--state', str(state_path)])
            self.assertEqual(result, 1)
            self.assertEqual(state_path.read_bytes(), original)
            self.assertFalse(output.exists())

    def test_large_entries_are_byte_budgeted_without_losing_pending_records(self):
        records = []
        for i in range(1, 21):
            item = record(i)
            item['shortDescription'] = '界' * 12000
            item['requiredAction'] = '衡' * 12000
            item['notes'] = '文' * 11000
            records.append(item)
        feed = snapshot(*records)
        self.assertLess(len(collector.encoded(feed)), collector.MAX_BYTES)
        first, state = self.collect(feed, limit=50)
        self.assertGreater(first['metadata']['counts']['pending'], 0)
        self.assertLess(len(collector.encoded(first)), 1900000)
        self.assertEqual(len(state['records']), len(first['entries']))
        second, state = self.collect(feed, state, limit=50)
        self.assertEqual(len(first['entries']) + len(second['entries']), 20)
        self.assertEqual(len(state['records']), 20)
        self.assertEqual(second['metadata']['counts']['pending'], 0)

    def test_source_fractional_seconds_are_validated_without_reformatting(self):
        for timestamp in ('2026-10-27T19:00:02.2307Z', '2026-10-27T19:00:02.2Z',
                          '2026-10-27T19:00:02.230700Z', '2026-10-27T19:00:02.230712345+08:00'):
            feed = snapshot(record())
            feed['dateReleased'] = timestamp
            pack, _ = self.collect(feed)
            self.assertEqual(pack['metadata']['dateReleased'], timestamp)
        for timestamp in ('2026-02-30T19:00:02.2307Z', '2026-10-27T24:00:02.2307Z',
                          '2026-10-27T19:00:02.2307+24:00', '2026-10-27T19:00:02.2307+08:60'):
            with self.assertRaises(collector.ValidationError):
                collector.valid_timestamp(timestamp, 'dateReleased')

    def test_normalized_keys_cwe_order_and_whitespace_do_not_duplicate(self):
        original = record()
        original['cwes'] = ['CWE-20', 'CWE-190']
        _, state = self.collect(snapshot(original))
        equivalent = dict(reversed(list(original.items())))
        equivalent['cwes'] = ['CWE-190', 'CWE-20', 'CWE-20']
        equivalent['vendorProject'] = '  Synthetic Vendor  '
        pack, _ = self.collect(snapshot(equivalent), state)
        self.assertEqual(pack['entries'], [])

    def test_bad_count_cve_fields_dates_and_urls_rejected(self):
        bad_feeds = []
        bad = snapshot(record()); bad['count'] = 2; bad_feeds.append(bad)
        bad = snapshot(record()); bad['count'] = True; bad_feeds.append(bad)
        for key, value in (
            ('cveID', 'CVE-24-1'), ('product', 'x' * 513), ('product', 'x\x00y'),
            ('dateAdded', '2026-02-31'),
            ('notes', 'https://user:password@example.test'),
            ('notes', 'https://example.test:99999/advisory'),
            ('notes', 'https://example.test/' + 'x' * 2050),
            ('knownRansomwareCampaignUse', 'No'), ('forensicTriage', 'Maybe'),
        ):
            bad = snapshot(record()); bad['vulnerabilities'][0][key] = value; bad_feeds.append(bad)
        bad = snapshot(record()); bad['vulnerabilities'][0]['unreviewedNewField'] = 'value'; bad_feeds.append(bad)
        bad_feeds.append(snapshot(record(), record()))
        for feed in bad_feeds:
            with self.subTest(feed=feed):
                with self.assertRaises(collector.ValidationError):
                    self.collect(feed)

    def test_oversize_json_duplicate_keys_and_nan_rejected(self):
        for data in (b' ' * (collector.MAX_BYTES + 1), b'{"count":1,"count":2}', b'{"x":NaN}'):
            with self.assertRaises(collector.ValidationError):
                collector.decode_json(data)

    def test_framework_file_keyword_is_not_mistaken_for_a_file_uri(self):
        item = record()
        item['shortDescription'] = 'A framework operation such as `render file:` requires input checks.'
        pack, _ = self.collect(snapshot(item))
        self.assertIn('render file:', pack['entries'][0]['content'])
        for uri in ('file:///synthetic/path',):
            item['notes'] = uri
            with self.assertRaises(collector.ValidationError):
                self.collect(snapshot(item))

    def test_source_prose_with_colon_fragments_stays_inert_text(self):
        item = record()
        item['shortDescription'] = 'data: a description; javascript:alert(1); file:/a/path; <script>example</script>'
        item['notes'] = 'The literal data:text/html fragment is source text, not an evidence URL.'
        pack, _ = self.collect(snapshot(item))
        content = pack['entries'][0]['content']
        self.assertIn('data: a description', content)
        self.assertNotIn('<script>', content)
        self.assertEqual(pack['entries'][0]['evidence_urls'], [collector.CATALOG_URL, collector.KEV_URL, collector.LICENSE_URL])

    def test_source_markup_is_inert_and_links_are_not_fetched(self):
        item = record()
        item['shortDescription'] = '<script>bad()</script> [click](https://example.test)'
        with mock.patch.object(collector, 'fetch_kev', side_effect=AssertionError('must remain offline')):
            pack, _ = self.collect(snapshot(item))
        content = pack['entries'][0]['content']
        self.assertNotIn('<script>', content)
        self.assertNotIn('[click](', content)
        self.assertNotIn('https://example.test', pack['entries'][0]['evidence_urls'])

    def test_redirects_are_disabled_and_fetch_is_fixed_bounded_json(self):
        self.assertIsNone(collector.NoRedirect().redirect_request(None, None, 302, '', {}, 'https://example.test'))
        body = json.dumps(snapshot(record())).encode()
        response = mock.MagicMock()
        response.__enter__.return_value = response
        response.geturl.return_value = collector.KEV_URL
        response.status = 200
        response.headers = {'Content-Type': 'application/json', 'Content-Length': str(len(body))}
        response.read.return_value = body
        opener = mock.Mock()
        opener.open.return_value = response
        with mock.patch.object(collector.urllib.request, 'build_opener', return_value=opener) as build:
            result = collector.fetch_kev()
        self.assertEqual(result['count'], 1)
        self.assertIsInstance(build.call_args.args[0], collector.NoRedirect)
        self.assertEqual(opener.open.call_args.args[0].full_url, collector.KEV_URL)
        response.read.assert_called_once_with(collector.MAX_BYTES + 1)
        response.geturl.return_value = 'https://example.test/changed'
        with mock.patch.object(collector.urllib.request, 'build_opener', return_value=opener):
            with self.assertRaises(collector.ValidationError):
                collector.fetch_kev()

    def test_response_length_encoding_and_type_rejected(self):
        for headers in (
            {'Content-Type': 'application/json', 'Content-Length': str(collector.MAX_BYTES + 1)},
            {'Content-Type': 'application/json', 'Content-Encoding': 'gzip'},
            {'Content-Type': 'text/html'},
        ):
            response = mock.MagicMock()
            response.__enter__.return_value = response
            response.geturl.return_value = collector.KEV_URL
            response.status = 200
            response.headers = headers
            opener = mock.Mock(); opener.open.return_value = response
            with mock.patch.object(collector.urllib.request, 'build_opener', return_value=opener):
                with self.assertRaises(collector.ValidationError):
                    collector.fetch_kev()
            response.read.assert_not_called()

    def test_cli_invalid_input_keeps_output_and_state_untouched(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            input_path, output_path, state_path = root/'input.json', root/'output.json', root/'state.json'
            _, state = self.collect(snapshot(record()))
            original = json.dumps(state).encode()
            state_path.write_bytes(original)
            invalid = snapshot(record()); invalid['count'] = 0
            input_path.write_text(json.dumps(invalid))
            with redirect_stderr(io.StringIO()), redirect_stdout(io.StringIO()):
                result = collector.main(['--input', str(input_path), '--output', str(output_path), '--state', str(state_path)])
            self.assertEqual(result, 1)
            self.assertEqual(state_path.read_bytes(), original)
            self.assertFalse(output_path.exists())

    def test_cli_offline_batches_and_existing_output_protection(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            input_path, state_path = root/'input.json', root/'state.json'
            input_path.write_text(json.dumps(snapshot(record(1), record(2))))
            with mock.patch.object(collector, 'fetch_kev', side_effect=AssertionError('network is opt-in')):
                for name in ('first.json', 'second.json', 'third.json'):
                    with redirect_stdout(io.StringIO()):
                        result = collector.main(['--input', str(input_path), '--output', str(root/name), '--state', str(state_path), '--limit', '1'])
                    self.assertEqual(result, 0)
            self.assertEqual(len(json.loads((root/'third.json').read_text())['entries']), 0)
            saved = state_path.read_bytes()
            first = (root/'first.json').read_bytes()
            with redirect_stderr(io.StringIO()):
                result = collector.main(['--input', str(input_path), '--output', str(root/'first.json'), '--state', str(state_path)])
            self.assertEqual(result, 1)
            self.assertEqual(state_path.read_bytes(), saved)
            self.assertEqual((root/'first.json').read_bytes(), first)

    def test_failed_output_write_does_not_advance_cursor(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            state_path = root/'state.json'
            original = json.dumps(empty_state()).encode()
            state_path.write_bytes(original)
            pack, state = self.collect(snapshot(record()))
            with mock.patch.object(collector.os, 'link', side_effect=OSError('synthetic write failure')):
                with self.assertRaises(OSError):
                    collector.persist(root/'output.json', state_path, pack, state)
            self.assertEqual(state_path.read_bytes(), original)
            self.assertFalse((root/'output.json').exists())
            self.assertEqual(sorted(p.name for p in root.iterdir()), ['state.json'])


if __name__ == '__main__':
    unittest.main()
