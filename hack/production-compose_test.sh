#!/usr/bin/env bash
set -euo pipefail
repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo"
python3 - <<'PY'
import json
import os
import pathlib
import re
import subprocess
import tempfile

# These are the existing API/Notification consumers' inputs. Values are fixtures,
# and assertions never print the resolved config or any caller credentials.
shared = {
    'ATHENA_LOGFORMAT': 'text',
    'ATHENA_LOG_FORMAT_ENABLE_FULL_TIMESTAMP': '1',
    'ATHENA_LOG_FORMAT_TIMESTAMP': '2006-01-02',
    'FORCE_LOG_COLORS': '1',
    'HTTP_PROXY': 'http://fixture-proxy:8080',
    'HTTPS_PROXY': 'http://fixture-proxy:8080',
    'NO_PROXY': 'postgres,redis',
    'ALL_PROXY': 'socks5://fixture-proxy:1080',
    'http_proxy': 'http://fixture-proxy:8080',
    'https_proxy': 'http://fixture-proxy:8080',
    'no_proxy': 'postgres,redis',
    'all_proxy': 'socks5://fixture-proxy:1080',
}
api_only = {
    'ETHERSCAN_GATEWAY_IPS': '192.0.2.1,192.0.2.2',
    'ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN': 'fixture-gateway-token',
    'ATHENA_ETHERSCAN_MANAGER_API_KEYS': 'fixture-key-one,fixture-key-two',
    'ATHENA_ETHERSCAN_GATEWAY_PROBE_QUERY_ADDRESS': '0x0000000000000000000000000000000000000001',
    'ATHENA_API_CONTENT_TYPES': 'application/json;application/cbor',
    'ATHENA_SERVER_OTLP_ADDRESS': 'collector:4317',
    'ATHENA_SERVER_OTLP_INSECURE': 'false',
    'ATHENA_SERVER_OTLP_HEADERS': 'authorization=fixture-trace-token',
    'ATHENA_SERVER_OTLP_ATTRS': 'deployment:acceptance,region:local',
    'ATHENA_SERVER_X_FRAME_OPTIONS': 'deny',
    'ATHENA_SERVER_CONTENT_SECURITY_POLICY': "frame-ancestors https://portal.example;",
    'ATHENA_SERVER_CONNECTION_STATUS_CACHE_EXPIRATION': '20m',
    'ATHENA_DEFAULT_CACHE_EXPIRATION': '12h',
    'REDISDB': '4',
    'REDIS_USERNAME': 'fixture-cache-user',
    'REDIS_COMPRESSION': 'none',
    'REDIS_RETRY_COUNT': '2',
    'REDIS_CREDS_DIR_PATH': '/fixture/cache-credentials',
    'REDIS_SENTINEL_USERNAME': 'fixture-sentinel-user',
    'REDIS_SENTINEL_PASSWORD': 'fixture-sentinel-password',
    'ATHENA_MAX_COOKIE_NUMBER': '12',
    'ATHENA_ADDITIONAL_URLS': 'https://portal.example,https://member.example',
    'ATHENA_SESSION_DURATION': '8h',
    'ATHENA_HELP_CHAT_URL': 'https://support.example',
    'ATHENA_HELP_CHAT_TEXT': 'Support',
    'ATHENA_STATUS_BADGE_ENABLED': 'true',
    'ATHENA_STATUS_BADGE_ROOT_URL': 'https://badges.example',
    'ATHENA_ANONYMOUS_USER_ENABLED': 'true',
    'ATHENA_UI_CSS_URL': 'https://static.example/site.css',
    'ATHENA_UI_BANNER_CONTENT': 'Maintenance notice',
    'ATHENA_UI_BANNER_PERMANENT': 'true',
    'ATHENA_UI_BANNER_POSITION': 'top',
    'ATHENA_UI_BANNER_URL': 'https://status.example',
}
for arch in ['DARWIN_AMD64', 'DARWIN_ARM64', 'WINDOWS_AMD64', 'LINUX_AMD64', 'LINUX_ARM64', 'LINUX_PPC64LE', 'LINUX_S390X']:
    api_only['ATHENA_HELP_DOWNLOAD_' + arch] = 'https://download.example/' + arch
notification_only = {
    'ATHENA_NOTIFICATION_TELEGRAM_API_URL': 'http://telegram-fixture:9000',
    'ATHENA_NOTIFICATION_TELEGRAM_TIMEOUT_SECONDS': '15',
    'ATHENA_NOTIFICATION_TELEGRAM_BOT_NAME': 'Fixture bot',
    'ATHENA_NOTIFICATION_TELEGRAM_BOT_SHORT_DESCRIPTION': 'Fixture notifications',
    'ATHENA_NOTIFICATION_TELEGRAM_BOT_DESCRIPTION': 'Fixture description',
    'ATHENA_NOTIFICATION_WORKER_CONCURRENCY': '3',
}
with tempfile.TemporaryDirectory(prefix='athena-production-compose-test-') as temp:
    envfile = pathlib.Path(temp) / 'compose.env'
    envfile.write_text('')
    config_env = dict(os.environ)
    globally_required = set(re.findall(r'\$\{([A-Z0-9_]+):\?', pathlib.Path('docker-compose.prod.yml').read_text()))
    for key in globally_required:
        config_env[key] = 'fixture-value'
    config_env['ATHENA_COMPOSE_ENV_FILE'] = str(envfile)
    config_env.update(shared | api_only | notification_only)
    def resolve(values):
        result = subprocess.run(['docker', 'compose', '-f', 'docker-compose.prod.yml', '--env-file', str(envfile), '--profile', 'tools', 'config', '--format', 'json'], env=values, capture_output=True, text=True, check=True)
        return json.loads(result.stdout)['services']
    services = resolve(config_env)
    retired = {'athena-sports-live', 'athena-sports-history', 'athena-worm-markets'}
    assert not retired.intersection(services), 'retired services still deploy'
    labelled = {name for name, s in services.items() if s.get('labels', {}).get('io.athena.account-state.consumer') == 'true'}
    assert labelled == {'athena-server', 'athena-notification', 'athena-trader-sync', 'athena-worm-trading', 'athena-operation-log'}, labelled
    for name in ['athena-worm-trading', 'athena-wallet', 'athena-notification']:
        assert name in services, name + ' was removed'
    assert 'athena-operation-log' in services, 'operation-log service is missing'
    operation_log = services['athena-operation-log']
    assert operation_log.get('expose') == ['8124'], operation_log.get('expose')
    assert not operation_log.get('ports'), 'operation-log must not publish a host port'
    assert operation_log.get('depends_on', {}).get('postgres', {}).get('condition') == 'service_healthy'
    assert operation_log.get('labels', {}).get('io.athena.account-state.consumer') == 'true'
    assert operation_log['environment'].get('ATHENA_OPERATION_LOG_POSTGRES_DSN') == operation_log['environment'].get('ATHENA_ACCOUNT_STATE_POSTGRES_DSN')
    assert operation_log['environment'].get('ATHENA_OPERATION_LOG_GRPC_TRANSPORT') == 'tls'
    assert 'ATHENA_OPERATION_LOG_CURSOR_HMAC_KEY_FILE' in operation_log['environment']
    assert 'ATHENA_OPERATION_LOG_INTERNAL_AUTH_TOKEN_FILE' in operation_log['environment']
    assert 'ATHENA_OPERATION_LOG_TLS_CERT_FILE' in operation_log['environment']
    assert 'ATHENA_OPERATION_LOG_TLS_KEY_FILE' in operation_log['environment']
    assert 'ATHENA_OPERATION_LOG_TLS_CA_FILE' in operation_log['environment']
    assert 'athena-operation-log-migrate' in services
    assert 'tools' in services['athena-operation-log-migrate'].get('profiles', [])
    api_environment = services['athena-server']['environment']
    assert 'ATHENA_OPERATION_LOG_INTERNAL_AUTH_TOKEN_FILE' in api_environment
    assert 'ATHENA_OPERATION_LOG_TLS_CA_FILE' in api_environment
    assert 'ATHENA_OPERATION_LOG_CURSOR_HMAC_KEY_FILE' not in api_environment
    for name, service in services.items():
        assert not retired.intersection(service.get('depends_on', {})), name + ' still depends on a retired service'
        assert not any(key.startswith(('ATHENA_SPORTS_LIVE_', 'ATHENA_SPORTS_HISTORY_')) for key in service.get('environment', {})), name + ' still configures Sports'
        assert not any(key.startswith('ATHENA_WORM_MARKETS_') for key in service.get('environment', {})), name + ' still configures Worm Markets'
    for service, values in [('athena-server', shared | api_only), ('athena-notification', shared | notification_only)]:
        for key, expected in values.items():
            assert services[service]['environment'].get(key) == expected, service + ' lost ' + key
    for service in ['athena-notification', 'athena-trader-sync', 'athena-account-state-migrate']:
        for key in api_only:
            assert key not in services[service]['environment'], service + ' unexpectedly received ' + key
    for service in ['athena-server', 'athena-trader-sync', 'athena-account-state-migrate']:
        for key in notification_only:
            assert key not in services[service]['environment'], service + ' unexpectedly received ' + key
    # Unset defaults differ from explicit empty, notably content types (AllowEmpty).
    defaults = {
        'ATHENA_LOGFORMAT': 'json',
        'ATHENA_API_CONTENT_TYPES': 'application/json',
        'ATHENA_SERVER_OTLP_INSECURE': 'true',
        'ATHENA_SERVER_X_FRAME_OPTIONS': 'sameorigin',
        'ATHENA_SERVER_CONTENT_SECURITY_POLICY': "frame-ancestors 'self';",
    }
    for key in defaults:
        config_env.pop(key, None)
    services = resolve(config_env)
    for key, expected in defaults.items():
        assert services['athena-server']['environment'].get(key) == expected, 'unset default lost for ' + key
    # Etherscan manager separately requires its credentials in the full stack.
    # Preserve that existing constraint while testing all permitted empties.
    for key in (shared | api_only | notification_only).keys() - globally_required:
        config_env[key] = ''
    services = resolve(config_env)
    for service, values in [('athena-server', shared | api_only), ('athena-notification', shared | notification_only)]:
        for key in values.keys() - globally_required:
            assert services[service]['environment'].get(key) == '', service + ' discarded explicit empty ' + key
    print('real Compose: API/Notification configured values, unset defaults, explicit empty values, and secret isolation passed')
PY
