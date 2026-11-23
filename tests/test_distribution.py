import contextlib,hashlib,importlib.util,json,pathlib,tempfile,unittest
from unittest import mock
spec=importlib.util.spec_from_file_location('distribution_pipeline',pathlib.Path(__file__).resolve().parents[1]/'scripts/ztarc.py')
pipeline=importlib.util.module_from_spec(spec);spec.loader.exec_module(pipeline)
class DistributionTests(unittest.TestCase):
 @contextlib.contextmanager
 def fixture(self):
  with tempfile.TemporaryDirectory() as temp:
   d=pathlib.Path(temp);meta=pipeline.META
   def artifact(name,content):
    (d/name).write_bytes(content);return {'file':name,'sha256':hashlib.sha256(content).hexdigest()}
   binaries={}
   for arch in ('amd64','arm64'):
    name=f'ztarc-cli_{meta["version"]}_linux_{arch}';b=artifact(name,b'fixture '+arch.encode());b.update(size=(d/name).stat().st_size,url=f'https://dl.ztarc.io/cli/{meta["version"]}/{name}');binaries[arch]=b
   m={'schemaVersion':1,'version':meta['version'],'upstream':meta['upstream'],'engines':meta['engines'],'binaries':binaries,'installer':artifact('install.sh',b'fixture installer'),'provenance':artifact('provenance.json',b'{}'),'licenses':artifact('licenses.tar.gz',b'fixture licenses'),'image':None,'platforms':[],'compatibility':{},'capabilities':[]}
   def seal():
    (d/'manifest.json').write_text(json.dumps(m));(d/'release-id').write_text(pipeline.digest(d/'manifest.json')+'\n')
    (d/'SHA256SUMS').write_text(''.join(f'{pipeline.digest(p)}  {p.name}\n' for p in sorted(d.iterdir()) if p.name!='SHA256SUMS'))
   seal()
   with mock.patch.object(pipeline,'DIST',d):yield d,m,seal
 def test_matching_artifacts_verify(self):
  with self.fixture():pipeline.verify()
 def test_manifest_installer_mismatch_rejected(self):
  with self.fixture() as (d,m,seal):
   (d/'install.sh').write_bytes(b'altered installer')
   with self.assertRaises(ValueError):pipeline.verify()
 def test_manifest_identity_mismatch_rejected(self):
  with self.fixture() as (d,m,seal):
   (d/'release-id').write_text('0'*64)
   with self.assertRaises(ValueError):pipeline.verify()
 def test_latest_url_or_unsupported_evidence_claim_rejected(self):
  with self.fixture() as (d,m,seal):
   m['binaries']['amd64']['url']='https://dl.ztarc.io/cli/latest/binary';seal()
   with self.assertRaises(ValueError):pipeline.verify()
  with self.fixture() as (d,m,seal):
   m['platforms']=[{'status':'supported','evidence':[]}];seal()
   with self.assertRaises(ValueError):pipeline.verify()
 def test_requested_directory_used_by_release_script(self):
  # A fake cosign isolates directory routing; cryptographic verification is separate.
  import os,subprocess
  with self.fixture() as (d,m,seal), tempfile.TemporaryDirectory() as temp:
   tools=pathlib.Path(temp);cosign=tools/'cosign'
   cosign.write_text('#!/bin/sh\nexit 0\n');cosign.chmod(0o755)
   env=os.environ.copy();env.update(PATH=str(tools)+os.pathsep+env['PATH'],ZTARC_CERTIFICATE_IDENTITY='test-workflow')
   script=pathlib.Path(__file__).resolve().parents[1]/'scripts/verify-release.sh'
   def verify():return subprocess.run(['bash',str(script),str(d)],cwd=temp,env=env,capture_output=True,text=True)
   result=verify();self.assertEqual(result.returncode,0,result.stderr)
   (d/m['binaries']['amd64']['file']).write_bytes(b'tampered')
   result=verify();self.assertNotEqual(result.returncode,0);self.assertIn('artifact mismatch',result.stderr)
 def test_explicit_directory_does_not_require_default_dist(self):
  with self.fixture() as (d,m,seal), mock.patch.object(pipeline,'DIST',d/'absent'):
   pipeline.verify(d)
 def test_unborn_checkout_allows_development_but_not_release(self):
  import subprocess
  with tempfile.TemporaryDirectory() as temp:
   root=pathlib.Path(temp)
   subprocess.run(['git','init',str(root)],check=True,capture_output=True)
   with mock.patch.object(pipeline,'ROOT',root):
    self.assertEqual(pipeline.identity(),{'commit':'uncommitted','dirty':True})
    with self.assertRaisesRegex(ValueError,'committed ZTARC HEAD'):pipeline.identity(required=True)
