#!/usr/bin/env python3
"""Build an OCI archive locally. No registry push or release promotion."""
import hashlib,json,pathlib,subprocess,tarfile,os,argparse
from ztarc import ROOT,DIST,META,digest,verify
parser=argparse.ArgumentParser(description=__doc__);parser.add_argument('--platform',default='linux/amd64,linux/arm64',choices=['linux/amd64','linux/arm64','linux/amd64,linux/arm64']);args=parser.parse_args()
base=json.loads((ROOT/'release/container.json').read_text())['baseImage']
if '@sha256:' not in base:raise SystemExit('container requires immutable base image')
archive=DIST/'ztarc-cli_image.tar'
subprocess.run(['docker','buildx','build','--platform',args.platform,'--provenance=false','--sbom=false','--build-arg','BASE_IMAGE='+base,'--build-arg','PRODUCT_VERSION='+META['version'],'--build-arg','SOURCE_REPOSITORY=https://github.com/'+os.environ.get('GITHUB_REPOSITORY','unconfigured'),'--output','type=oci,dest='+str(archive),'-f','packaging/docker/Dockerfile','.'],cwd=ROOT,check=True)
with tarfile.open(archive) as t:
 index=json.load(t.extractfile('index.json'))
 descriptor=index['manifests'][0]
 # Exporter wraps the multi-platform image index in the layout index.
 image_digest=descriptor['digest']
 blob=t.extractfile('blobs/sha256/'+image_digest.split(':')[1]).read()
 if 'sha256:'+hashlib.sha256(blob).hexdigest()!=image_digest:raise SystemExit('OCI digest mismatch')
manifest=json.loads((DIST/'manifest.json').read_text())
manifest['image']={'repository':'registry.ztarc.io/ztarc/cli','digest':image_digest,'archive':archive.name,'sha256':digest(archive),'size':archive.stat().st_size,'status':'not_tested','published':False,'baseImage':base,'platforms':args.platform.split(',')}
(DIST/'manifest.json').write_text(json.dumps(manifest,sort_keys=True,indent=2)+'\n')
(DIST/'release-id').write_text(digest(DIST/'manifest.json')+'\n')
# Container must be attached before signing; stale bundles are rejected/removed.
for p in DIST.glob('*.sigstore.json'):p.unlink()
files=sorted(p for p in DIST.iterdir() if p.is_file() and p.name!='SHA256SUMS')
(DIST/'SHA256SUMS').write_text(''.join(f'{digest(p)}  {p.name}\n' for p in files))
verify()
print('OCI archive built, not published:',image_digest)
