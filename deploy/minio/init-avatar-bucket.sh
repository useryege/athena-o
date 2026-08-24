#!/usr/bin/env bash

set -euo pipefail

required_variables=(
  MINIO_ENDPOINT
  MINIO_ROOT_USER
  MINIO_ROOT_PASSWORD
  ATHENA_ACCOUNT_AVATAR_S3_BUCKET
  ATHENA_ACCOUNT_AVATAR_S3_ACCESS_KEY_ID
  ATHENA_ACCOUNT_AVATAR_S3_SECRET_ACCESS_KEY
)

for variable_name in "${required_variables[@]}"; do
  if [[ -z "${!variable_name:-}" ]]; then
    printf '%s is required\n' "${variable_name}" >&2
    exit 1
  fi
done

bucket="${ATHENA_ACCOUNT_AVATAR_S3_BUCKET}"
if [[ ${#bucket} -lt 3 || ${#bucket} -gt 63 ||
  ! "${bucket}" =~ ^[a-z0-9][a-z0-9.-]*[a-z0-9]$ ||
  "${bucket}" == *..* || "${bucket}" == *.-* || "${bucket}" == *-.* ]]; then
  printf 'ATHENA_ACCOUNT_AVATAR_S3_BUCKET is not a valid S3 bucket name\n' >&2
  exit 1
fi

connected=false
for attempt in {1..60}; do
  if mc alias set athena "${MINIO_ENDPOINT}" "${MINIO_ROOT_USER}" "${MINIO_ROOT_PASSWORD}" >/dev/null 2>&1 &&
    mc ready athena >/dev/null 2>&1; then
    connected=true
    break
  fi
  sleep 1
done
if [[ "${connected}" != "true" ]]; then
  printf 'MinIO did not become ready at %s\n' "${MINIO_ENDPOINT}" >&2
  exit 1
fi

policy_file="$(mktemp -t athena-avatar-policy.XXXXXX.json)"
trap 'rm -f -- "${policy_file}"' EXIT
cat >"${policy_file}" <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["s3:GetBucketLocation", "s3:ListBucket"],
      "Resource": ["arn:aws:s3:::${bucket}"]
    },
    {
      "Effect": "Allow",
      "Action": ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"],
      "Resource": ["arn:aws:s3:::${bucket}/*"]
    }
  ]
}
POLICY

mc mb --ignore-existing "athena/${bucket}" >/dev/null
mc anonymous set none "athena/${bucket}" >/dev/null
# The pinned server treats policy creation and user creation as updates when the
# names already exist. The pinned mc also treats an already-attached policy as
# a successful no-op, so hot deploys can rerun this initializer safely.
mc admin policy create athena athena-account-avatars "${policy_file}" >/dev/null
mc admin user add athena \
  "${ATHENA_ACCOUNT_AVATAR_S3_ACCESS_KEY_ID}" \
  "${ATHENA_ACCOUNT_AVATAR_S3_SECRET_ACCESS_KEY}" >/dev/null
mc admin policy attach athena athena-account-avatars \
  --user "${ATHENA_ACCOUNT_AVATAR_S3_ACCESS_KEY_ID}" >/dev/null

printf 'initialized private MinIO bucket %s and application policy\n' "${bucket}"
