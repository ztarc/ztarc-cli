#!/usr/bin/env python3
"""Shared local/CI pipeline. Never modifies upstream/."""
import argparse, hashlib, json, os, pathlib, shutil, subprocess, sys, tarfile, re, io
ROOT = pathlib.Path(__file__).resolve().parents[1]
SRC = ROOT / 'build/src'
DIST = ROOT / 'dist'
META = json.loads((ROOT / 'release/version.json').read_text())
def run(*args, cwd=ROOT, env=None, capture=False):
    return subprocess.run(args, cwd=cwd, env=env, check=True, text=True, stdout=subprocess.PIPE if capture else None).stdout

def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def pin():
    source_marker=ROOT/'upstream/SOURCE_COMMIT'
    if source_marker.is_file() and not (ROOT/'upstream/.git').exists():
        if source_marker.read_text().strip()!=META['upstream']['commit']: raise ValueError('source bundle commit mismatch')
        inventory=json.loads((ROOT/'release/upstream-files.json').read_text())
        actual_files={str(p.relative_to(ROOT/'upstream')):digest(p) for p in (ROOT/'upstream').rglob('*') if p.is_file() and p!=source_marker}
        if actual_files!=inventory: raise ValueError('source bundle upstream inventory mismatch')
        return
    actual = run('git','-C',str(ROOT/'upstream'),'rev-parse','HEAD',capture=True).strip()
    if actual != META['upstream']['commit']: raise ValueError('upstream commit mismatch')
    if run('git','-C',str(ROOT/'upstream'),'status','--porcelain','--untracked-files=all',capture=True).strip():
        raise ValueError('upstream must be clean, including untracked files')
    gomod = (ROOT/'upstream/go.mod').read_text()
    for required in ('go 1.26.0','toolchain '+META['toolchain'],'github.com/fosrl/newt v'+META['engines']['newt'],'github.com/fosrl/olm v'+META['engines']['olm']):
        if required not in gomod: raise ValueError('dependency/toolchain metadata mismatch: '+required)

def stage():
    pin()
    if SRC.exists(): shutil.rmtree(SRC)
    SRC.mkdir(parents=True)
    # Export tracked bytes only. Do not copy upstream Git metadata, caches or local files.
    if (ROOT/'upstream/SOURCE_COMMIT').is_file() and not (ROOT/'upstream/.git').exists():
        shutil.copytree(ROOT/'upstream',SRC,dirs_exist_ok=True)
        (SRC/'SOURCE_COMMIT').unlink()
    else:
        proc = subprocess.Popen(['git','-C',str(ROOT/'upstream'),'archive','HEAD'],stdout=subprocess.PIPE)
        with tarfile.open(fileobj=proc.stdout, mode='r|') as archive: archive.extractall(SRC, filter='data')
        proc.stdout.close()
        if proc.wait(): raise ValueError('upstream export failed')
    for patch in sorted((ROOT/'brand/patches').glob('*.patch')):
        run('git','apply','--no-index','--check',str(patch),cwd=SRC)
        run('git','apply','--no-index',str(patch),cwd=SRC)
    for rule in json.loads((ROOT/'brand/replacements/rules.json').read_text()):
        path = SRC / rule['path']; content = path.read_text()
        count = content.count(rule['from'])
        if count != rule['count']: raise ValueError(f"replacement drift: {rule['path']}: {count} != {rule['count']}")
        path.write_text(content.replace(rule['from'],rule['to']))
    for path in sorted((ROOT/'brand/overrides').rglob('*')):
        if path.is_file():
            dest=SRC/path.relative_to(ROOT/'brand/overrides'); dest.parent.mkdir(parents=True,exist_ok=True); shutil.copyfile(path,dest)
    goroot=run('go','env','GOROOT',env=goenv(),capture=True).strip()
    run(str(pathlib.Path(goroot)/'bin/gofmt'),'-w',*map(str,SRC.rglob('*.go')))
    run('go','mod','verify',cwd=SRC,env=goenv(),capture=True)
    run('go','mod','vendor',cwd=SRC,env=goenv())
    for rule in json.loads((ROOT/'release/dependency-transforms.json').read_text()):
        path=SRC/rule['path']; content=path.read_text()
        if content.count(rule['from'])!=rule['count']: raise ValueError('dependency patch drift: '+rule['path'])
        path.write_text(content.replace(rule['from'],rule['to']))
    audit()
    print('Staged',SRC)

def audit():
    spec=json.loads((ROOT/'brand/audit.json').read_text())
    for path, assertions in spec.items():
        content=(SRC/path).read_text()
        for expected in assertions.get('required',[]):
            if expected not in content: raise ValueError(f'audit missing {path}: {expected}')
        for forbidden in assertions.get('forbidden',[]):
            if forbidden in content: raise ValueError(f'audit forbidden {path}: {forbidden}')
    # Exceptions are exact path + literal, with a reason. No blanket vendor exemptions.
    exceptions=json.loads((ROOT/'brand/audit-exceptions.json').read_text())
    for path in SRC.rglob('*.go'):
        if 'vendor' in path.relative_to(SRC).parts: continue
        if path.name.endswith('_test.go'): continue
        for n,line in enumerate(path.read_text().splitlines(),1):
            if line.lstrip().startswith('//'): continue
            literals=re.findall(r'"(?:[^"\\]|\\.)*"',line)
            if not any(any(x in literal.lower() for x in ('pangolin','fossorial')) for literal in literals): continue
            rel=str(path.relative_to(SRC))
            if not any(e['path']==rel and e['literal'] in line and e['reason'] for e in exceptions):
                raise ValueError(f'unreviewed upstream identifier: {rel}:{n}: {line.strip()}')
    for rule in json.loads((ROOT/'release/dependency-transforms.json').read_text()):
        content=(SRC/rule['path']).read_text()
        if rule['from'] in content or rule['to'] not in content: raise ValueError('dependency namespace audit failed')
    for pkg in ('vendor/github.com/fosrl/newt/newt','vendor/github.com/fosrl/olm/olm'):
        for path in (SRC/pkg).rglob('*.go'):
            if path.name.endswith('_test.go'): continue
            if any(x in path.read_text() for x in ('CheckAndSelfUpdate(', 'CheckForUpdate(', 'SelfUpdate(')): raise ValueError('unreviewed engine updater entry: '+str(path))
    print('Brand/protocol audit passed')

def goenv():
    env=os.environ.copy();env.update(GOTOOLCHAIN=META['toolchain'],GOCACHE=str(ROOT/'.cache/go-build'),GOMODCACHE=str(ROOT/'.cache/go-mod'))
    return env

def test():
    stage()
    run(sys.executable,'-m','unittest','discover','-s','tests','-v')
    run('go','test','./...',cwd=SRC,env=goenv())

def identity(required=False):
    rev=subprocess.run(['git','rev-parse','--show-toplevel'],cwd=ROOT,text=True,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
    if rev.returncode or pathlib.Path(rev.stdout.strip())!=ROOT:
        if required: raise ValueError('release requires a real ZTARC Git checkout')
        return {'commit':'uncommitted','dirty':True}
    head=subprocess.run(['git','rev-parse','--verify','HEAD'],cwd=ROOT,text=True,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
    if head.returncode:
        if required: raise ValueError('release requires a committed ZTARC HEAD')
        return {'commit':'uncommitted','dirty':True}
    commit=head.stdout.strip()
    dirty=bool(run('git','status','--porcelain','--untracked-files=all',capture=True).strip())
    if required and dirty: raise ValueError('release refuses a dirty source tree')
    return {'commit':commit,'dirty':dirty}

def build():
    stage();DIST.mkdir(exist_ok=True)
    ident=identity(); patch_inventory={str(p.relative_to(ROOT)):digest(p) for p in sorted((ROOT/'brand').rglob('*')) if p.is_file()}
    patch_inventory['release/dependency-transforms.json']=digest(ROOT/'release/dependency-transforms.json')
    provenance={'schemaVersion':1,'product':META,'ztarc':ident,'transformations':patch_inventory,'toolchain':META['toolchain']}
    for arch in ('amd64','arm64'):
        env=goenv();env.update(GOOS='linux',GOARCH=arch,CGO_ENABLED='0')
        dest=DIST/f'ztarc-cli_{META["version"]}_linux_{arch}'
        flags=f'-buildid= -X github.com/fosrl/cli/internal/version.Version={META["upstream"]["version"]} -X github.com/fosrl/cli/internal/version.NewtVersionOverride=1.20.1 -X github.com/fosrl/cli/internal/version.OlmVersionOverride=1.10.2 -X github.com/fosrl/cli/internal/version.ProductVersion={META["version"]} -X github.com/fosrl/cli/internal/version.ZtarcCommit={ident["commit"]}'
        run('go','build','-buildvcs=false','-trimpath','-ldflags',flags,'-o',str(dest),'.',cwd=SRC,env=env)
    (DIST/'provenance.json').write_text(json.dumps(provenance,indent=2)+'\n')

def source():
    pin(); DIST.mkdir(exist_ok=True)
    if not (SRC/'vendor').exists(): stage()
    audit()
    with tarfile.open(DIST/f'ztarc-cli_{META["version"]}_source.tar.gz','w:gz') as archive:
        for path in sorted(ROOT.iterdir()):
            if path.name in ('.git','.agents','.codex','.aws','.cache','build','dist','upstream','staged-source'): continue
            archive.add(path,arcname='ztarc-cli/'+path.name,filter=lambda i: None if '__pycache__' in i.name else i)
        for path in sorted((ROOT/'upstream').iterdir()):
            if path.name=='.git': continue
            archive.add(path,arcname='ztarc-cli/upstream/'+path.name)
        marker=tarfile.TarInfo('ztarc-cli/upstream/SOURCE_COMMIT'); data=(META['upstream']['commit']+'\n').encode();marker.size=len(data);archive.addfile(marker,io.BytesIO(data))
        archive.add(SRC,arcname='ztarc-cli/staged-source')

def package():
    build();source()
    licenses=DIST/f'ztarc-cli_{META["version"]}_licenses.tar.gz'
    with tarfile.open(licenses,'w:gz') as bundle:
        for name in ('LICENSE','NOTICE','THIRD_PARTY_NOTICES'): bundle.add(ROOT/name,arcname=name)
        bundle.add(SRC/'vendor/modules.txt',arcname='modules.txt')
        for path in sorted((SRC/'vendor').rglob('*')):
            if path.is_file() and path.name.upper().startswith(('LICENSE','COPYING','NOTICE','AUTHORS')):
                bundle.add(path,arcname=str(path.relative_to(SRC/'vendor')))
    binaries={}
    for arch in ('amd64','arm64'):
        p=DIST/f'ztarc-cli_{META["version"]}_linux_{arch}'
        binaries[arch]={'file':p.name,'sha256':digest(p),'size':p.stat().st_size,'url':f'https://dl.ztarc.io/cli/{META["version"]}/{p.name}'}
    template=(ROOT/'packaging/linux/install.sh.in').read_text()
    for arch,b in binaries.items(): template=template.replace('@'+arch.upper()+'_SHA256@',b['sha256'])
    template=template.replace('@VERSION@',META['version']).replace('@LICENSES_SHA256@',digest(licenses))
    installer=DIST/'install.sh';installer.write_text(template);installer.chmod(0o755)
    manifest={'schemaVersion':1,'version':META['version'],'upstream':META['upstream'],'engines':META['engines'],'binaries':binaries,'installer':{'file':'install.sh','sha256':digest(installer)},'licenses':{'file':licenses.name,'sha256':digest(licenses)},'image':None,'platforms':json.loads((ROOT/'release/platforms.json').read_text())['platforms'],'compatibility':json.loads((ROOT/'release/compatibility.json').read_text()),'provenance':{'file':'provenance.json','sha256':digest(DIST/'provenance.json')},'capabilities':json.loads((ROOT/'release/commands.json').read_text())}
    p=DIST/'manifest.json';p.write_text(json.dumps(manifest,sort_keys=True,indent=2)+'\n')
    (DIST/'release-id').write_text(digest(p)+'\n')
    for stale in DIST.glob('*.sigstore.json'): stale.unlink()
    files=sorted(p for p in DIST.iterdir() if p.is_file() and p.name not in ('SHA256SUMS',))
    (DIST/'SHA256SUMS').write_text(''.join(f'{digest(p)}  {p.name}\n' for p in files))
    verify()

def verify(dist=None):
    dist = pathlib.Path(dist) if dist is not None else DIST
    manifest=json.loads((dist/'manifest.json').read_text())
    if manifest.get('schemaVersion')!=1 or manifest.get('version')!=META['version'] or manifest.get('upstream')!=META['upstream']: raise ValueError('manifest version/upstream mismatch')
    if set(manifest.get('binaries',{}))!={'amd64','arm64'}: raise ValueError('incomplete platform artifacts')
    for arch,b in manifest['binaries'].items():
        expected=f'https://dl.ztarc.io/cli/{META["version"]}/ztarc-cli_{META["version"]}_linux_{arch}'
        if b.get('url')!=expected or not isinstance(b.get('size'),int) or b['size']<=0: raise ValueError('invalid immutable binary metadata')
    for platform in manifest['platforms']:
        if platform['status']=='supported' and not platform['evidence']: raise ValueError('supported platform needs evidence')
    if digest(dist/'manifest.json')!=(dist/'release-id').read_text().strip(): raise ValueError('releaseId mismatch')
    for b in [*manifest['binaries'].values(),manifest['installer'],manifest['provenance'],manifest['licenses']]:
        p=dist/b['file']
        if pathlib.Path(b['file']).name!=b['file']: raise ValueError('unsafe artifact filename')
        if digest(p)!=b['sha256'] or ('size' in b and p.stat().st_size!=b['size']): raise ValueError('artifact mismatch: '+p.name)
    image=manifest.get('image')
    if image:
        p=dist/image['archive']
        if pathlib.Path(image['archive']).name!=image['archive'] or digest(p)!=image['sha256']: raise ValueError('image archive mismatch')
    for line in (dist/'SHA256SUMS').read_text().splitlines():
        sha,name=line.split('  ',1)
        if pathlib.Path(name).name!=name or digest(dist/name)!=sha: raise ValueError('checksum mismatch')
    print('Artifact integrity verified (signature verification is a separate mandatory release step)')

def main():
    p=argparse.ArgumentParser();p.add_argument('command',choices=['stage','audit','test','build','package','source','verify','clean','release-check']);p.add_argument('--dist',type=pathlib.Path,help='distribution directory for verify');args=p.parse_args()
    if args.dist is not None and args.command!='verify': p.error('--dist is only valid with verify')
    if args.command=='clean':
        for path in (ROOT/'build',DIST):
            if path.exists(): shutil.rmtree(path)
    elif args.command=='verify': verify(args.dist)
    elif args.command=='release-check': pin();identity(required=True)
    else: globals()[args.command]()
if __name__=='__main__':
    try: main()
    except (ValueError,subprocess.CalledProcessError) as e: sys.exit(str(e))
