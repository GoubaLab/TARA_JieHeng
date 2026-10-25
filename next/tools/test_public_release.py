"""Public release scope tests use original, synthetic inputs only."""
import importlib.util
import json
from pathlib import Path
import re
import tempfile
import unittest

spec=importlib.util.spec_from_file_location('public_release',Path(__file__).with_name('public_release.py'))
release=importlib.util.module_from_spec(spec);spec.loader.exec_module(release)


class SourceReleaseTests(unittest.TestCase):
    def test_private_data_binaries_and_fulltext_are_excluded(self):
        for name in ['next/data/regulations/manifest.json','next/data/regulations/sample.json','next/data/regulations/sources/sample.pdf','next/data/regulations/sources/sample.zip','next/bin/jieheng','next/web/assets/app.js','next/var/sample.sqlite3','next/assurance/workbook.mjs','next/tools/private-material-fingerprints.json','next/tools/package.py','next/server/internal/assurance/testdata/example.json']:
            self.assertFalse(release.source_allowed(name),name)
        for name in ['next/server/internal/assurance/example_test.go','next/tests/test_method_config.mjs','next/client/public/lane-router.js','next/tools/public_release.py','next/EXTENDING.md','next/data/regulations/build_gb44495.py']:
            self.assertTrue(release.source_allowed(name),name)

    def test_staged_builders_remove_machine_paths(self):
        text=release.transformed('next/data/regulations/build_gb44495.py',b"SOURCE = Path('/a/synthetic/private/input.pdf')\n")
        self.assertNotIn(b'/a/synthetic',text)
        self.assertIn(b'provide-your-authorized-input.pdf',text)

    def test_manifest_must_be_explicitly_empty(self):
        evidence={'file_sha256':[],'identifier_sha256':[],'normalized_text_windows':[]}
        with tempfile.TemporaryDirectory() as directory:
            stage=Path(directory);file=stage/'next/data/regulations/manifest.json';file.parent.mkdir(parents=True)
            file.write_text(json.dumps(release.EMPTY_MANIFEST));self.assertEqual(len(release.audit(stage,evidence,set())),1)
            file.write_text(json.dumps({'schema_version':1,'documents':[{'id':'synthetic'}]}))
            with self.assertRaises(ValueError):release.audit(stage,evidence,set())


    def test_local_paths_and_credentials_are_detected(self):
        for root in ('/Users','/home'):
            text='see '+root+'/someone/work/input.pdf'
            self.assertIsNotNone(re.search(release.LOCAL_OR_CREDENTIAL,text),text)
        token='token '+'ghp_'+'a'*30
        self.assertIsNotNone(re.search(release.LOCAL_OR_CREDENTIAL,token),token)
        for text in ('relative/path/only','see /opt/shared/work/input.pdf'):
            self.assertIsNone(re.search(release.LOCAL_OR_CREDENTIAL,text),text)

    def test_staging_tool_source_passes_its_own_audit(self):
        source=Path(release.__file__).read_text(encoding='utf-8')
        self.assertIsNone(re.search(release.LOCAL_OR_CREDENTIAL,source))


if __name__ == '__main__':unittest.main()
