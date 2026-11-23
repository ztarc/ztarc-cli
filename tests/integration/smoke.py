#!/usr/bin/env python3
"""Offline smoke checks. Never starts a tunnel, enrolls or changes host services."""
import json,os,pathlib,subprocess,tempfile
ROOT=pathlib.Path(__file__).resolve().parents[2]
version=json.loads((ROOT/'release/version.json').read_text())['version']
binary=ROOT/'dist'/f'ztarc-cli_{version}_linux_amd64'
with tempfile.TemporaryDirectory() as temp:
 env=os.environ.copy();env.update(HOME=temp,SUDO_USER='',SUDO_UID='')
 def run(*args,ok=True):
  p=subprocess.run([str(binary),*args],env=env,text=True,capture_output=True,timeout=10)
  assert (p.returncode==0)==ok,(args,p.stdout,p.stderr)
  return p
 p=run('version','--json');data=json.loads(p.stdout);assert data['version']==version
 assert not list(pathlib.Path(temp).iterdir()),'version wrote user state'
 assert 'Pangolin desktop' not in p.stdout+p.stderr
 assert 'disabled' in run('update').stdout.lower()
 assert 'deferred' in run('ssh','test.invalid',ok=False).stderr
 assert 'deferred' in run('up','site','--blueprint-file=/nonexistent',ok=False).stderr
 d=pathlib.Path(temp)/'private';d.mkdir(mode=0o700)
 credentials=d/'client.json';credentials.write_text('{"id":"i","secret":"do-not-print","endpoint":"https://invalid.test","org":"o"}');credentials.chmod(0o644)
 p=run('up','client','--credentials-file',str(credentials),ok=False)
 assert 'do-not-print' not in p.stdout+p.stderr
 assert 'owner-only' in p.stderr
 credentials.chmod(0o600)
 p=run('up','client','--credentials-file',str(credentials),'--secret','x',ok=False)
 assert 'conflicts' in p.stderr
 site=d/'site.json';site.write_text('{"id":"site","secret":"do-not-print","endpoint":"https://invalid.test"}');site.chmod(0o600)
 run('config','validate-site',str(site))
 assert not credentials.is_symlink()
print('Offline version, updater, credential and capability smoke checks passed')
