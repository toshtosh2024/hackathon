#!/usr/bin/env bash
set -euo pipefail

PROJECT_ID="${PROJECT_ID:-term9-toshiie-shiomi}"
REGION="${REGION:-asia-northeast1}"
SERVICE="${SERVICE:-next-market}"
INSTANCE="${INSTANCE:-next-market-mysql}"

CONNECTION_NAME="$(gcloud sql instances describe "${INSTANCE}" \
  --project "${PROJECT_ID}" \
  --format='value(connectionName)')"

if [[ -z "${CONNECTION_NAME}" ]]; then
  echo "Cloud SQL instance not found: ${INSTANCE}" >&2
  exit 1
fi

gcloud run services update "${SERVICE}" \
  --project "${PROJECT_ID}" \
  --region "${REGION}" \
  --min-instances 1 \
  --add-cloudsql-instances "${CONNECTION_NAME}" \
  --set-env-vars "INSTANCE_UNIX_SOCKET=/cloudsql/${CONNECTION_NAME}" \
  --quiet

gcloud run services describe "${SERVICE}" \
  --project "${PROJECT_ID}" \
  --region "${REGION}" \
  --format='table(metadata.name,status.url,spec.template.metadata.annotations.autoscaling.knative.dev/minScale)'
