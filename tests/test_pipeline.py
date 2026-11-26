import hashlib,importlib.util,json,pathlib,subprocess,sys,tempfile,unittest
from unittest import mock
ROOT=pathlib.Path(__file__).resolve().parents[1]
def module(name,path):
 spec=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
pipeline=module('pipeline',ROOT/'scripts/ztarc.py')
migration=module('migration',ROOT/'scripts/migrate-systemd.py')
class PipelineTests(unittest.TestCase):
 def test_machine_registration_transform_is_in_staged_source(self):
  source=(pipeline.SRC/'cmd/up/client/client.go').read_text()
  self.assertIn('OrgID:                             registrationOrg(orgID, userToken),',source)
 def test_source_pin_and_stage_determinism(self):
  pipeline.pin()
  def tree():return {str(p.relative_to(pipeline.SRC)):hashlib.sha256(p.read_bytes()).hexdigest() for p in pipeline.SRC.rglob('*') if p.is_file()}
  before=tree();pipeline.stage();self.assertEqual(before,tree())
  self.assertEqual(subprocess.check_output(['git','-C',str(ROOT/'upstream'),'status','--porcelain']).strip(),b'')
 def test_audit_detects_update_and_brand_drift(self):
  p=pipeline.SRC/'cmd/up/site/site.go';original=p.read_text()
  try:
   p.write_text(original+'\n// CheckAndSelfUpdate must never return\n')
   with self.assertRaises(ValueError):pipeline.audit()
  finally:p.write_text(original)
 def test_release_requires_git_identity(self):
  fake=mock.Mock(returncode=128,stdout='')
  with mock.patch.object(pipeline.subprocess,'run',return_value=fake):
   with self.assertRaises(ValueError):pipeline.identity(required=True)
 def test_command_inventory_statuses(self):
  entries=json.loads((ROOT/'release/commands.json').read_text())
  for command in ('ztarc-cli ssh','ztarc-cli apply blueprint','ztarc-cli configure','ztarc-cli companion'):
   self.assertEqual(next(x['status'] for x in entries if x['command']==command),'deferred')
class MigrationTests(unittest.TestCase):
 def test_preserves_operational_unit_and_identity(self):
  src=b'[Unit]\nAfter=network-online.target\n[Service]\nExecStart=/usr/local/bin/ztarc-connector --config-file /etc/ztarc/connector.json\nRestart=always\n'
  out,old=migration.replacement(src,pathlib.Path('/usr/local/bin/ztarc-cli'))
  self.assertIn('Restart=always',out);self.assertIn('up site --config-file /etc/ztarc/connector.json',out)
  self.assertTrue(out.startswith(migration.MARKER));self.assertEqual(old,'/usr/local/bin/ztarc-connector')
 def test_refuses_nonstandard_flags_or_hooks(self):
  for unit in (b'ExecStart=/bin/newt --secret redacted',b'ExecStart=/bin/newt --config-file /etc/ztarc/connector.json\nExecStop=/bin/true'):
   with self.assertRaises(ValueError):migration.replacement(unit,pathlib.Path('/usr/local/bin/ztarc-cli'))
 def test_stop_requires_no_main_process(self):
  responses=[mock.Mock(stdout=''),mock.Mock(stdout='inactive'),mock.Mock(stdout='123')]
  with mock.patch.object(migration,'run',side_effect=responses):
   with self.assertRaises(ValueError):migration.stopped()
 def test_journal_atomic_and_private(self):
  with tempfile.TemporaryDirectory() as d:
   p=pathlib.Path(d)/'state.json';migration.atomic(p,b'{"phase":"prepared"}');self.assertEqual(p.stat().st_mode&0o777,0o600)
   migration.atomic(p,b'{"phase":"stopped"}');self.assertEqual(json.loads(p.read_text())['phase'],'stopped')
if __name__=='__main__':unittest.main()
