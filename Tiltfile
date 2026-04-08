load('ext://restart_process', 'docker_build_with_restart')
load('ext://uibutton', 'cmd_button', 'location')

# add ui button in web ui to run make codegen-local (top nav)
cmd_button(
    'make codegen-local',
    argv=['sh', '-c', 'make codegen-local'],
    location=location.NAV,
    icon_name='terminal',
    text='make codegen-local',
)

cmd_button(
    'make test-local',
    argv=['sh', '-c', 'make test-local'],
    location=location.NAV,
    icon_name='science',
    text='make test-local',
)

# add ui button in web ui to run make codegen-local (top nav)
cmd_button(
    'make cli-local',
    argv=['sh', '-c', 'make cli-local'],
    location=location.NAV,
    icon_name='terminal',
    text='make cli-local',
)

# detect cluster architecture for build
cluster_version = decode_yaml(local('kubectl version -o yaml'))
platform = cluster_version['serverVersion']['platform']
arch = platform.split('/')[1]

# build the athena binary on code changes
code_deps = [
    'applicationset',
    'cmd',
    'cmpserver',
    'commitserver',
    'common',
    'controller',
    'notification-controller',
    'pkg',
    'reposerver',
    'server',
    'util',
    'go.mod',
    'go.sum',
]
local_resource(
    'build',
    'CGO_ENABLED=0 GOOS=linux GOARCH=' + arch + ' go build -gcflags="all=-N -l" -mod=readonly -o .tilt-bin/athena_linux cmd/main.go',
    deps = code_deps,
    allow_parallel=True,
)

# deploy the athena manifests
k8s_yaml(kustomize('manifests/dev-tilt'))

# build dev image
docker_build_with_restart(
    'athena', 
    context='.',
    dockerfile='Dockerfile.tilt',
    entrypoint=[
        "/usr/bin/tini",
        "-s",
        "--",
        "dlv",
        "exec",
        "--continue",
        "--accept-multiclient",
        "--headless",
        "--listen=:2345",
        "--api-version=2"
    ],
    platform=platform,
    live_update=[
        sync('.tilt-bin/athena_linux', '/usr/local/bin/athena'),
    ],
    only=[
        '.tilt-bin',
        'hack',
        'entrypoint.sh',
    ],
    restart_file='/tilt/.restart-proc'
)

# build image for athena-cli jobs
docker_build(
    'athena-job', 
    context='.',
    dockerfile='Dockerfile.tilt',
    platform=platform,
    only=[
        '.tilt-bin',
        'hack',
        'entrypoint.sh',
    ]
)

# track athena-server resources and port forward
k8s_resource(
    workload='athena-server',
    objects=[
        'athena-server:serviceaccount',
        'athena-server:role',
        'athena-server:rolebinding',
        'athena-cm:configmap',
        'athena-cmd-params-cm:configmap',
        'athena-gpg-keys-cm:configmap',
        'athena-rbac-cm:configmap',
        'athena-ssh-known-hosts-cm:configmap',
        'athena-tls-certs-cm:configmap',
        'athena-secret:secret',
        'athena-server-network-policy:networkpolicy',
        'athena-server:clusterrolebinding',
        'athena-server:clusterrole',
    ],
    port_forwards=[
        '8080:8080',
        '9345:2345',
        '8083:8083'
    ],
    resource_deps=['build']
)

# track crds
k8s_resource(
    new_name='cluster-resources',
    objects=[
        'applications.argoproj.io:customresourcedefinition',
        'applicationsets.argoproj.io:customresourcedefinition',
        'appprojects.argoproj.io:customresourcedefinition',
        'athena:namespace'
    ]
)

# track athena-repo-server resources and port forward
k8s_resource(
    workload='athena-repo-server',
    objects=[
        'athena-repo-server:serviceaccount',
        'athena-repo-server-network-policy:networkpolicy',
    ],
    port_forwards=[
        '8081:8081',
        '9346:2345',
        '8084:8084'
    ],
    resource_deps=['build']
)

# track athena-redis resources and port forward
k8s_resource(
    workload='athena-redis',
    objects=[
        'athena-redis:serviceaccount',
        'athena-redis:role',
        'athena-redis:rolebinding',
        'athena-redis-network-policy:networkpolicy',
    ],
    port_forwards=[
        '6379:6379',
    ],
    resource_deps=['build']
)

# track athena-applicationset-controller resources
k8s_resource(
    workload='athena-applicationset-controller',
    objects=[
        'athena-applicationset-controller:serviceaccount',
        'athena-applicationset-controller-network-policy:networkpolicy',
        'athena-applicationset-controller:role',
        'athena-applicationset-controller:rolebinding',
        'athena-applicationset-controller:clusterrolebinding',
        'athena-applicationset-controller:clusterrole',
    ],
    port_forwards=[
        '9347:2345',
        '8085:8080',
        '7000:7000'
    ],
    resource_deps=['build']
)

# track athena-application-controller resources
k8s_resource(
    workload='athena-application-controller',
    objects=[
        'athena-application-controller:serviceaccount',
        'athena-application-controller-network-policy:networkpolicy',
        'athena-application-controller:role',
        'athena-application-controller:rolebinding',
        'athena-application-controller:clusterrolebinding',
        'athena-application-controller:clusterrole',
    ],
    port_forwards=[
        '9348:2345',
        '8086:8082',
    ],
    resource_deps=['build']
)

# track athena-notifications-controller resources
k8s_resource(
    workload='athena-notifications-controller',
    objects=[
        'athena-notifications-controller:serviceaccount',
        'athena-notifications-controller-network-policy:networkpolicy',
        'athena-notifications-controller:role',
        'athena-notifications-controller:rolebinding',
        'athena-notifications-cm:configmap',
        'athena-notifications-secret:secret',
    ],
    port_forwards=[
        '9349:2345',
        '8087:9001',
    ],
    resource_deps=['build']
)

# track athena-dex-server resources
k8s_resource(
    workload='athena-dex-server',
    objects=[
        'athena-dex-server:serviceaccount',
        'athena-dex-server-network-policy:networkpolicy',
        'athena-dex-server:role',
        'athena-dex-server:rolebinding',
    ],
    resource_deps=['build']
)

# track athena-commit-server resources
k8s_resource(
    workload='athena-commit-server',
    objects=[
        'athena-commit-server:serviceaccount',
        'athena-commit-server-network-policy:networkpolicy',
    ],
    port_forwards=[
        '9350:2345',
        '8088:8087',
        '8089:8086',
    ],
    resource_deps=['build']
)

# ui dependencies
local_resource(
    'node-modules',
    'yarn',
    dir='ui',
    deps = [
        'ui/package.json',
        'ui/yarn.lock',
    ],
    allow_parallel=True,
)

# docker for ui
docker_build(
    'athena-ui',
    context='.',
    dockerfile='Dockerfile.ui.tilt',
    entrypoint=['sh', '-c', 'cd /app/ui && yarn start'], 
    only=['ui'],
    live_update=[
        sync('ui', '/app/ui'),
        run('sh -c "cd /app/ui && yarn install"', trigger=['/app/ui/package.json', '/app/ui/yarn.lock']),
    ],
)

# track athena-ui resources and port forward
k8s_resource(
    workload='athena-ui',
    port_forwards=[
        '4000:4000',
    ],
    resource_deps=['node-modules'],
)

# linting
local_resource(
    'lint',
    'make lint-local',
    deps = code_deps,
    allow_parallel=True,
    resource_deps=['vendor']
)

local_resource(
    'lint-ui',
    'make lint-ui-local',
    deps = [
        'ui',
    ],
    allow_parallel=True,
    resource_deps=['node-modules'],
)

local_resource(
    'vendor',
    'go mod vendor',
    deps = [
        'go.mod',
        'go.sum',
    ],
    allow_parallel=True,
)

