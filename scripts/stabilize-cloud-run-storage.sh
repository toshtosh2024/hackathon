#!/usr/bin/env bash
set -euo pipefail

PROJECT_ID="${PROJECT_ID:-term9-toshiie-shiomi}"
REGION="${REGION:-asia-northeast1}"
SERVICE="${SERVICE:-next-market}"
BUCKET="${BUCKET:-${PROJECT_ID}-next-market-uploads}"
RUNTIME_SERVICE_ACCOUNT="${RUNTIME_SERVICE_ACCOUNT:-next-market-run@${PROJECT_ID}.iam.gserviceaccount.com}"

gcloud storage buckets describe "gs://${BUCKET}" \
  --project "${PROJECT_ID}" >/dev/null

gcloud storage buckets add-iam-policy-binding "gs://${BUCKET}" \
  --project "${PROJECT_ID}" \
  --member "serviceAccount:${RUNTIME_SERVICE_ACCOUNT}" \
  --role roles/storage.objectCreator \
  --quiet

gcloud storage buckets add-iam-policy-binding "gs://${BUCKET}" \
  --project "${PROJECT_ID}" \
  --member allUsers \
  --role roles/storage.objectViewer \
  --quiet

gcloud run services update "${SERVICE}" \
  --project "${PROJECT_ID}" \
  --region "${REGION}" \
  --service-account "${RUNTIME_SERVICE_ACCOUNT}" \
  --update-env-vars "GCS_BUCKET=${BUCKET}" \
  --quiet

gcloud run services describe "${SERVICE}" \
  --project "${PROJECT_ID}" \
  --region "${REGION}" \
  --format='yaml(status.url,spec.template.spec.serviceAccountName,spec.template.spec.containers[0].env)'
