"""Opt-in installer integration test; all root filesystem changes stay in Docker."""
import hashlib
import io
import os
import pathlib
import subprocess
import tarfile
import tempfile
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]


@unittest.skipUnless(os.environ.get('ZTARC_INSTALLER_CONTAINER_TESTS') == '1',
                     'set ZTARC_INSTALLER_CONTAINER_TESTS=1 to run isolated Docker tests')
class InstallerTests(unittest.TestCase):
    def test_fresh_destinations_are_executable_by_ordinary_users(self):
        with tempfile.TemporaryDirectory() as temp:
            fixtures = pathlib.Path(temp)
            binary = b'#!/bin/sh\nprintf "fixture-cli-ok\\n"\n'
            (fixtures / 'binary').write_bytes(binary)
            with tarfile.open(fixtures / 'licenses.tar.gz', 'w:gz') as archive:
                notice = tarfile.TarInfo('NOTICE')
                notice.mode = 0o644
                content = b'test license notice\n'
                notice.size = len(content)
                archive.addfile(notice, io.BytesIO(content))
            template = (ROOT / 'packaging/linux/install.sh.in').read_text()
            for key, value in {
                '@VERSION@': 'test.1',
                '@AMD64_SHA256@': hashlib.sha256(binary).hexdigest(),
                '@ARM64_SHA256@': hashlib.sha256(binary).hexdigest(),
                '@LICENSES_SHA256@': hashlib.sha256((fixtures / 'licenses.tar.gz').read_bytes()).hexdigest(),
            }.items():
                template = template.replace(key, value)
            (fixtures / 'install.sh').write_text(template)
            result = subprocess.run([
                'docker', 'run', '--rm', '--pull=never', '--network', 'none',
                '--mount', f'type=bind,src={fixtures},dst=/fixtures,readonly',
                '--mount', f'type=bind,src={ROOT / "tests/integration/install-container.sh"},dst=/check.sh,readonly',
                '--entrypoint', '/bin/sh', os.environ.get('ZTARC_INSTALLER_TEST_IMAGE', 'ubuntu:24.04@sha256:2198e4c65076e9cb0b3d695219df667c0e3b0072da64ca3c2648aca17dcecaf5'),
                '/check.sh',
            ], capture_output=True, text=True, timeout=60)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
