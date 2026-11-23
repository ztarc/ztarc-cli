#!/usr/bin/env python3
"""Explicit, journaled migration of one ztarc-connector systemd unit.
Never run automatically by binary installation. Credentials stay on the host.
"""
import argparse, fcntl, hashlib, json, os, pathlib, stat, subprocess, sys, tempfile
UNIT=pathlib.Path('/etc/systemd/system/ztarc-connector.service')
STATE=pathlib.Path('/var/lib/ztarc-cli/migration')
CONFIG=pathlib.Path('/etc/ztarc/connector.json')
MARKER='# Managed by ZTARC CLI v1\n'

def run(*args, check=True):
    # Output may include secrets in old unit status/argv. Never echo it.
    return subprocess.run(args,check=check,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True)

def read_regular(path, private=False, max_size=1024*1024):
    # Every ancestor is traversed without following symlinks.
    fd=os.open('/',os.O_RDONLY|os.O_DIRECTORY)
    try:
        for part in path.absolute().parts[1:-1]:
            nxt=os.open(part,os.O_RDONLY|os.O_DIRECTORY|os.O_NOFOLLOW,dir_fd=fd)
            os.close(fd);fd=nxt
        leaf=os.open(path.name,os.O_RDONLY|os.O_NOFOLLOW|os.O_NONBLOCK,dir_fd=fd)
        try:
            st=os.fstat(leaf)
            if not stat.S_ISREG(st.st_mode) or st.st_uid!=0 or st.st_mode&0o022 or st.st_nlink!=1:
                raise ValueError('unsafe root-owned file: '+str(path))
            if private and st.st_mode&0o077: raise ValueError('credential/backup must be root-only')
            with os.fdopen(os.dup(leaf),'rb') as f:
                data=f.read(max_size+1)
                if len(data)>max_size: raise ValueError('file exceeds size limit')
                return data
        finally:os.close(leaf)
    finally:os.close(fd)

def atomic(path,data):
    fd,tmp=tempfile.mkstemp(prefix='.ztarc-',dir=path.parent)
    try:
        with os.fdopen(fd,'wb') as f:f.write(data);f.flush();os.fsync(f.fileno())
        os.replace(tmp,path)
        d=os.open(path.parent,os.O_RDONLY|os.O_DIRECTORY)
        try:os.fsync(d)
        finally:os.close(d)
    finally:
        if os.path.exists(tmp):os.unlink(tmp)

def save(state):atomic(STATE/'state.json',(json.dumps(state,indent=2)+'\n').encode())

def replacement(unit,binary):
    # Preserve operational settings; refuse layouts that need a bespoke runbook.
    text=unit.decode();lines=text.splitlines();commands=[i for i,l in enumerate(lines) if l.startswith('ExecStart=')]
    if len(commands)!=1 or any(l.rstrip().endswith('\\') for l in lines):raise ValueError('nonstandard unit; manual migration required')
    old=lines[commands[0]]
    # Extra old flags must be accounted for explicitly in config first.
    import shlex
    tokens=shlex.split(old.split('=',1)[1])
    if len(tokens)!=3 or tokens[1] not in ('--config-file','-config-file') or tokens[2]!=str(CONFIG):
        raise ValueError('unit must use only --config-file /etc/ztarc/connector.json; preserve extra settings in config or use a reviewed runbook')
    if any(l.startswith(('ExecStartPre=','ExecStartPost=','ExecStop=','Environment=','EnvironmentFile=')) for l in lines):
        raise ValueError('unit contains hooks or environment; requires a reviewed migration runbook')
    lines[commands[0]]='ExecStart='+str(binary)+' up site --config-file '+str(CONFIG)
    return MARKER+'\n'.join(l for l in lines if l!=MARKER.strip())+'\n',tokens[0]

def stopped():
    run('systemctl','stop','ztarc-connector')
    active=run('systemctl','is-active','ztarc-connector',check=False).stdout.strip()
    pid=run('systemctl','show','ztarc-connector','--property=MainPID','--value').stdout.strip()
    if active in ('active','activating','deactivating') or pid!='0':raise ValueError('old connector did not stop; refusing second runtime')
    for process in pathlib.Path('/proc').glob('[0-9]*/cmdline'):
        try:argv=process.read_bytes().split(b'\0')
        except (OSError,PermissionError):continue
        if str(CONFIG).encode() in argv:raise ValueError('another process references the connector config; refusing duplicate runtime')

def health(path):
    read_regular(path,private=True)
    if not os.access(path,os.X_OK):raise ValueError('health-check must be executable')
    run(str(path))

def prepare_dir(path):
    # Called only on fixed root-controlled paths, never a supplied state path.
    current=pathlib.Path('/')
    for part in path.parts[1:]:
        current=current/part
        try:current.mkdir(mode=0o700)
        except FileExistsError:pass
        st=current.lstat()
        if not stat.S_ISDIR(st.st_mode) or st.st_uid!=0 or st.st_mode&0o022:raise ValueError('unsafe migration directory')
    if path.stat().st_mode&0o077:raise ValueError('migration state directory must be root-only')

def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('action',choices=['plan','apply','resume','rollback'])
    p.add_argument('--binary',type=pathlib.Path,default=pathlib.Path('/usr/local/bin/ztarc-cli'))
    p.add_argument('--sha256',required=True)
    p.add_argument('--health-check',type=pathlib.Path,required=True,help='Root-only executable that tests site identity and resource dataplane; exit 0 only on success')
    a=p.parse_args()
    if os.geteuid()!=0:raise ValueError('migration requires root')
    if a.binary!=pathlib.Path('/usr/local/bin/ztarc-cli'):raise ValueError('binary must be /usr/local/bin/ztarc-cli')
    if a.action!='rollback':
        if hashlib.sha256(read_regular(a.binary,max_size=256*1024*1024)).hexdigest()!=a.sha256:raise ValueError('candidate binary hash mismatch')
        run(str(a.binary),'config','validate-site',str(CONFIG))
    read_regular(CONFIG,private=True)
    fragment=run('systemctl','show','ztarc-connector','--property=FragmentPath','--value').stdout.strip()
    dropins=run('systemctl','show','ztarc-connector','--property=DropInPaths','--value').stdout.strip()
    if fragment!=str(UNIT) or dropins:raise ValueError('systemd fragment/drop-ins differ from the standard unit; use a reviewed runbook')
    read_regular(a.health_check,private=True)
    if a.action=='plan':
        new,old=replacement(read_regular(UNIT),a.binary)
        read_regular(pathlib.Path(old),max_size=256*1024*1024);print('Validated standard systemd cutover; site credentials and config remain unchanged. No tunnel started.');return
    prepare_dir(pathlib.Path('/run/ztarc-cli-install'))
    host_lock=open('/run/ztarc-cli-install/lock','a')
    fcntl.flock(host_lock,fcntl.LOCK_SH|fcntl.LOCK_NB)
    prepare_dir(STATE)
    with (STATE/'lock').open('a') as lock:
        fcntl.flock(lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
        journal=STATE/'state.json'
        state=json.loads(read_regular(journal,True)) if journal.exists() else None
        if a.action=='apply':
            if state and state['phase']=='complete':print('Already migrated.');return
            if state:raise ValueError('migration already started; use resume or rollback')
            original=read_regular(UNIT);new,old=replacement(original,a.binary)
            old_binary=pathlib.Path(old);read_regular(old_binary,max_size=256*1024*1024)
            if old_binary==a.binary:raise ValueError('old binary must remain available separately')
            atomic(STATE/'unit.backup',original);atomic(STATE/'config.backup',read_regular(CONFIG,True))
            atomic(STATE/'unit.candidate',new.encode())
            state={'phase':'prepared','binarySha256':a.sha256,'configSha256':hashlib.sha256(read_regular(CONFIG,True)).hexdigest(),'oldBinary':old,'oldBinarySha256':hashlib.sha256(read_regular(old_binary,max_size=256*1024*1024)).hexdigest()}
            save(state)
        if not state:raise ValueError('no migration journal to recover')
        if state['binarySha256']!=a.sha256:raise ValueError('candidate differs from journal')
        if hashlib.sha256(read_regular(CONFIG,True)).hexdigest()!=state['configSha256']:raise ValueError('config changed during migration; manual recovery required')
        if a.action=='resume' and state['phase']=='complete':health(a.health_check);print('Already complete; dataplane rechecked.');return
        if a.action=='resume' and state['phase'] in ('rolled_back','rollback_failed'):raise ValueError('migration was rolled back; inspect journal before a new migration')
        try:
            if a.action=='rollback':
                old=pathlib.Path(state['oldBinary'])
                if hashlib.sha256(read_regular(old,max_size=256*1024*1024)).hexdigest()!=state['oldBinarySha256']:raise ValueError('old binary differs from backup inventory')
                state['phase']='rollback_stopping';save(state);stopped()
                atomic(UNIT,read_regular(STATE/'unit.backup',True));run('systemctl','daemon-reload');fcntl.flock(host_lock,fcntl.LOCK_SH);run('systemctl','start','ztarc-connector');health(a.health_check)
                state['phase']='rolled_back';save(state);print('Rollback passed dataplane check.');return
            state['phase']='stopping';save(state);stopped()
            state['phase']='stopped';save(state)
            atomic(UNIT,read_regular(STATE/'unit.candidate',True));run('systemctl','daemon-reload')
            state['phase']='starting';save(state);fcntl.flock(host_lock,fcntl.LOCK_SH);run('systemctl','start','ztarc-connector');health(a.health_check)
            state['phase']='complete';save(state);print('Migration passed dataplane check; retain backup and old binary.')
        except Exception:
            state['phase']='rollback_failed' if a.action=='rollback' else 'needs_recovery';save(state)
            raise ValueError('migration failed; journal retained. Inspect host and use explicit rollback; no automatic runtime switching loop')
if __name__=='__main__':
    try:main()
    except (ValueError,OSError,subprocess.CalledProcessError) as e:sys.exit(str(e) if not isinstance(e,subprocess.CalledProcessError) else 'host operation failed; inspect journal and systemd locally')
