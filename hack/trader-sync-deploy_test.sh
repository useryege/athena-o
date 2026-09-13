#!/usr/bin/env bash
set -euo pipefail
repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo"
bash hack/production-compose_test.sh
python3 - <<'PY'
import json, os, pathlib, re, subprocess, tempfile
root = pathlib.Path.cwd()
with tempfile.TemporaryDirectory(prefix='athena-ts-deploy-test-') as temp:
    path = pathlib.Path(temp)
    envfile = path / 'compose.env'
    envfile.write_text('')
    config_env = dict(os.environ)
    for key in re.findall(r'\$\{([A-Z0-9_]+):\?', (root / 'docker-compose.prod.yml').read_text()):
        config_env[key] = 'fixture-value'
    config_env.update(ATHENA_COMPOSE_ENV_FILE=str(envfile), TRADER_SYNC_IMAGE='trader-sync:compose-test', ATHENA_SERVER_BASEHREF='/athena/', ATHENA_SERVER_ROOTPATH='/athena', ATHENA_NOTIFICATION_TELEGRAM_API_URL='http://fixture:9000')
    result = subprocess.run(['docker', 'compose', '-f', 'docker-compose.prod.yml', '--env-file', str(envfile), '--profile', 'tools', 'config', '--format', 'json'], env=config_env, capture_output=True, text=True, check=True)
    config = json.loads(result.stdout)
    services = config['services']
    ts, tool, api, notification = [services[x] for x in ['athena-trader-sync','athena-account-state-migrate','athena-server','athena-notification']]
    assert api['environment']['ATHENA_SERVER_BASEHREF'] == '/athena/'
    assert api['environment']['ATHENA_SERVER_ROOTPATH'] == '/athena'
    assert notification['environment']['ATHENA_NOTIFICATION_TELEGRAM_API_URL'] == 'http://fixture:9000'
    assert ts['image'] == tool['image'] == 'trader-sync:compose-test'
    assert not ts.get('ports') and set(ts['depends_on']) == {'postgres'}
    assert ts['stop_grace_period'] == '40s'
    assert 'athena-trader-sync' not in api.get('depends_on', {})
    assert ts['environment']['ATHENA_TRADER_SYNC_LISTEN_ADDRESS'] == '0.0.0.0:8122'
    assert api['environment']['ATHENA_TRADER_SYNC_SERVER_ADDRESS'] == 'athena-trader-sync:8122'
    for service in [ts, api]:
        assert service['environment']['ATHENA_TRADER_SYNC_GRPC_TRANSPORT'] == 'tls'
        assert service['environment']['ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN_FILE'] == '/run/secrets/trader-sync-token'
        assert service['environment']['ATHENA_TRADER_SYNC_TLS_CA_FILE'] == '/run/secrets/trader-sync-ca'
        assert service['environment']['ATHENA_TRADER_SYNC_TLS_SERVER_NAME'] == 'athena-trader-sync'
    assert ts['healthcheck']['test'] == ['CMD','athena-trader-sync','health','--target','127.0.0.1:8122']
    assert tool['entrypoint'][-1] == 'athena-account-state-migrate' and tool['command'] == ['verify']
    assert set(tool['environment']) == {'ATHENA_ACCOUNT_STATE_POSTGRES_DSN'}
    assert {x['environment']['ATHENA_ACCOUNT_STATE_POSTGRES_DSN'] for x in [ts, tool, api, notification]} == {'fixture-value'}
    for service in [ts, tool, api, notification]:
        assert 'ATHENA_SERVER_POSTGRES_DSN' not in service['environment']
    for key in ts['environment']:
        assert key.startswith('ATHENA_TRADER_SYNC_') or key in {'ATHENA_ACCOUNT_STATE_POSTGRES_DSN','ATHENA_URL','ATHENA_LOGLEVEL','ATHENA_GRPC_MAX_SIZE_MB'}
    for service in [api, notification]:
        for key in ['ATHENA_TRADER_SYNC_HTTP_URL','ATHENA_TRADER_SYNC_WSS_URL','ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY','ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY_FILE']:
            assert key not in service['environment']
    assert all(not key.startswith('ATHENA_TRADER_SYNC_') for key in notification['environment'])
    print('real Compose configuration and process secret boundaries passed')
    bindir = path / 'bin'; bindir.mkdir()
    recorder = bindir / 'docker'
    recorder.write_text('#!/usr/bin/python3\nimport json,os,sys\nwith open(os.environ["BUILD_ARGV"],"a") as f: f.write(json.dumps(sys.argv[1:])+"\\n")\n')
    recorder.chmod(0o755)
    build_env = dict(os.environ, PATH=str(bindir)+':'+os.environ['PATH'], BUILD_ARGV=str(path/'argv'), TRADER_SYNC_IMAGE="image'$(touch unexpected-injection):test")
    result = subprocess.run(['make','--no-print-directory','build-service-image','SERVICE=trader-sync'], env=build_env, capture_output=True, text=True)
    assert result.returncode == 0, result.stderr
    calls = [json.loads(line) for line in (path/'argv').read_text().splitlines()]
    assert len(calls) == 1 and calls[0][0] == 'build', calls
    argv = calls[0]
    assert argv[argv.index('-f')+1] == 'deploy/trader-sync/Dockerfile'
    assert argv[argv.index('-t')+1] == build_env['TRADER_SYNC_IMAGE']
    assert not pathlib.Path('unexpected-injection').exists()
    result = subprocess.run(['make','--no-print-directory','build-service-image','SERVICE=invalid'], env=build_env, capture_output=True)
    assert result.returncode != 0
    assert len((path/'argv').read_text().splitlines()) == 1
    print('independent image build selection and literal argv passed')
PY
TEST_TRADER_SYNC_ONLY=yes bash hack/deploy-scripts_test.sh
