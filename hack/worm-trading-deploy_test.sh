#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."
python3 - <<'PY'
import json, os, pathlib, re, subprocess, tempfile
root=pathlib.Path.cwd()
with tempfile.TemporaryDirectory(prefix='worm-trading-deploy-') as temp:
    path=pathlib.Path(temp); envfile=path/'compose.env'; envfile.write_text('')
    env=dict(os.environ)
    for key in re.findall(r'\$\{([A-Z0-9_]+):\?',(root/'docker-compose.prod.yml').read_text()): env[key]='fixture-value'
    env.update(ATHENA_COMPOSE_ENV_FILE=str(envfile), WORM_TRADING_IMAGE='worm-trading:fixture')
    result=subprocess.run(['docker','compose','-f','docker-compose.prod.yml','--env-file',str(envfile),'--profile','tools','config','--format','json'],env=env,capture_output=True,text=True,check=True)
    services=json.loads(result.stdout)['services']; trading=services['athena-worm-trading']; migrate=services['athena-worm-trading-migrate']
    assert trading['image']==migrate['image']=='worm-trading:fixture'
    assert trading['command']==['athena-worm-trading','--port','8090']
    assert trading['labels']['io.athena.account-state.consumer']=='true'
    assert trading['environment']['ATHENA_ACCOUNT_STATE_POSTGRES_DSN']=='fixture-value'
    assert migrate['entrypoint'][-1]=='athena-worm-trading-migrate'
    assert set(migrate['environment'])=={'ATHENA_WORM_TRADING_POSTGRES_DSN'}
    assert not any('MARKETS' in key for key in trading['environment'])
    bindir=path/'bin';bindir.mkdir();recorder=bindir/'docker'
    recorder.write_text('#!/usr/bin/python3\nimport json,os,sys\nwith open(os.environ["BUILD_ARGV"],"a") as f:f.write(json.dumps(sys.argv[1:])+"\\n")\n');recorder.chmod(0o755)
    env=dict(os.environ,PATH=str(bindir)+':'+os.environ['PATH'],BUILD_ARGV=str(path/'argv'),WORM_TRADING_IMAGE="image'$(touch unexpected-injection):test")
    result=subprocess.run(['make','--no-print-directory','build-service-image','SERVICE=worm-trading'],env=env,capture_output=True,text=True)
    assert result.returncode==0,result.stderr
    calls=[json.loads(line) for line in (path/'argv').read_text().splitlines()];assert len(calls)==1
    argv=calls[0];assert argv[argv.index('-f')+1]=='deploy/worm-trading/Dockerfile';assert argv[argv.index('-t')+1]==env['WORM_TRADING_IMAGE']
    assert not pathlib.Path('unexpected-injection').exists()
    default_env = dict(env)
    for key in ['WORM_TRADING_IMAGE', 'MAKEFLAGS', 'MFLAGS', 'MAKEOVERRIDES']:
        default_env.pop(key, None)
    result=subprocess.run(['make','--no-print-directory','build-service-image','SERVICE=worm-trading'],env=default_env,capture_output=True,text=True)
    assert result.returncode==0,result.stderr
    calls=[json.loads(line) for line in (path/'argv').read_text().splitlines()];assert len(calls)==2
    default_argv=calls[1]
    assert default_argv[default_argv.index('-f')+1]=='deploy/worm-trading/Dockerfile'
    assert default_argv[default_argv.index('-t')+1]=='athena-worm-trading:local', default_argv
    print('Trading independent Compose, default image tag, and literal custom build argv passed')
PY
